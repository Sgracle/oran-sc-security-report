# 实测回归结果

统计叶子用例，不重复计入父级 Test。

| 用例 | 修复前 | 修复后 |
|---|---|---|
| `TestCleanupDoesNotRequireRemoteSuccess/failure` | pass | pass |
| `TestCleanupDoesNotRequireRemoteSuccess/timeout` | pass | pass |
| `TestCleanupFailureCharacterization` | pass | pass |
| `TestConcurrentDifferentSubscriptions` | pass | pass |
| `TestConcurrentForeignAndOwn` | fail | pass |
| `TestIdentityValidation/empty_source` | fail | pass |
| `TestIdentityValidation/nil_owner` | fail | pass |
| `TestIdentityValidation/nil_source` | fail | pass |
| `TestIdentityValidation/nil_subscription_config` | fail | pass |
| `TestIdentityValidation/wrong_function` | fail | pass |
| `TestIdentityValidation/wrong_requestor` | fail | pass |
| `TestMixedBatch` | fail | pass |
| `TestOwnership/A_own` | pass | pass |
| `TestOwnership/A_references_B` | fail | pass |
| `TestOwnership/B_own` | pass | pass |
| `TestOwnership/B_references_A` | fail | pass |
| `TestOwnership/unknown_instance` | pass | pass |
| `TestRejectedMalformedOrOngoing/malformed_payload` | pass | pass |
| `TestRejectedMalformedOrOngoing/nil_params` | fail | pass |
| `TestRejectedMalformedOrOngoing/ongoing_delete` | pass | pass |
| `TestSubscriptionMatchesDeleteRequired/empty_names` | 未加入基线 | pass |
| `TestSubscriptionMatchesDeleteRequired/foreign_A` | 未加入基线 | pass |
| `TestSubscriptionMatchesDeleteRequired/missing_config` | 未加入基线 | pass |
| `TestSubscriptionMatchesDeleteRequired/missing_owner` | 未加入基线 | pass |
| `TestSubscriptionMatchesDeleteRequired/missing_source` | 未加入基线 | pass |
| `TestSubscriptionMatchesDeleteRequired/own_B` | 未加入基线 | pass |
| `TestSubscriptionMatchesDeleteRequired/wrong_function` | 未加入基线 | pass |
| `TestSubscriptionMatchesDeleteRequired/wrong_instance` | 未加入基线 | pass |
| `TestSubscriptionMatchesDeleteRequired/xapp_requestor_is_not_wire_requestor` | 未加入基线 | pass |

基线失败表示安全期望未满足；它不是执行器异常。nil owner/config 用例捕获的 panic 另见原始日志。

- baseline: 9/20 通过，11 失败，退出码 1。
- fixed: 29/29 通过，0 失败，退出码 0。

原生匹配单元测试仅在 fixed 夹具中执行；包导入替换为本地测试 DTO。完整原生包测试缺依赖，详见 native-suite.log。
