# CNVD 漏洞报送：O-RAN SC o-du-l2 订阅待响应槽位越界写（CWE-787）

编制日期：2026-11-24
状态：源码定位 + 隔离程序实测复现（普通构建 + AddressSanitizer 双证据）；尚未向 CNVD 提交；无 CNVD 编号、无 CVE 编号。

## 一、基本信息（对应 CNVD 表单字段）

| 字段 | 内容 |
|---|---|
| 漏洞名称 | O-RAN SC o-du-l2 订阅请求错误清理路径越界写（待响应槽位数组越界 memset） |
| 厂商 / 维护组织 | O-RAN Software Community（O-RAN SC） |
| 产品 | o-du-l2（O-RAN 分布式单元 L2 层，含 E2AP 消息处理） |
| 测试版本 | 源码提交 `2278be373834a0a198e0b5dac94cc66397d88421` |
| 漏洞类型 | 越界写入（CWE-787）；伴随计数字段生命周期缺陷（CWE-664 / CWE-665 类） |
| 危害等级 | 建议“中”，以审核认定为准。可造成相邻结构体成员被清零（状态破坏）与进程崩溃；未证明可利用为代码执行 |
| 漏洞位置 | `src/du_app/du_e2ap_mgr.h:39`（`MAX_PENDING_SUBSCRIPTION_RSP = 5`）、`src/du_app/du_e2ap_mgr.h:445`（`RanFunction.pendingSubsRspInfo[5]`）、`src/du_app/du_e2ap_msg_hdl.c:3520`（容量检查）、`src/du_app/du_e2ap_msg_hdl.c:3602-3609`（错误清理 memset） |
| 触发者 | RIC 侧（xApp 经 E2 接口）发送订阅请求的实体；无需并发 |
| 攻击向量 | 网络（AV:N），需具备向 O-DU 发送 E2AP RIC Subscription Request 的能力 |
| 源码仓库 | https://github.com/o-ran-sc/o-du-l2 |
| 首次发现日期 | 2026-10-27（静态审查）；2026-11-06（隔离实测复现） |
| 已分配编号 | 无 |
| 报告人 / 联系方式 | [由提交人填写] |

## 二、验证基线

- 环境：WSL Ubuntu，GCC 13.3，AddressSanitizer（libasan）
- 验证方式：隔离 C 程序复刻生产控制流（结构体布局镜像 `du_e2ap_mgr.h:445` 的 `RanFunction`，检查与错误路径 memset 按 `du_e2ap_msg_hdl.c:3520/3602-3609` 复刻），不依赖完整 O-DU 构建
- 说明：本文结论基于该固定提交的源码。未在完整 O-DU 运行实例上复现端到端崩溃，也未验证其他版本；提交前应核对上游是否已有修复。

## 三、漏洞描述（可粘贴至“漏洞描述”栏）

`RanFunction` 结构体中，用于记录“已发出、等待响应的订阅请求”的数组固定为 5 个元素：

```c
/* src/du_app/du_e2ap_mgr.h */
#define MAX_PENDING_SUBSCRIPTION_RSP 5     /* :39  */
...
PendingSubsRspInfo pendingSubsRspInfo[MAX_PENDING_SUBSCRIPTION_RSP];  /* :445 */
```

`procRicSubscriptionRequest()`（`src/du_app/du_e2ap_msg_hdl.c`）在处理订阅请求时，先做容量检查，满则置失败并跳出：

```c
/* 约 :3520 */
if (ranFuncDb->numPendingSubsRsp >= MAX_PENDING_SUBSCRIPTION_RSP)
{
    /* 置失败、goto 到统一错误清理 */
}
```

问题在于该函数结尾的统一错误清理路径对数组下标**不再做任何检查**，无条件执行：

```c
/* :3602-3609 统一错误清理 */
memset(&ranFuncDb->pendingSubsRspInfo[ranFuncDb->numPendingSubsRsp],
       0, sizeof(PendingSubsRspInfo));
```

当 `numPendingSubsRsp == 5`（容量已满）时，该语句对 `pendingSubsRspInfo[5]` 执行 64 字节清零——下标 5 已越过合法范围 0..4，写入的是数组之后的相邻内存。

加重因素（使触发条件可稳定达成）：

1. 成功路径执行 `numPendingSubsRsp++`（约 `:3596`），而响应完成路径 `procStatsRspForSubsReq()`（`src/du_app/du_e2ap_mgr.c:832-840`）**只清零槽位内容、从不递减计数**。因此计数只增不减，同一 RAN Function 累计处理 5 次订阅请求后即永久处于“满”状态。
2. 同样的缺陷模式也存在于修改请求路径：`du_e2ap_msg_hdl.c:10094-10116` 中 `numPendingSubsModRsp` 被直接用作 `pendingSubsModRspInfo` 的下标，连容量检查都没有。

触发路径（无需并发，顺序请求即可）：

