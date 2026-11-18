# SUB-01 复现记录（2026-11-18 环境重建版）

## 环境

- WSL Ubuntu，Go 1.24.6（`source env.sh`）
- 真实 ric-plt-submgr 源码树（审查提交 `98ebf666`）内运行 `go test`
- **本次部署新增的关键环境变量**（此前 2026-11-06 的记录缺这几条，导致重跑失败）：

```bash
source /root/oran-lab/experiments/crash-validation-20261105/env.sh   # Go 1.24.6 + GOPROXY
export CFG_FILE=/tmp/submgr-test.yaml          # xapp-frame 的 viper 配置（-f/CFG_FILE 读取）
export RMR_SEED_RT=<submgr>/config/submgr-uta-rtg.rt
export DBAAS_SERVICE_HOST=10.0.2.12            # sdlgo 的 Redis 地址（ric_dbaas 容器 IP）
export DBAAS_SERVICE_PORT=6379
export SUBMGR_CRASH_DEMO=1                     # 崩溃演示测试的安全门（缺省 Skip）
```

其中 `/tmp/submgr-test.yaml` 由仓库 `config/submgr-config.yaml` 复制并将
`db.host` 从 `:6379` 改为 `10.0.2.12:6379`。

## 测试运行前的必要操作

上游测试文件 `ut_test.go` 的 `TestMain` 依赖配置文件与 RTMGR stub，
会把整个测试二进制在 init 阶段退出（`Config File "config" Not Found`）。
**将上游 11 个 `*_test.go` 移入 stash 目录，只保留 4 个 crash_*_test.go**；
其中 `crash_sub01_test.go` / `crash_sub03_test.go` 依赖上游 `mainCtrl` 变量，
如只跑隔离版也一并 stash，跑完恢复。

## 命令

```sh
cd ric-plt-submgr/pkg/control
# 1) race 检测（证据：7 条 DATA RACE）
go test -run TestSub01RaceIsolated -v -count=1 -race -vet=off .
# 2) 原生并发 map 写崩溃（证据：fatal error，进程终止）
SUBMGR_CRASH_DEMO=1 go test -run TestSub01ConcurrentMapWriteCrash -v -count=1 -vet=off .
```

## 原始输出（关键行，2026-11-18 实测）

race 测试（7 条 DATA RACE，写入方 110/111/122 行，读取方 196/205 行）：

```text
WARNING: DATA RACE
Write at 0x... by goroutine 182:
      pkg/control/e2if_state.go:110 +0x2bc      # NotificationCb 写 NbIdMap
Previous read at 0x... by goroutine 183:
      pkg/control/e2if_state.go:196 +0x239      # 订阅路径读同一 map
WARNING: DATA RACE
Write at ... e2if_state.go:111 +0x336
Previous read at ... e2if_state.go:205 +0x108
```

原生崩溃测试（无 race detector）：

```text
fatal error: concurrent map iteration and map write
goroutine 94 [running]:
...
gerrit.o-ran-sc.org/r/ric-plt/submgr/pkg/control.TestSub01ConcurrentMapWriteCrash.func2()
      crash_sub01_isolated_test.go:139 +0x8f
```

## 测试源码

- `crash_sub01_isolated_test.go`：
  - `TestSub01RaceIsolated`：单写者 + 解锁读者 + 加锁读者并发驱动
    （race 报告干净，全部指向被测代码）
  - `TestSub01ConcurrentMapWriteCrash`：`GetAllE2Nodes()` 返回内部 map 的
    迭代者与 SDL 通知写者并发（需 `SUBMGR_CRASH_DEMO=1`）

## 已知行为差异（本次环境 vs 2026-11-06）

- Go 1.24 vet 新增"非常量格式串"检查，会拦截上游 `ut_test.go:38`，需 `-vet=off`
- race 测试在本次环境共报 7 条 DATA RACE（上次记录为 2 条路径），
  行数差异来自并发调度时序，不影响结论
