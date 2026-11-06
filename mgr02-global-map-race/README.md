# ric-plt-e2mgr：全局 map 同步机制失效（VUL-MGR-02）

- **组件**：o-ran-sc/ric-plt-e2mgr（验证提交 `353939c066468e3c25244a82c65a1da24701669b`；运行栈 v6.0.4）
- **CWE**：CWE-362（并发竞争）
- **结论**：CONFIRMED（两个独立子缺陷均实测复现：race detector + Go 原生 fatal error 双证据）
- **影响**：进程崩溃 DoS（Go runtime fatal 无条件终止整个 e2mgr 进程，非概率性）
- **触发者**：RIC 内网中时序重叠的常规消息（E2 Setup / Service Update / Error Indication）

## 成因（两个独立子缺陷）

**子缺陷 1：跨锁竞态**。`E2Manager/models/e2_error_indication_message.go:22-33`——写方 `UpdateProcedureType()` 使用 models 包级 `procedureMapMutex` 保护 `ProcedureMap` 写入；读方 `ErrorIndicationHandler.HandleBasedOnProcedureType()`（`E2Manager/handlers/rmrmsghandlers/error_indication_notification_handler.go:114-117`）却使用 handler 结构体自己的另一把同名锁。两把锁互不排斥，`ProcedureMap` 的读写不受任何互斥保护。

**子缺陷 2：完全无锁**。`models.ExistingRanFunctiuonsMap`（`e2_error_indication_message.go:33`）在 `E2Manager/managers/ric_service_update_manager.go:58`（写）和 `:70`（读）之间无任何同步。

e2mgr 的通知管理器对每条通知启动独立 goroutine，E2 Setup / Service Update / Error Indication 天然重叠处理，无需攻击者精确控制时序。不同 RAN 名称的不同 map key 也不能让 Go 原生 map 安全并发。

## 触发路径

1. gNB 断连/重连时 Error Indication 与 E2 Setup、Service Update 消息重叠（常态场景）；
2. Error Indication 读 `ProcedureMap` 与 Setup/ServiceUpdate 写它并发 → 数据竞争；
3. Service Update 备份/回滚路径并发读写 `ExistingRanFunctiuonsMap` → Go runtime 检测到并发 map 访问；
4. `fatal error: concurrent map read and map write` → 整个 e2mgr 进程立即退出。

## 实测证据

- race detector：`WARNING: DATA RACE`，写方 `models.UpdateProcedureType`（e2_error_indication_message.go:30）vs 读方 `HandleBasedOnProcedureType`（error_indication_notification_handler.go:116）；
- 原生运行（无 race detector）：0.009 秒内 `fatal error: concurrent map read and map write`，进程级崩溃。

## 复现

测试源码与命令见 [验证实验记录](../crash-validation-20261106.md)。核心：真实 e2mgr 源码树内 `go test ./handlers/rmrmsghandlers/ -run TestMgr02 -v -race -vet=off`（测试文件 `crash_mgr02_test.go`，写方走生产 `UpdateProcedureType`、读方走生产 `HandleBasedOnProcedureType`）。

## 修复建议

把两个 map 封装在拥有它们的包内，所有读、写、删除通过同一所有者上的锁进行；删除 handler 内误导性的"局部读锁"；读取函数在锁内取值、锁外执行业务或数据库操作；备份的 RanFunction 列表应深拷贝。若业务要求同一 RAN 操作严格有序，还需按 RAN 串行化。
