# O-RAN SC 组件已验证安全漏洞说明（用于 CVE 上报）

整理日期：2026-11-06
本报告所有漏洞均已在隔离环境中完成实测复现，复现方法与原始输出见同目录 `validation-report-20261106.md`。静态审查发现但**尚未实测**的候选（E2-01、DU-01、DU-02）不在本报告正文，见附录 A。

---

## 0. 受影响组件与版本

| 组件 | 项目 | 审查/验证版本 | 运行形态 |
|---|---|---|---|
| Subscription Management (submgr) | o-ran-sc/ric-plt-submgr | commit `98ebf666e797ee02a7b2946742ebcc4259252b4c`（运行栈为 v0.10.1） | RIC 平台内 Go 服务，处理 E2 订阅 |
| E2 Manager (e2mgr) | o-ran-sc/ric-plt-e2mgr | commit `353939c066468e3c25244a82c65a1da24701669b`（运行栈为 v6.0.4） | RIC 平台内 Go 服务，管理 E2 节点 |
| O-DU L2 (du_app) | o-ran-sc/o-du-l2 | commit `2278be373834a0a198e0b5dac94cc66397d88421` | C 语言 O-DU，通过 E2AP 接收 RIC 请求 |

网络位置：以上组件均部署在 RIC 平台内部网络。攻击者前提通常为：可控的 E2 节点（gNB/eNB）、RIC 内网中可发送 RMR/REST 消息的进程、或能注入/伪造 E2AP/SCTP 流量的内网主机。

---

## 1. VUL-DU-04：O-DU L2 订阅待响应槽位越界写（堆缓冲区溢出）

- **CWE**：CWE-787（越界写入）；伴随 CWE-762/计数生命周期缺陷
- **严重性参考**：内存破坏类，可导致相邻状态损坏、进程崩溃；建议按 High 评估
- **验证方式**：隔离 C 程序复刻生产控制流，GCC 直跑 + AddressSanitizer 双重确认

### 成因
`src/du_app/du_e2ap_mgr.h:39,445`：`RanFunction` 结构体中 `pendingSubsRspInfo` 数组固定 5 个元素（`MAX_PENDING_SUBSCRIPTION_RSP = 5`）。`procRicSubscriptionRequest()`（`src/du_app/du_e2ap_msg_hdl.c`）在 3520 行检查 `numPendingSubsRsp >= 5` 即置失败并跳出，但函数结尾的统一错误清理路径（3607 行）**无条件**执行：

```c
memset(&ranFuncDb->pendingSubsRspInfo[ranFuncDb->numPendingSubsRsp],
       0, sizeof(PendingSubsRspInfo));
```

当 `numPendingSubsRsp == 5` 时，该语句对 `pendingSubsRspInfo[5]`（合法下标 0..4）做一次 64 字节清零，写入数组之后的相邻内存。

加重因素：成功请求会使 `numPendingSubsRsp++`（3596 行），而响应完成路径 `procStatsRspForSubsReq()`（`du_e2ap_mgr.c:832-840`）只清零槽位、**从不递减计数**——即无需 5 个请求并发，仅靠连续成功的订阅请求即可把计数推到 5，随后下一个请求必然触发越界。同类问题也存在于修改请求路径（`du_e2ap_msg_hdl.c:10094-10116`，`numPendingSubsModRsp` 直接作下标，连容量检查都没有）。

### 触发路径
1. RIC（xApp）向 O-DU 发送 E2AP RIC Subscription Request；
2. 同一 RAN Function 累计处理 5 次订阅（无论成功还是失败，计数只增不减）；
3. 第 6 个订阅请求到达 → 3520 行容量检查失败 → 错误清理路径执行 `memset` 越界写。

攻击者只需能反复发起订阅请求（这是 xApp 的常规操作），不需要并发。

### 实测证据
- 普通构建：越界 memset 将紧邻的结构体成员清零（哨兵值 0x41424344 → 0x00000000），布局检查确认写入目标即相邻成员地址；
- ASan：`heap-buffer-overflow, WRITE of size 64`，位于数组末尾 0 字节偏移。

