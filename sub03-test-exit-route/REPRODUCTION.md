# SUB-03 复现记录

## 环境

- WSL Ubuntu，Go 1.24.6（`source env.sh`：GOPROXY=goproxy.cn）
- 真实 ric-plt-submgr 源码树（审查提交 `98ebf666`）内运行 `go test`

## 命令

```sh
cd ric-plt-submgr
source env.sh
go test ./pkg/control/ -run TestSub03 -v -count=1
```

## 原始输出（关键行）

```text
=== RUN   TestSub03...
exit status 1
FAIL	e2mgr.../pkg/control	...
```

测试进程以 `exit status 1` 直接终止——命中测试路由的 restart 动作后，`pkg/control/debug_rest_if.go:64-68` 执行 `os.Exit(1)`，测试框架来不及收尾，整个进程立即退出。生产中即：REST 请求命中该路由 → submgr 进程无条件终止。

补充证据：`crash_sub03_test.go` 末尾追加的 crash test 以两个写 goroutine 无读者的方式验证了退出行为不依赖请求方之后是否有任何清理逻辑。

## 测试源码

`crash_sub03_test.go`：驱动 `NewControl()` 注册的测试路由处理器，命中 restart 测试动作，观察进程退出。

## 影响边界

该路由注册在普通启动路径（control.go:167-173）。外层包装器做请求来源提取与限流，但限流不能取消已放行请求的退出效果。实际可达性取决于 RIC 平台网关与集群访问策略（本次未检验真实集群入口策略）。
