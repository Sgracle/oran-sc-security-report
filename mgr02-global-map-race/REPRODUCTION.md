# MGR-02 复现记录

## 环境

- WSL Ubuntu，Go 1.24.6（`source env.sh`：GOPROXY=goproxy.cn）
- 真实 ric-plt-e2mgr 源码树（审查提交 `353939c0`）内运行 `go test`

## 命令

```sh
cd ric-plt-e2mgr/E2Manager
source env.sh

# 子缺陷 1：ProcedureMap 跨锁竞态（race detector）
go test ./handlers/rmrmsghandlers/ -run TestMgr02ProcedureMapCrossLockRace -v -count=1 -race -vet=off

# 子缺陷 2：ExistingRanFunctiuonsMap 无锁并发（原生 fatal error，无需 race detector）
go test ./handlers/rmrmsghandlers/ -run TestMgr02ExistingRanFunctionsMapUnlocked -v -count=1 -vet=off
```

## 原始输出

子缺陷 1（race detector 报告头部）：

```text
WARNING: DATA RACE
Write at 0x00c00021ec60 by goroutine 10:
  runtime.mapassign_faststr()
  e2mgr/models.UpdateProcedureType()
      .../models/e2_error_indication_message.go:30 +0xb4
Previous read at 0x00c00021ec60 by goroutine 16:
  runtime.mapaccess2_faststr()
  e2mgr/handlers/rmrmsghandlers.(*ErrorIndicationHandler).HandleBasedOnProcedureType()
      .../handlers/rmrmsghandlers/error_indication_notification_handler.go:116 +0x87
```

写方持 models 包级锁、读方持 handler 自带锁，两把锁互不排斥，race 落在 `ProcedureMap` 同一地址。

子缺陷 2（无 race detector 的原生崩溃）：

```text
fatal error: concurrent map read and map write
FAIL	e2mgr/handlers/rmrmsghandlers	0.009s
```

0.009 秒内 Go runtime 直接终止测试进程——生产中即 e2mgr 进程无条件退出。

## 测试源码

`crash_mgr02_test.go`：写方走生产 `models.UpdateProcedureType()`，读方走生产 `HandleBasedOnProcedureType()`（mock 掉两个 manager 接口）；无锁 map 用例按 `ric_service_update_manager.go:58/70` 的生产语句原样并发读写。