---

## 2. VUL-DU-03：O-DU L2 失败响应清理函数误用分配宏（内存资源耗尽）

- **CWE**：CWE-401（缺少内存释放后的有效引用/内存泄漏）；伴随 CWE-770（资源分配无限制）
- **严重性参考**：远程可触发、每请求必然泄漏，长期可致 OOM；建议按 Medium~High 评估
- **验证方式**：隔离 C 程序，用计数内存池原样替换 `SGetSBuf/SPutSBuf`，函数体原样复刻，1000 次调用统计

### 成因
`FreeRicSubscriptionFailure()`（`src/du_app/du_e2ap_msg_hdl.c:3630-3652`）名为 Free，实际四处全部调用 `DU_ALLOC` 而非 `DU_FREE`（宏定义见 `src/du_app/du_utils.h:67-92`）。`DU_ALLOC` 会**分配新内存并覆盖指针**：原对象从此不可达，新分配的对象也永远不会被释放。四处分别是：每个 IE（循环内）、IE 数组整体、`unsuccessfulOutcome`、`e2apMsg` 本身。

该函数在 `BuildAndSendRicSubscriptionFailure()` 的收尾路径（`du_e2ap_msg_hdl.c:3782`）被调用——**每一个被拒绝的订阅请求**都会走到这里。

### 触发路径
1. RIC/xApp 发送会被 O-DU 拒绝的 RIC Subscription Request（如无效 RAN Function ID、资源不足等常规失败场景）；
2. O-DU 构建并发送 RIC Subscription Failure；
3. 收尾时 `FreeRicSubscriptionFailure` 反向泄漏：每次泄漏 4+ 次池分配（IE 数量越多越多）；
4. 持续的订阅失败流量逐步耗尽 DU 自定义内存池 / 触发 OOM。

### 实测证据
1000 次"构建失败响应→释放"循环：`allocs=6000, frees=0`（每次调用净泄漏 6 次分配）；LeakSanitizer 独立报出同规模直接泄漏。泄漏量与失败响应次数严格线性增长。

---

## 3. VUL-MGR-01：e2mgr 节点配置更新越界索引（不可恢复 panic，进程崩溃 DoS）

- **CWE**：CWE-248（未捕获异常）/ CWE-703；Go 运行时边界检查将潜在越界写转化为确定性 panic
- **严重性参考**：远程 DoS（e2mgr 进程退出）；建议按 High 评估
- **验证方式**：真实 e2mgr 源码树内 `go test` 调用生产函数

### 成因
`E2Manager/handlers/rmrmsghandlers/e2_node_config_update_notification_handler.go:134-167` 的 `handleUpdateConfig()`：在内层循环 `j` 上找到与更新项匹配的现有配置 `NodeConfigs[j]`，但随后写回的是**外层请求索引** `NodeConfigs[i]`（GNB、ENB 两个分支均如此）。当更新列表长度超过已存配置列表长度时，`NodeConfigs[i]` 越界，Go 运行时抛出 index out of range panic。该函数运行在通知处理 goroutine 中，派发链（`notification_manager.go:40-54`，每条通知一个 goroutine）没有 recover，panic 直接终止整个 e2mgr 进程。

注：实测同时证伪了"长度相同也会静默覆写错误条目"的子场景——生产比较条件为接口类型 + AmfName 双条件，不匹配即跳过，无静默数据破坏。上报时应按"越界 panic / DoS"表述。

### 触发路径
1. E2 节点发送 E2 Node Configuration Update（gNB 的常规操作，如配置变更、AMF 调整）；
2. 更新通知到达 e2mgr 后异步派发到 `handleUpdateConfig()`；
3. 只要更新列表项数 > 已存 `NodeConfigs` 项数且先出现无法匹配的项（或直接多于现有项），`NodeConfigs[i]` 越界；
4. panic 未被 recover → e2mgr 进程退出 → 该 RIC 实例所有 E2 节点管理能力中断。

