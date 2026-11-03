# 独立只读复核记录

日期：2026-10-23。复核对象：本地生产差异、原生 matcher 测试、提取器、外部替身、行为用例、REPORT.md 和保存的前后原始证据。

结论：没有发现 Critical/Important 生产代码问题。可按本地源码与隔离内部流程的限定范围交付，不代表完整原生或端到端集成验证。

核对结果：

- E2Term 的连接 `peerInfo->enodbName` 传入 RMR MEID；订阅 owner 来自创建事务，RanName 关联正确。
- xApp Requestor ID=900 与线上固定 123 区别正确；新校验符合当前发送实现。
- nil 来源/owner/config 短路拒绝；校验先于收集删除对象、DeleteFromDb 设置、事务调度及清理。
- AST 提取保留声明字节，随后格式化；生成包的生产差异与补丁对应。fixed 六份源码 SHA 与工作树相符。
- 原始 A→B 证据显示基线输出 B `(123,102,2)` 并清理模拟记录；修复后无发送/清理且 B 保持原状。
- 报告正确限定 mock、静态分析、原生缺依赖及原有 vet 问题；没有误称失败/超时清理风险已经修复。

保留的 Minor：

1. `harness/ownership_test.go` 的混合批次覆盖 foreign、own、unknown，但错误 Requestor/Function 只覆盖单条；缺少错误标识条目与合法条目前后混排的用例。当前没有证据表明生产检查错误，属于额外组合覆盖不足。
2. `run.sh` 使用 mkdir/cp 复用 harness 目录，切换同一 label 的源码 revision 可能残留旧 `native_ownership_test.go` 或其他测试文件。交付的 baseline/fixed 文件集合正确；标准命令未触发此问题。自定义切版应使用干净实验副本，未来宜为每次执行建立全新目录。

列为范围外的行为：同一订阅并发、编号复用、重连代际、内部 RMR 注入、Setup 身份认证、merged 多端点、其他响应 handler、REST 资源回收和远端失败后的清理一致性。理由：属于其他生命周期或信任边界问题；报告明确限制，当前不同订阅 race 结果不能证明这些范围安全。

复核为只读，没有修改文件、索引、HEAD 或分支，也没有操作真实服务。复核结论来自源码与保存的原始证据，不声称复核者独立执行过测试。
