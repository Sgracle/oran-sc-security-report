# ric-plt-submgr：断连清理绕过注册表锁（VUL-SUB-02）

- **组件**：o-ran-sc/ric-plt-submgr（验证提交 `98ebf666e797ee02a7b2946742ebcc4259252b4c`；运行栈 v0.10.1）
- **CWE**：CWE-362（并发竞争）
- **结论**：CONFIRMED（真实源码树内复现：race detector）
- **影响**：进程崩溃 DoS + 订阅 ID 重复分配等状态损坏
- **触发者**：某一 RAN 断连回调 × 任意 RAN 的订阅创建/删除

## 成因

`pkg/control/registry.go:627-673` 的 `DeleteAllE2Subscriptions()` 遍历并删除 `r.register` 和 `r.restSubscriptions`、回收 `r.subIds`，但**全程不获取 `r.mutex`**；调用点（`pkg/control/e2if_state.go:119-136` 的断连回调）也不代持。常规订阅创建/删除/查询路径均持 `r.mutex`（如 registry.go:356-361），因此清理函数与它们之间没有任何互斥。

"跳过正在处理的订阅"只判断条目状态，保护不了 map 本身，也保护不了其他 RAN 的条目。

## 触发路径

1. 某一 RAN 断连，触发 `DeleteAllE2Subscriptions()` 无锁遍历/删除注册表；
2. 同时另一 xApp 对任意 RAN（包括其他 RAN）创建或删除订阅（持锁路径）；
3. 无锁遍历与持锁写并发 → Go fatal error 或数据竞争 → submgr 进程退出；`subIds` ID 池竞争还可能造成订阅 ID 重复分配。

## 实测证据

race 报告：registry.go:630（无锁遍历 `r.register`）vs 651/652（无锁 delete + subIds 回收）。

## 复现

测试源码 `crash_sub02_test.go` 与完整复现记录（命令、race 报告原始输出）见本目录 [REPRODUCTION.md](REPRODUCTION.md)。

## 修复建议

统一注册表的锁规则：锁内选择并标记待清理对象，释放锁后做路由和 SDL I/O；再次加锁时校验对象身份/版本后移除和回收 ID，防止误删期间新建的订阅；禁止在未同步的 map 上遍历。若先采用整段加锁的最小修复，应检查持锁网络调用的阻塞与锁顺序问题。
