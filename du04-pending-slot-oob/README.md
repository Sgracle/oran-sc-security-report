# o-du-l2：订阅待响应槽位越界写（VUL-DU-04）

- **组件**：o-ran-sc/o-du-l2（验证提交 `2278be373834a0a198e0b5dac94cc66397d88421`）
- **CWE**：CWE-787（越界写入），伴随计数生命周期缺陷
- **结论**：CONFIRMED（隔离实测复现：GCC 直跑 + ASan 双证据）
- **影响**：内存破坏 / 进程崩溃
- **触发者**：RIC 侧订阅流量，**无需并发**（计数只增不减，顺序请求即可）

## 成因

`src/du_app/du_e2ap_mgr.h:39,445`：`RanFunction.pendingSubsRspInfo` 数组固定 5 元素（`MAX_PENDING_SUBSCRIPTION_RSP = 5`）。`procRicSubscriptionRequest()`（`src/du_app/du_e2ap_msg_hdl.c`）在 3520 行检查 `numPendingSubsRsp >= 5` 即置失败跳出，但函数结尾的统一错误清理路径（3607 行）**无条件**执行：

```c
memset(&ranFuncDb->pendingSubsRspInfo[ranFuncDb->numPendingSubsRsp],
       0, sizeof(PendingSubsRspInfo));
```

计数为 5 时对 `pendingSubsRspInfo[5]`（合法下标 0..4）做 64 字节清零，写入数组之后的相邻内存。

加重因素：成功请求 `numPendingSubsRsp++`（3596 行），响应完成路径 `procStatsRspForSubsReq()`（`du_e2ap_mgr.c:832-840`）只清零槽位、**从不递减计数**。同类问题也存在于修改请求路径（`du_e2ap_msg_hdl.c:10094-10116`，`numPendingSubsModRsp` 直接作下标，连容量检查都没有）。

## 触发路径

1. RIC（xApp）向 O-DU 发送 E2AP RIC Subscription Request；
2. 同一 RAN Function 累计处理 5 次订阅（成功或失败均使计数只增不减）；
3. 第 6 个请求到达 → 容量检查失败 → 错误清理路径越界 memset。

## 实测证据

- 普通构建：越界 memset 将紧邻结构体成员清零（哨兵 0x41424344 → 0x00000000），布局检查确认写入目标即相邻成员地址；
- ASan：`heap-buffer-overflow, WRITE of size 64`，位于数组末尾 0 字节偏移。

## 复现

测试源码 `du04_oob_test.c` 与完整复现记录（构建命令、原始输出）见本目录 [REPRODUCTION.md](REPRODUCTION.md)。

## 修复建议

把"有效待响应数量"和"槽位编号"分开：通过占用标记寻找空闲槽并记录本次分配的槽位；只清理本次确实分配的槽；容量满时直接构造资源不足响应、不访问数组；成功、失败、超时和取消路径都释放槽位。只给计数补 `--` 不能处理乱序完成和中间空洞。
