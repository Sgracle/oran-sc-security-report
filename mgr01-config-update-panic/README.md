# ric-plt-e2mgr：节点配置更新越界索引 panic（VUL-MGR-01）

- **组件**：o-ran-sc/ric-plt-e2mgr（验证提交 `353939c066468e3c25244a82c65a1da24701669b`；运行栈 v6.0.4）
- **CWE**：CWE-248（未捕获异常）/ CWE-703；Go 运行时边界检查将潜在越界写转化为确定性 panic
- **结论**：CONFIRMED（panic 变体，真实源码树内复现）；REFUTED（"写错条目"静默数据破坏变体）
- **影响**：进程崩溃 DoS（e2mgr 退出 → 该 RIC 实例所有 E2 节点管理中断）
- **触发者**：E2 节点的常规配置更新消息

## 成因

`E2Manager/handlers/rmrmsghandlers/e2_node_config_update_notification_handler.go:134-167` 的 `handleUpdateConfig()`：在内层循环 `j` 上找到与更新项匹配的现有配置 `NodeConfigs[j]`，但随后写回的是**外层请求索引** `NodeConfigs[i]`（GNB、ENB 两个分支均如此）。当更新列表项数超过已存配置列表项数时，`NodeConfigs[i]` 越界，Go 运行时抛出 index out of range panic。

该函数运行在通知处理 goroutine 中，派发链（`E2Manager/managers/notificationmanager/notification_manager.go:40-54`，每条通知一个 goroutine）没有 recover，panic 直接终止整个 e2mgr 进程。

**实测证伪的子场景**：原审查报告推测"列表等长但顺序不同时会静默覆写错误条目"。实测生产比较条件为"接口类型相等 AND AmfName 相等"双条件，不匹配即跳过，不存在静默数据破坏。本漏洞的可达危害只有越界 panic 一条路径。

## 触发路径

1. E2 节点发送 E2 Node Configuration Update（gNB 常规操作：配置变更、AMF 调整等）；
2. 通知到达 e2mgr 后异步派发到 `handleUpdateConfig()`；
3. 更新列表项数 > 已存 `NodeConfigs` 项数（含先出现无法匹配项的情形）→ `NodeConfigs[i]` 越界；
4. panic 未被 recover → e2mgr 进程退出。

## 实测证据

真实生产代码路径触发：`panic: runtime error: index out of range [1] with length 1`，panic 前生产日志输出 "item at position [1] should be updated"。

## 复现

测试源码 `crash_mgr01_test.go` 与完整复现记录（命令、原始输出）见本目录 [REPRODUCTION.md](REPRODUCTION.md)。

## 修复建议

GNB 与 ENB 两处分支均按匹配索引 `j` 替换（`NodeConfigs[i] = &u` → `NodeConfigs[j] = &u`），不要对 `i` 做边界截断；更新前完整校验未知、重复配置并明确语义。
