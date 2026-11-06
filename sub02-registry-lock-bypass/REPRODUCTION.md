# SUB-02 复现记录

## 环境

- WSL Ubuntu，Go 1.24.6（`source env.sh`：GOPROXY=goproxy.cn）
- 真实 ric-plt-submgr 源码树（审查提交 `98ebf666`）内运行 `go test -race`

## 命令

```sh
cd ric-plt-submgr
source env.sh
go test ./pkg/control/ -run TestSub02 -v -count=1 -race
```

## 原始输出（关键行）

```text
WARNING: DATA RACE
Read at ... by goroutine N:
      pkg/control/registry.go:630        # DeleteAllE2Subscriptions 无锁遍历 r.register
Previous write at ... by goroutine M:
      pkg/control/registry.go:651        # 无锁 delete
      pkg/control/registry.go:652        # subIds 回收
```

race 双方均在 `DeleteAllE2Subscriptions()` 内部：遍历读取（630）与删除+ID 回收（651/652）与持锁的常规创建/删除路径（registry.go:356-361）并发时同样成立。断连回调调用点（e2if_state.go:119-136）不代持 `r.mutex`，是该无锁路径的入口。

## 测试源码

`crash_sub02_test.go`：并发驱动 `DeleteAllE2Subscriptions()`（断连清理）与订阅创建/删除（持锁路径），覆盖同 RAN 与跨 RAN 组合，并检查 subIds 回收竞争。