### 实测证据
真实生产代码路径触发：`panic: runtime error: index out of range [1] with length 1`，panic 前生产日志输出 "item at position [1] should be updated"。

---

## 4. VUL-MGR-02：e2mgr 全局 map 同步机制失效（并发崩溃 DoS）

- **CWE**：CWE-362（并发竞争）/ CWE-414
- **严重性参考**：远程 DoS（Go runtime fatal 直接杀进程）；建议按 High 评估
- **验证方式**：真实 e2mgr 源码树内 `go test`，race detector + 原生运行双证据

### 成因（两个独立子缺陷）
1. **跨锁竞态**：`models/e2_error_indication_message.go:22-33`——写方 `UpdateProcedureType()` 使用 models 包级 `procedureMapMutex`；读方 `ErrorIndicationHandler.HandleBasedOnProcedureType()`（`error_indication_notification_handler.go:114-117`）却使用 handler 结构体自己的另一把同名锁。两把锁互不排斥，`ProcedureMap` 的读写不受任何互斥保护。
2. **完全无锁**：`models.ExistingRanFunctiuonsMap`（`e2_error_indication_message.go:33`）在 `ric_service_update_manager.go:58`（写）和 `:70`（读）之间无任何同步。

e2mgr 的通知管理器对每条通知启动独立 goroutine，E2 Setup / Service Update / Error Indication 天然重叠处理，无需攻击者精确控制时序。

### 触发路径
1. RIC 内网中产生并发的 E2 Setup、RIC Service Update、Error Indication 消息（gNB 断连/重连时 Error Indication 与其他消息重叠是常态）；
2. Error Indication 读 `ProcedureMap` 与 Setup/ServiceUpdate 写它并发 → 数据竞争；
3. Service Update 备份/回滚路径并发读写 `ExistingRanFunctiuonsMap` → Go runtime 检测到并发 map 访问；
4. `fatal error: concurrent map read and map write` → 整个 e2mgr 进程立即退出。

### 实测证据
- race detector：`WARNING: DATA RACE`，写方 `models.UpdateProcedureType`（e2_error_indication_message.go:30）vs 读方 `HandleBasedOnProcedureType`（error_indication_notification_handler.go:116）；
- 原生运行（无 race detector）：0.009 秒内 `fatal error: concurrent map read and map write`，进程级崩溃。

---

## 5. VUL-SUB-01：submgr E2 状态回调与订阅处理并发读写 map（并发崩溃 DoS）

- **CWE**：CWE-362（并发竞争）
- **严重性参考**：远程 DoS；建议按 High 评估
- **验证方式**：真实 submgr 源码树内 `go test -race` + 原生 fatal error

### 成因
`pkg/control/e2if_state.go`：`NotificationCb()`（94-139 行）处理 CONNECTED / DISCONNECTED / UNDER_RESET 状态回调时写 `NbIdMap` / `NbIdStatusMap`（110 行），而订阅处理路径读取这两个 map（196 行）。结构体虽有 mutex，部分查询接口也持锁，但**写端不使用它**，无法形成互斥。`GetAllE2Nodes()`（87-92 行）还直接返回内部 map 引用，锁释放后共享数据仍暴露给调用方。

### 触发路径
1. E2 节点断连/重连（触发 RAN 断连状态回调写入状态 map）；
2. 同时任意 xApp 正常发起订阅请求或查询（读取同一 map）；
3. 两者重叠 → Go runtime 检测到并发 map 读写 → `fatal error: concurrent map writes` → submgr 进程退出。

### 实测证据
并发回调+读路径测试：原生 `fatal error: concurrent map writes`；race 报告指向 e2if_state.go:110（写）vs :196（读）。

---

## 6. VUL-SUB-02：submgr 断连清理绕过注册表锁（并发崩溃 DoS）

- **CWE**：CWE-362（并发竞争）
- **严重性参考**：远程 DoS，且可能造成订阅 ID 重复分配等状态损坏；建议按 High 评估
- **验证方式**：真实 submgr 源码树内 `go test -race`

