# Submgr：跨 E2 节点订阅归属问题（RIC-1079）

研究对象：O-RAN SC `ric-plt-submgr` 的 `RIC Subscription Delete Required` 处理。
本概览整理于 2026-11-03；原始报告、代码、日志和清单保留其原始内容与日期。

## 已观察到什么

审查的处理函数按全局 Instance ID 选择订阅，但在调度删除前未将来源节点
`params.Meid` 与该订阅的所有者 `subs.Meid` 绑定。后续流程从被选中的订阅
重建目标节点及删除标识，因此节点 A 的输入在隔离测试中可以选中 B 的订阅，
产生发往 B 的删除输出并清理 B 的模拟管理状态。

这与 Appmgr 的镜像签名问题不同；本目录的结果只适用于 Submgr。

## 证据强度与限制

| 层次 | 现有材料支持的结论 |
|---|---|
| 固定源码快照 | 记录的 H/I/J/L/M/master 快照中关键处理函数内容一致；不等于已逐一验证所有发布镜像 |
| I-release 原函数隔离测试 | 使用原函数体、替代 DTO、JSON codec 和 mock 外部服务，观察了错误关联与模拟清理 |
| 本地建议修复 | 原共享用例从 9/20 变为 20/20；另有 9 个 matcher 用例，修复包共 29/29 |
| 真实协议与产品部署 | APER/SCTP/RMR 全链路、发布镜像与完整原生回归尚未完成 |

`RemoteActive` 是合成测试状态。完整原生测试曾因离线依赖缺失无法编译，
相关记录在材料包内；失败/超时清理只是行为刻画，不是另一个已确认漏洞。
没有宣称用户面中断、任意代码执行、厂商确认、正式 CVSS 或已分配 CVE。
本次整理没有重跑实验。

## 阅读路线

1. [英文完整报告](reports/submgr.md)：问题、固定提交、调用链、前提和影响限制。
2. [规范与影响范围复核](reports/standards-and-scope.md)：区分规范要求、SC 实现和其他候选线索。
3. [历史证据摘要](evidence/SUMMARY.md)及[机器可读摘要](evidence/summary.json)。
4. [本地建议补丁](patches/ownership-fix.patch)：未获上游合入。
5. [隔离测试说明](tests/README.md)、[来源与许可](PROVENANCE.md)、
   [原始哈希清单](BUNDLE-MANIFEST.json)。
6. [协调记录](COORDINATION.md)：Jira、私下报送草稿和当前材料边界。

[原始英文包说明](README.md)也完整保留。

## 运行位置

从整个仓库根目录：

```sh
cd submgr
unshare -n -- bash tools/verify-bundle.sh
```

从本目录执行时，只运行第二行。详细依赖与预期退出码见[测试说明](tests/README.md)。
工具仍以本目录为包根目录；`evidence/` 是历史证据，`verification/` 是打包复核输出。

## 协调状态

- 项目工单：[RIC-1079](https://lf-o-ran-sc.atlassian.net/browse/RIC-1079)。
- 本仓库跟踪：[Issue #1](https://github.com/Sgracle/oran-sc-security-report/issues/1)。
- 2026-11-03 打包记录中，报告人称 Jira 尚无回复；当次未重新查询 Jira。
- 私有归档和本地建议补丁不等于上游已收到、接受或修复该问题。
- Appmgr 的 `CAN-2026-2039411` 不适用于本问题。

[返回仓库首页](../README.md)
