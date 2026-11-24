# SUB-NEW-1：未认证 RMR 消息导致 submgr 进程 SIGSEGV 崩溃（拒绝服务）

> 日期：2026-11-19
> 组件：o-ran-sc/ric-plt-submgr（容器 ric-plt-submgr:0.10.1，含 xapp-frame-py/go v0.9.16 的 RMR 接收线程）
> 漏洞类型：CWE-20（输入验证不当）→ 未认证远程拒绝服务
> 测试版本：ric-plt-submgr 0.10.1（Docker 镜像 nexus3.o-ran-sc.org:10002/o-ran-sc/ric-plt-submgr:0.10.1）

## 一、漏洞概述

submgr 的 RMR 接收端点（tcp:4560）不对连接方做任何认证，且 RMR 消息解析线程
（xapp-frame 的 `RMRClient.parseMessage`，rmr.go:305）对畸形/未预期消息缺乏健壮性。
攻击者只要能访问 RIC 内网的 4560 端口，发送一条约 300 字节的伪造 RMR 消息
（类型 12023 / RIC Subscription Delete Required，E2AP 载荷为合法 ASN.1 编码），
即可使 submgr 进程立即发生 SIGSEGV 段错误并崩溃退出。

这是**未认证的远程拒绝服务**：无需任何凭据，单条消息即崩溃整个订阅管理服务，
导致 RIC 上所有 xApp 的 E2 订阅管理能力全部失效。

## 二、崩溃证据（两次独立复现）

### 第一次崩溃
- 注入时间：UTC 2026-11-19T07:11:13 之前数秒（本地 15:11）
- 崩溃：`[signal SIGSEGV: segmentation violation code=0x1 addr=0x7d803c003241 pc=0xa3e886]`
- 崩溃点：`xapp.(*RMRClient).parseMessage` rmr.go:305
- 容器：`ric_submgr Exited (2)`

### 第二次崩溃（submgr 重启后，容器内为全新进程）
- 注入时间：UTC 2026-11-19T11:12:50 之前数秒
- 崩溃：`[signal SIGSEGV ... addr=0x7f59ac000c41 pc=0xa3e886]`（pc 相同=同一代码位置）
- 容器再次退出

两次崩溃 pc=0xa3e886 完全一致，排除偶然因素。

## 三、触发路径

1. 攻击端（RIC 内网任意主机）向 submgr 的 RMR 端口 `10.0.2.13:4560` 建立 TCP 连接；
2. 发送 350 字节：[50 字节传输层前缀（含 `$` 标记与双格式长度）] + [276 字节 uta_mhdr_t RMR 头
   （mtype=12023, plen=24, rmr_ver=3, meid="gnb_0115_100_00025b"）] + [24 字节 E2AP 载荷
   （asn1c 编码的 RIC Subscription Delete Required，requestor=123, instance=2, func=2, cause=misc/unspecified）]；
3. submgr 的 RMR 接收线程收到后，在 `parseMessage`（rmr.go:305）处理期间发生空指针/非法地址解引用，
   触发 `runtime.sigpanic` → SIGSEGV → 进程立即退出；
4. 崩溃发生在进入业务处理函数 `handleE2TSubscriptionDeleteRequired` 之前或之中——
   即框架层对消息字段（Meid/长度）的解析本身缺乏防护。

## 四、复现步骤

```bash
# 攻击工具：rmr_inject（Go，纯 TCP 客户端，无需 RMR 库）
cd /root/oran-lab/experiments/submgr-e2node/rmr-inject
export GOROOT=/tmp/goroot124
export PATH=/tmp/goroot124/bin:/usr/bin:/bin
export GOTOOLCHAIN=local
go build -o rmr_inject ./cmd

# 注入（目标：submgr 容器 RMR 端口）
./rmr_inject 10.0.2.13:4560

# 观察：submgr 容器立即退出
docker ps -a --format '{{.Names}}: {{.Status}}' | grep submgr
# ric_submgr: Exited (2)
```

注入器源码：`rmr-inject/cmd/main.go`（Windows 侧 `D:\309\graduation\tmp\rmr_inject_v2.go`）
- RMR 头构建：`buildRMRHeader()`，276 字节 uta_mhdr_t（字段顺序与 rmr_agnostic.h 中
  uta_mhdr_t 完全一致：mtype/plen/rmr_ver/xid/sid/src/meid/ts/flags/len0-3/sub_id/srcip）
- 传输层前缀：50 字节 TP_HDR_LEN（insert_mlen 格式：本地序 len + 网络序 len + '$' 标记）
- 载荷：24 字节 asn1c 编码的 RIC Subscription Delete Required（用 e2term 3rdparty/oranE2
  生成的代码编码，本地解码验证通过：requestor=123 instance=2 func=2）

## 五、影响

- submgr 崩溃期间，RIC 平台上所有 xApp 的订阅管理不可用；
- 攻击者可持续发送该消息实现**持续性拒绝服务**（每次 submgr 重启后再次注入即再次崩溃）；
- 无需认证、无需有效订阅、无需 E2 连接——只需要能访问 RIC 内网 4560 端口；
- 修复建议：
  1. parseMessage 对畸形消息（Meid 为空、长度超界、未注册类型）做防御性检查并丢弃；
  2. RMR 端口增加源认证（mTLS/令牌）；
  3. 对消息类型与载荷长度做白名单校验后再进入解析。

## 六、证据文件

| 文件 | 内容 |
|---|---|
| crash-evidence/submgr_full.log | submgr 完整日志（含两次崩溃） |
| crash-evidence/submgr_crash_tail.log | 崩溃前最后 300 行 |
| crash-evidence/both_sigsegs.json | 两次 SIGSEGV 的原始日志行 |
| crash-evidence/crash2_sigsegv.json | 第二次崩溃的 SIGSEGV 与完整 goroutine 堆栈 |
| crash-evidence/crash2_before.json | 第二次崩溃前的最后日志 |
| crash-evidence/injected_12023.json | 注入前 12023 消息的参考记录 |

## 七、源码定位

- 崩溃点：xapp-frame v0.9.16 `pkg/xapp/rmr.go:305`（parseMessage）
- RMR 头结构：ric-plt/rmr `src/rmr/common/include/rmr_agnostic.h`（uta_mhdr_t）
- 传输层前缀：ric-plt/rmr `src/rmr/si/include/rmr_si_private.h`（TP_HDR_LEN=50、TP_SZ_MARKER='$'）
- submgr 的消息处理入口：ric-plt-submgr `pkg/control/control.go:1686`
  （handleE2TSubscriptionDeleteRequired，与 RIC-1079 的 C1 案例同属此文件）