### 成因
`pkg/control/registry.go:627-673` 的 `DeleteAllE2Subscriptions()` 遍历并删除 `r.register` 和 `r.restSubscriptions`、回收 `r.subIds`，但**全程不获取 `r.mutex`**；调用点（`e2if_state.go:119-136` 的断连回调）也不代持。常规订阅创建/删除/查询路径均持 `r.mutex`（如 registry.go:356-361），因此清理函数与它们之间没有任何互斥。"跳过处理中订阅"只判断条目状态，保护不了 map 本身。

### 触发路径
1. 某一 RAN 断连，触发 `DeleteAllE2Subscriptions()` 无锁遍历/删除注册表；
2. 同时另一 xApp 对任意 RAN（包括其他 RAN）创建或删除订阅（持锁路径）；
3. 无锁遍历与持锁写并发 → Go fatal error 或 race → submgr 进程退出；`subIds` ID 池竞争还可能造成订阅 ID 重复分配。

### 实测证据
race 报告：registry.go:630（无锁遍历）vs 651/652（无锁 delete + subIds 回收）。

---

## 7. VUL-SUB-03：submgr 生产启动注册测试退出入口（认证后远程进程终止）

- **CWE**：CWE-489（活动调试代码暴露于生产）
- **严重性参考**：取决于 REST 端点的实际可达性（RIC 平台网关/入口策略）；端点在平台内网可直接访问时建议按 Medium~High 评估
- **验证方式**：真实 submgr 源码树内测试确认退出行为

### 成因
`pkg/control/control.go:167-173` 在**普通启动路径** `NewControl()` 中注册测试路由，对应处理器 `pkg/control/debug_rest_if.go:64-68` 的 restart 测试动作直接执行 `os.Exit(1)`。这不是内存安全问题，而是明确的测试退出能力被编入生产二进制且默认注册。

### 触发路径
1. 能访问 submgr REST 端点的主体（RIC 平台内任意进程；若入口策略允许，则为更广的攻击者）发送对应测试动作的 HTTP 请求；
2. 处理器命中测试动作即执行 `os.Exit(1)`，限流等前置中间件不能阻止已放行请求的退出效果；
3. submgr 进程立即终止。

### 实测证据
测试路由命中即退出进程（`os.Exit(1)`）。

---

## 8. 漏洞汇总表

| 编号 | 组件 | CWE | 类型 | 影响 | 触发者 |
|---|---|---|---|---|---|
| VUL-DU-04 | o-du-l2 | CWE-787 | 堆越界写（64 字节 memset） | 内存破坏/崩溃 | RIC 侧订阅流量（无需并发） |
| VUL-DU-03 | o-du-l2 | CWE-401 | 内存泄漏（每失败响应 6 次分配） | 内存池耗尽/OOM | RIC 侧订阅拒绝流量 |
| VUL-MGR-01 | ric-plt-e2mgr | CWE-248/703 | 索引越界 panic | 进程崩溃 DoS | E2 节点配置更新消息 |
| VUL-MGR-02 | ric-plt-e2mgr | CWE-362 | 跨锁/无锁并发 map 访问 | 进程崩溃 DoS | RIC 内网消息时序重叠 |
| VUL-SUB-01 | ric-plt-submgr | CWE-362 | 状态回调并发 map 读写 | 进程崩溃 DoS | 断连回调 × 订阅请求 |
| VUL-SUB-02 | ric-plt-submgr | CWE-362 | 清理路径绕过注册表锁 | 进程崩溃 DoS + 状态损坏 | 断连回调 × 任意订阅操作 |
| VUL-SUB-03 | ric-plt-submgr | CWE-489 | 生产暴露测试退出入口 | 认证后进程终止 | REST 端点可达者 |

