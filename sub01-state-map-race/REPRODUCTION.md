# SUB-01 复现记录

## 环境

- WSL Ubuntu，Go 1.24.6（`source env.sh`：GOPROXY=goproxy.cn）
- 真实 ric-plt-submgr 源码树（审查提交 `98ebf666`）内运行 `go test -race`

## 命令

```sh
cd ric-plt-submgr
source env.sh
# 隔离用例：不启动完整 RMR 测试框架，仅驱动 E2IfState map 与 NotificationCb
go test ./pkg/control/ -run TestSub01 -v -count=1 -race
```

`crash_sub01_isolated_test.go` 是不依赖 RMR harness 的隔离版本，保证 race 报告只指向被测代码而非上游测试桩。

## 原始输出（关键行）

```text
fatal error: concurrent map writes
```

以及 race detector 报告（隔离用例）：

```text
WARNING: DATA RACE
Write at ... by goroutine N:
  e2mgr.../control.(*E2IfState)...
      pkg/control/e2if_state.go:110      # NotificationCb 写 NbIdMap / NbIdStatusMap
Previous read at ... by goroutine M:
      pkg/control/e2if_state.go:196      # 订阅处理路径读同一 map
```

Go runtime 原生 `fatal error: concurrent map writes` 为进程级、无条件终止——生产中即 submgr 进程退出，不依赖 race detector。

## 测试源码

- `crash_sub01_isolated_test.go`：bare E2IfState + 真实计数组，并发驱动 `NotificationCb` 写路径与状态读路径（推荐，报告干净）；
- `crash_sub01_test.go`：完整 harness 版本（需 RMR 初始化）。
