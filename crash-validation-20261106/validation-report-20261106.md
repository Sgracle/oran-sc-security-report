# O-RAN 缺陷验证实验记录（2026-11-06）

对应评审文档：`D:\309\graduation\docs\code-review\oran-crash-review-2026-10-27.md`
实验环境：WSL Ubuntu（Go 1.24.6 / GCC 13.3），隔离源码副本
- Go 组件副本：`/root/oran-lab/experiments/crash-validation-20261105/ric-plt-{submgr,e2mgr}`
- C 组件副本：`D:\309\graduation\tmp\crash-review-20261027\o-du-l2`
- C 隔离测试：`/root/oran-lab/experiments/crash-validation-20261105/du-test/`
- Go 环境脚本：`source /root/oran-lab/experiments/crash-validation-20261105/env.sh`（GOPROXY=goproxy.cn）
- 全程未触碰运行中的 RIC 栈（submgr/e2term/e2mgr 6.0.x）。

## 结论总表

| 编号 | 组件 | 结论 | 关键证据 |
|------|------|------|----------|
| SUB-01 | ric-plt-submgr | CONFIRMED | 并发 NotificationCb → `fatal error: concurrent map writes`；race 报告 e2if_state.go:110 写 vs :196 读 |
| SUB-02 | ric-plt-submgr | CONFIRMED | DeleteAllE2Subscriptions 无锁遍历/删除注册表；race 报告 registry.go:630 vs 651/652 |
| SUB-03 | ric-plt-submgr | CONFIRMED | 测试路由命中指定动作时 `os.Exit(1)`，进程直接退出 |
| MGR-01（panic 变体） | ric-plt-e2mgr | CONFIRMED | 真实生产 update 循环 panic：`index out of range [1] with length 1`（生产日志 "item at position [1] should be updated" 后越界） |
| MGR-01（写错条目变体） | ric-plt-e2mgr | **REFUTED** | 生产比较逻辑（接口类型 + GetAmfName 两条件）行为正确，不匹配条目被跳过（连打 "didn't match"），无覆写错条目现象 |
| MGR-02 | ric-plt-e2mgr | CONFIRMED | ① ProcedureMap 跨锁竞态：race 报告写方 `models.UpdateProcedureType`（e2_error_indication_message.go:30，包级锁）vs 读方 `HandleBasedOnProcedureType`（error_indication_notification_handler.go:116，handler 自带锁）；② `ExistingRanFunctiuonsMap` 零锁并发 → 原生 `fatal error: concurrent map read and map write`（0.009s 内进程崩溃） |
| DU-03 | o-du-l2 | CONFIRMED | 原样复刻生产函数 + DU_ALLOC/DU_FREE 宏（计数内存池）：1000 次失败响应 → allocs=6000, frees=0，每周期净泄漏 6 块；LeakSanitizer 独立确认多批直接泄漏 |
| DU-04 | o-du-l2 | CONFIRMED | 槽位满（numPendingSubsRsp=5）后错误路径 `memset(&pendingSubsRspInfo[5])` 越界：普通构建实测把相邻结构体成员清零（哨兵 0x41424344→0）；ASan 报 `heap-buffer-overflow WRITE of size 64`（恰好一个 PendingSubsRspInfo），偏移在数组末尾 0 字节处 |

## 复现命令

```bash
# SUB-01/02/03（此前已完成，测试位于 ric-plt-submgr 各包）
source /root/oran-lab/experiments/crash-validation-20261105/env.sh
go test <对应包> -run TestSub -v -count=1 -race

# MGR-01（-vet=off 用于绕过 Go 1.24 对老代码 4 处 Errorf 非常量格式串的 vet 报错，与本缺陷无关）
cd /root/oran-lab/experiments/crash-validation-20261105/ric-plt-e2mgr/E2Manager
go test ./handlers/rmrmsghandlers/ -run TestMgr01 -v -count=1 -vet=off

# MGR-02
go test ./handlers/rmrmsghandlers/ -run TestMgr02 -v -count=1 -race -vet=off
# 无 race detector 的原生崩溃复现：
go test ./handlers/rmrmsghandlers/ -run TestMgr02ExistingRanFunctionsMapUnlocked -v -count=1 -vet=off

# DU-03 / DU-04
cd /root/oran-lab/experiments/crash-validation-20261105/du-test
./du03_plain   # 计数器汇总：cycles=1000 allocs=6000 frees=0
./du03_asan    # LeakSanitizer 直接泄漏报告
./du04_plain   # 哨兵被清零 + 布局检查确认越界目标即相邻成员
./du04_asan    # heap-buffer-overflow WRITE of size 64
```

## 对评审文档的修正建议

1. **MGR-01**：崩溃路径（越界 panic）成立；但"找到第 j 项却写第 i 项导致覆写错误条目"的静默数据破坏**不成立**——生产比较条件是"接口类型相等 AND AmfName 相等"，不匹配即跳过，不会写入。建议将文档中该子场景改为"更新请求中存在不匹配条目时跳过；仅当更新列表长于现有列表时越界 panic"。
2. **MGR-02 / SUB-01 / SUB-02**：均按文档描述成立，其中 ExistingRanFunctiuonsMap 与 e2if_state 的并发访问可直接 `fatal error` 杀死整个进程（不依赖 race detector）。
3. **DU-03**：确认泄漏量级为"每次订阅拒绝泄漏 6 次分配"（含 IE 数组整体），长期运行会耗尽内存池，与文档"非单次必崩、长期 OOM"的定性一致。

## 测试产物清单

- `ric-plt-e2mgr/E2Manager/handlers/rmrmsghandlers/crash_mgr01_test.go`（panic + wrong-entry 两用例）
- `ric-plt-e2mgr/E2Manager/handlers/rmrmsghandlers/crash_mgr02_test.go`（跨锁竞态 + 无锁 map 两用例）
- `du-test/du03_leak_test.c`、`du-test/du04_oob_test.c`（含 plain / asan 两种构建）
- submgr 侧测试（SUB-01/02/03）见 ric-plt-submgr 对应包内 crash_*_test.go