上报建议：
- 可按组件分别上报（o-du-l2 两项、e2mgr 两项、submgr 三项），同组件内可申请合并编号；
- 三个 Go 并发问题（MGR-02、SUB-01、SUB-02）表现为 runtime fatal 杀死整个服务进程，均为**确定性进程退出**（Go 对并发 map 访问是无条件 fatal，非概率性崩溃），这一点在报告中应明确；
- DU-04 的"计数只增不减"加重因素意味着触发不依赖并发，仅靠顺序请求即可，建议在上报中突出。

## 9. 复现材料索引

- Go 测试（真实源码树内）：`/root/oran-lab/experiments/crash-validation-20261105/ric-plt-{submgr,e2mgr}` 内 `crash_*_test.go`
- C 测试：`/root/oran-lab/experiments/crash-validation-20261105/du-test/`（du03_leak_test.c、du04_oob_test.c，plain/ASan 双构建）
- 完整复现命令与原始输出：`validation-report-20261106.md`

## 附录 A：静态审查发现、尚未实测的候选（不随本批上报）

以下发现来自 2026-10-27 静态审查、机制可信，但截至本报告未做运行时复现，建议完成验证后再单独上报：

1. **E2-01**（ric-plt-e2，`RIC-E2-TERMINATION/sctpThread.cpp:1566-1578`）：DEBUG 十六进制日志按 `2N+1` 需求只分配了 N 大小栈缓冲，长 SCTP 消息（>64KiB）+ DEBUG 开启时栈溢出写。位于 ASN.1 解码之前，解码校验不能防护。
2. **DU-01**（o-du-l2，`du_e2ap_mgr.c:297-311`）：`fetchRanFuncFromRanFuncId()` 以 `ranFunction[ranFuncId-1]` 直接索引 256 容量数组，而 ASN.1 允许 ID 0..4095；ID=0 为负下标，ID 257..4095 越界读，Service Query 路径还有越界写。
3. **DU-02**（o-du-l2，`du_e2ap_msg_hdl.c:3472-3566`）：订阅请求处理依赖 IE 顺序且不检查必需 IE，缺失/乱序时解引用空指针。


---

# 附：隔离实测验证记录（复现命令与原始输出）

对应评审文档：`D:\309\graduation\docs\code-review\oran-crash-review-2026-10-27.md`
实验环境：WSL Ubuntu（Go 1.24.6 / GCC 13.3），隔离源码副本
- Go 组件副本：`/root/oran-lab/experiments/crash-validation-20261105/ric-plt-{submgr,e2mgr}`
- C 组件副本：`D:\309\graduation\tmp\crash-review-20261027\o-du-l2`
- C 隔离测试：`/root/oran-lab/experiments/crash-validation-20261105/du-test/`
- Go 环境脚本：`source /root/oran-lab/experiments/crash-validation-20261105/env.sh`（GOPROXY=goproxy.cn）
- 全程未触碰运行中的 RIC 栈（submgr/e2term/e2mgr 6.0.x）。

## 结论总表

| 编号 | 组件 | 结论 | 关键证据 |
|------|------|------|----------|
| SUB-01 | ric-plt-submgr | CONFIRMED | 并发 NotificationCb → `fatal error: concurrent map writes`；race 报告 e2if_state.go:110 写 vs :196 读 |
| SUB-02 | ric-plt-submgr | CONFIRMED | DeleteAllE2Subscriptions 无锁遍历/删除注册表；race 报告 registry.go:630 vs 651/652 |
| SUB-03 | ric-plt-submgr | CONFIRMED | 测试路由命中指定动作时 `os.Exit(1)`，进程直接退出 |
| MGR-01（panic 变体） | ric-plt-e2mgr | CONFIRMED | 真实生产 update 循环 panic：`index out of range [1] with length 1`（生产日志 "item at position [1] should be updated" 后越界） |
| MGR-01（写错条目变体） | ric-plt-e2mgr | **REFUTED** | 生产比较逻辑（接口类型 + GetAmfName 两条件）行为正确，不匹配条目被跳过（连打 "didn't match"），无覆写错条目现象 |
| MGR-02 | ric-plt-e2mgr | CONFIRMED | ① ProcedureMap 跨锁竞态：race 报告写方 `models.UpdateProcedureType`（e2_error_indication_message.go:30，包级锁）vs 读方 `HandleBasedOnProcedureType`（error_indication_notification_handler.go:116，handler 自带锁）；② `ExistingRanFunctiuonsMap` 零锁并发 → 原生 `fatal error: concurrent map read and map write`（0.009s 内进程崩溃） |
| DU-03 | o-du-l2 | CONFIRMED | 原样复刻生产函数 + DU_ALLOC/DU_FREE 宏（计数内存池）：1000 次失败响应 → allocs=6000, frees=0，每周期净泄漏 6 块；LeakSanitizer 独立确认多批直接泄漏 |
| DU-04 | o-du-l2 | CONFIRMED | 槽位满（numPendingSubsRsp=5）后错误路径 `memset(&pendingSubsRspInfo[5])` 越界：普通构建实测把相邻结构体成员清零（哨兵 0x41424344→0）；ASan 报 `heap-buffer-overflow WRITE of size 64`（恰好一个 PendingSubsRspInfo），偏移在数组末尾 0 字节处 |

