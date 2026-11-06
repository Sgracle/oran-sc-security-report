# o-du-l2：失败响应清理函数误用分配宏，持续内存泄漏（VUL-DU-03）

- **组件**：o-ran-sc/o-du-l2（验证提交 `2278be373834a0a198e0b5dac94cc66397d88421`）
- **CWE**：CWE-401（内存泄漏），伴随 CWE-770（资源分配无限制）
- **结论**：CONFIRMED（隔离实测复现：计数内存池 + LeakSanitizer 双证据）
- **影响**：内存池耗尽 / OOM（非单次必崩，长期累积）
- **触发者**：RIC 侧订阅拒绝流量，每请求必然泄漏

## 成因

`FreeRicSubscriptionFailure()`（`src/du_app/du_e2ap_msg_hdl.c:3630-3652`）名为 Free，实际四处全部调用 `DU_ALLOC` 而非 `DU_FREE`（宏定义见 `src/du_app/du_utils.h:67-92`）。`DU_ALLOC` 会**分配新内存并覆盖指针**：原对象从此不可达，新对象也永远不会被释放。四处分别是：每个 IE（循环内）、IE 数组整体、`unsuccessfulOutcome`、`e2apMsg` 本身。

该函数在 `BuildAndSendRicSubscriptionFailure()` 收尾路径（`du_e2ap_msg_hdl.c:3782`）调用——**每一个被拒绝的订阅请求**都会走到这里。

## 触发路径

1. RIC/xApp 发送会被 O-DU 拒绝的 RIC Subscription Request（无效 RAN Function ID、资源不足等常规失败场景）；
2. O-DU 构建并发送 RIC Subscription Failure；
3. 收尾时反向泄漏：每次泄漏 4+ 次池分配（IE 数量越多越多）；
4. 持续失败流量逐步耗尽 DU 自定义内存池 / 触发 OOM。

## 实测证据

1000 次"构建失败响应→释放"循环：`allocs=6000, frees=0`（每次调用净泄漏 6 次分配）；LeakSanitizer 独立报出同规模直接泄漏。泄漏量与失败响应次数严格线性增长。

## 复现

测试源码与命令见 [验证实验记录](../crash-validation-20261106.md)。核心：复刻生产函数体与宏的隔离 C 程序（`du-test/du03_leak_test.c`），用计数内存池替代 `SGetSBuf/SPutSBuf`。

## 修复建议

四处 `DU_ALLOC` 改为对应 `DU_FREE`，保留逐层清理顺序和空指针保护。必须保持 DU 内存池分配与释放成对，不能直接替换为 libc `free()`。
