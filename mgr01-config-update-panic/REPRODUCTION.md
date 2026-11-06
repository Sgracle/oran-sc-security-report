# MGR-01 复现记录

## 环境

- WSL Ubuntu，Go 1.24.6（`source env.sh`：GOPROXY=goproxy.cn）
- 真实 ric-plt-e2mgr 源码树（审查提交 `353939c0`）内运行 `go test`

## 命令

```sh
cd ric-plt-e2mgr/E2Manager
source env.sh   # Go 代理等环境
go test ./handlers/rmrmsghandlers/ -run TestMgr01 -v -count=1 -vet=off
```

`-vet=off` 仅用于绕过 Go 1.24 对仓库既有 4 处 `Errorf` 非常量格式串的 vet 报错，与本缺陷无关。

## 原始输出（关键行）

用例 1：越界 panic（真实生产路径）：

```text
=== RUN   TestMgr01UpdateIndexPanic
{"...","msg":"#E2nodeConfigUpdateNotificationHandler.Handle - item at position [1] should be updated"}
    crash_mgr01_test.go:87: PANIC reproduced as predicted: runtime error: index out of range [1] with length 1
--- PASS: TestMgr01UpdateIndexPanic (0.00s)
```

生产日志先打出 "item at position [1] should be updated"，随后真实生产循环 panic。

用例 2：静默写错条目变体（被证伪）：

```text
{"...","msg":"#E2nodeConfigUpdateNotificationHandler.Handle - dint match"}   （连续 4 条）
    crash_mgr01_test.go:113: after update: NodeConfigs[0].AmfName="AMF-A" (was AMF-A), NodeConfigs[1].AmfName="AMF-OLD" (was AMF-OLD)
    crash_mgr01_test.go:117: unexpected state: got0="AMF-A" got1="AMF-OLD"
--- FAIL: TestMgr01UpdateWrongEntry (0.00s)
```

所有不匹配条目均被跳过（"didn't match"），两个条目原值保持不变——不存在覆写错误条目的静默数据破坏。

## 测试源码

`crash_mgr01_test.go`：直接调用生产 `compareConfigIDs` + 原样复刻 `handleUpdateConfig` 的生产更新循环（e2_node_config_update_notification_handler.go:134-150），两个用例分别为越界场景与"匹配第 2 项"场景。