1. RIC（xApp）向 O-DU 发送 E2AP RIC Subscription Request；
2. 同一 RAN Function 累计处理 5 次订阅请求（成功或失败均使计数只增不减）；
3. 第 6 个请求到达 → 容量检查失败 → 进入错误清理路径 → 对 `pendingSubsRspInfo[5]` 越界 memset。

## 四、证据（隔离实测，双构建）

普通构建：越界 memset 将紧邻结构体成员清零，布局检查确认写入目标即相邻成员地址：

```text
before: numPendingSubsRsp=5 sentinel=0x41424344 array_end=0x...13e4 sentinel=0x...13e4
after : ret=RFAILED, sentinel=0x00000000
layout check: memset target IS the adjacent member address
OBSERVED DAMAGE: adjacent struct member zeroed by the OOB memset
```

ASan 构建：报 heap-buffer-overflow，写入长度恰为一个 `PendingSubsRspInfo`（64 字节），位置在数组末尾 0 字节偏移：

```text
ERROR: AddressSanitizer: heap-buffer-overflow on address 0x513000000188
WRITE of size 64 at 0x513000000188 thread T0
    #0 ... in memset
    #1 ... in procRicSubsReq du04_oob_test.c:39
0x513000000188 is located 0 bytes after 328-byte region
SUMMARY: AddressSanitizer: heap-buffer-overflow ... in memset
```

| 材料 | 说明 |
|---|---|
| `du04-pending-slot-oob/du04_oob_test.c` | 复刻生产控制流的隔离测试程序（含结构体布局镜像、检查与错误路径） |
| `du04-pending-slot-oob/REPRODUCTION.md` | 构建命令、执行命令、上述两段原始输出 |
| `du04-pending-slot-oob/README.md` | 成因、触发路径、修复建议汇总 |

## 五、复现步骤

```sh
# 需要 GCC 与 libasan
cd du04-pending-slot-oob

# 1. 普通构建：观察相邻成员被清零
gcc -g -O0 -o du04_plain du04_oob_test.c
./du04_plain

# 2. ASan 构建：取得越界写报告
gcc -g -O0 -fsanitize=address -o du04_asan du04_oob_test.c
./du04_asan
```

预期结果：`du04_plain` 打印相邻哨兵成员由 `0x41424344` 变为 `0x00000000`，并输出布局一致性与破坏确认；`du04_asan` 报 `heap-buffer-overflow ... WRITE of size 64`。

## 六、影响与边界（如实说明）

已证实：
- 当待响应计数达到 5 后，错误清理路径对数组尾部之外写入 64 字节零值；
- ASan 明确判定为 heap-buffer-overflow，写入大小与结构体元素大小一致，非误报；
- 触发不需要并发、不需要竞态窗口，仅需顺序发送 6 个订阅请求。

未证实 / 不作宣称：
- 破坏的具体后果取决于 `pendingSubsRspInfo` 之后的结构体成员布局，本次隔离程序只证明了“写入越界且覆盖相邻成员”这一事实，未在完整 O-DU 上确认真实受损字段及其业务后果；
- 未证明可借此实现任意地址写、代码执行或持久化控制；破坏形式首先是控制面状态破坏与潜在进程崩溃；
- 未在真实 O-RAN 前传/管理面拓扑中验证触发所需权限；O-DU 侧 E2 接入通常要求先建立 SCTP 关联并处于已注册状态。

## 七、修复建议

不要把“有效待响应数量”和“槽位编号”混用为同一个计数器：

1. 为每个槽位增加占用标记，分配时遍历寻找空闲槽并**记录本次实际分配的槽位号**；
2. 清理时只清理本次确实分配的槽位，用记录下来的槽位号作下标，不再用计数作下标；
3. 容量满时直接构造“资源不足”响应返回，不访问数组；
4. 成功、失败、超时、取消四条路径都释放槽位（置空占用标记），并分别维护“有效数量”；
5. 修改请求路径（`du_e2ap_msg_hdl.c:10094-10116`）按同样方式改造，补上缺失的容量检查。

仅给计数补 `--` 不能解决问题：乱序完成与中间空洞仍会导致重复使用或再次越界。

## 八、提交前附件清单

已有：本报告、隔离测试源码、复现记录（含普通构建与 ASan 两段原始输出）。
建议附：`gcc --version` 与 libasan 版本、固定提交的源码行截图或链接、ASan 完整报告（未被截断的版本）。
待补：报告人信息、是否已向 O-RAN SC 上游报送的说明、完整 O-DU 环境下的端到端验证结果（如有）。

## 九、披露状态声明

- 截至本文编制日，本漏洞尚未向 O-RAN SC 上游安全团队、MITRE 或 CNVD 提交；无 CVE / CNVD 编号。
- 证据为隔离程序复刻生产控制流所得，并非完整 O-DU 运行实例的崩溃记录；本报告已如实标注该边界，不将隔离结论表述为端到端复现。
- 若提交 CNVD，应在“是否已向其他平台报送”栏如实填写。
