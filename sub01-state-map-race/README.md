# ric-plt-submgr：E2 状态回调与订阅处理并发读写 map（VUL-SUB-01）

- **组件**：o-ran-sc/ric-plt-submgr（验证提交 `98ebf666e797ee02a7b2946742ebcc4259252b4c`；运行栈 v0.10.1）
- **CWE**：CWE-362（并发竞争）
- **结论**：CONFIRMED（真实源码树内复现：Go 原生 `fatal error` + race detector 双证据）
- **影响**：进程崩溃 DoS（Go runtime fatal 无条件终止整个 submgr 进程，非概率性）
- **触发者**：E2 节点断连/重连回调 × 任意 xApp 的订阅请求/查询（常态组合）

## 成因

`pkg/control/e2if_state.go`：`NotificationCb()`（94-139 行）处理 CONNECTED / DISCONNECTED / UNDER_RESET 状态回调时写 `NbIdMap` / `NbIdStatusMap`（110 行），而订阅处理路径读取这两个 map（196 行）。结构体虽有 mutex，部分查询接口也持锁，但**写端不使用它**，无法形成互斥。`GetAllE2Nodes()`（87-92 行）还直接返回内部 map 引用，锁释放后共享数据仍暴露给调用方。

## 触发路径

1. E2 节点断连/重连，触发 RAN 状态回调写入状态 map；
2. 同时任意 xApp 正常发起订阅请求或查询（读取同一 map）；
3. 两者重叠 → Go runtime 检测到并发 map 读写 → `fatal error: concurrent map writes` → submgr 进程退出。

## 实测证据

并发回调+读路径测试：原生 `fatal error: concurrent map writes`；race 报告指向 e2if_state.go:110（写）vs :196（读）。

## 复现

测试源码 `crash_sub01_isolated_test.go`（隔离版）与 `crash_sub01_test.go`（harness 版）及完整复现记录见本目录 [REPRODUCTION.md](REPRODUCTION.md)。

## 修复建议

为两个 map 的全部读写统一加锁（可用同一 RWMutex）；getter 返回副本而非内部 map；回调在锁内更新状态后先解锁，再调用注册表清理，避免持状态锁执行路由/数据库 I/O 或造成锁顺序循环。
