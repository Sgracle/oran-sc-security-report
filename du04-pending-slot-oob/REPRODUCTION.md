# DU-04 复现记录

## 环境

- WSL Ubuntu，GCC 13.3，ASan（libasan）
- 隔离 C 程序复刻生产控制流，不依赖完整 O-DU 构建

## 构建

```sh
gcc -g -O0 -fsanitize=address -o du04_asan du04_oob_test.c
gcc -g -O0 -o du04_plain du04_oob_test.c
```

## 命令

```sh
./du04_plain   # 观察相邻成员被清零
./du04_asan    # ASan 越界报告
```

## 原始输出

普通构建（关键行）：

```text
before: numPendingSubsRsp=5 sentinel=0x41424344 array_end=0x...13e4 sentinel=0x...13e4
after : ret=RFAILED, sentinel=0x00000000
layout check: memset target IS the adjacent member address
OBSERVED DAMAGE: adjacent struct member zeroed by the OOB memset
```

`array_end`（即 `pendingSubsRspInfo[5]` 的地址）与相邻哨兵成员地址完全相同，且该成员被越界 memset 清零。

ASan 构建：

```text
ERROR: AddressSanitizer: heap-buffer-overflow on address 0x513000000188
WRITE of size 64 at 0x513000000188 thread T0
    #0 ... in memset
    #1 ... in procRicSubsReq du04_oob_test.c:39
0x513000000188 is located 0 bytes after 328-byte region
SUMMARY: AddressSanitizer: heap-buffer-overflow ... in memset
```

`WRITE of size 64` 恰为一个 `PendingSubsRspInfo`（64 字节），偏移在数组末尾 0 字节处。

## 测试源码

`du04_oob_test.c`：结构体布局镜像 `RanFunction`（du_e2ap_mgr.h:445），检查与错误路径 memset 按 `du_e2ap_msg_hdl.c:3520/3607` 原样复刻。
