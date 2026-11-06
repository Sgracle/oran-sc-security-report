# DU-03 复现记录

## 环境

- WSL Ubuntu，GCC 13.3，ASan/LSan
- 隔离 C 程序：函数体与 DU_ALLOC/DU_FREE 宏按生产源码原样复刻，用计数内存池替代 `SGetSBuf/SPutSBuf`

## 构建

```sh
gcc -g -O0 -fsanitize=address -o du03_asan du03_leak_test.c
gcc -g -O0 -o du03_plain du03_leak_test.c
```

## 命令

```sh
./du03_plain   # 内存池计数汇总
./du03_asan    # LeakSanitizer 独立泄漏报告
```

## 原始输出

普通构建（计数内存池）：

```text
cycles=1000 allocs=6000 frees=0 leaked_allocs=6000 (per cycle: 6.0 alloc vs 0.0 free)
LEAK CONFIRMED: every failure response leaks its allocations (DU_ALLOC used where DU_FREE intended)
```

1000 次"构建失败响应→释放"调用：6000 次分配、0 次释放，每次调用净泄漏 6 次池分配
（3 个 IE + IE 数组整体 + unsuccessfulOutcome + e2apMsg 本身），泄漏量与调用次数严格线性。

ASan 构建（LeakSanitizer）：

```text
ERROR: LeakSanitizer: detected memory leaks
Direct leak of 24000 byte(s) in 1000 object(s) ...
Direct leak of 8000 byte(s) in 1000 object(s) ...
（多批直接泄漏，全部指向 one_request_cycle 的构造点，无对应释放）
```

## 测试源码

`du03_leak_test.c`：`FreeRicSubscriptionFailure` 函数体复刻自 `du_e2ap_msg_hdl.c:3636-3652`，
宏复刻自 `du_utils.h:67-92`（保留分配语义），ASN.1 结构按 `protocolIEs.list.*` 层级仿制。