## 复现命令

```bash
# SUB-01/02/03（此前已完成，测试位于 ric-plt-submgr 各包）
source /root/oran-lab/experiments/crash-validation-20261105/env.sh
go test <对应包> -run TestSub -v -count=1 -race

# MGR-01（-vet=off 用于绕过 Go 1.24 对老代码 4 处 Errorf 非常量格式串的 vet 报错，与本缺陷无关）
cd /root/oran-lab/experiments/crash-validation-20261105/ric-plt-e2mgr/E2Manager
go test ./handlers/rmrmsghandlers/ -run TestMgr01 -v -count=1 -vet=off

# MGR-02
go test ./handlers/rmrmsghandlers/ -run TestMgr02 -v -count=1 -race -vet=off
# 无 race detector 的原生崩溃复现：
go test ./handlers/rmrmsghandlers/ -run TestMgr02ExistingRanFunctionsMapUnlocked -v -count=1 -vet=off

# DU-03 / DU-04
cd /root/oran-lab/experiments/crash-validation-20261105/du-test
./du03_plain   # 计数器汇总：cycles=1000 allocs=6000 frees=0
./du03_asan    # LeakSanitizer 直接泄漏报告
./du04_plain   # 哨兵被清零 + 布局检查确认越界目标即相邻成员
./du04_asan    # heap-buffer-overflow WRITE of size 64
```

## 对评审文档的修正建议

1. **MGR-01**：崩溃路径（越界 panic）成立；但"找到第 j 项却写第 i 项导致覆写错误条目"的静默数据破坏**不成立**——生产比较条件是"接口类型相等 AND AmfName 相等"，不匹配即跳过，不会写入。建议将文档中该子场景改为"更新请求中存在不匹配条目时跳过；仅当更新列表长于现有列表时越界 panic"。
2. **MGR-02 / SUB-01 / SUB-02**：均按文档描述成立，其中 ExistingRanFunctiuonsMap 与 e2if_state 的并发访问可直接 `fatal error` 杀死整个进程（不依赖 race detector）。
3. **DU-03**：确认泄漏量级为"每次订阅拒绝泄漏 6 次分配"（含 IE 数组整体），长期运行会耗尽内存池，与文档"非单次必崩、长期 OOM"的定性一致。

## 测试产物清单

- `ric-plt-e2mgr/E2Manager/handlers/rmrmsghandlers/crash_mgr01_test.go`（panic + wrong-entry 两用例）
- `ric-plt-e2mgr/E2Manager/handlers/rmrmsghandlers/crash_mgr02_test.go`（跨锁竞态 + 无锁 map 两用例）
- `du-test/du03_leak_test.c`、`du-test/du04_oob_test.c`（含 plain / asan 两种构建）
- submgr 侧测试（SUB-01/02/03）见 ric-plt-submgr 对应包内 crash_*_test.go
