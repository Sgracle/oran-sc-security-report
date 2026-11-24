# CNVD 报送材料

本目录保存面向 CNVD 表单整理的中文报告。两份都是从本仓库已归档的实测发现转写而来，
字段按 CNVD 常见填报项组织（产品、版本、类型、位置、复现、影响、修复建议）。

| 文件 | 对应发现 | 证据级别 | 结论 |
|---|---|---|---|
| [CNVD-SUBNEW1-submgr-rmr-dos.zh-CN.md](CNVD-SUBNEW1-submgr-rmr-dos.zh-CN.md) | [submgr-rmr-crash](../submgr-rmr-crash/README.md) | 真实容器实测，两次独立复现（SIGSEGV，PC 一致） | CONFIRMED：未认证 RMR 消息致进程崩溃，远程 DoS |
| [CNVD-DU04-o-du-l2-slot-oob.zh-CN.md](CNVD-DU04-o-du-l2-slot-oob.zh-CN.md) | [du04-pending-slot-oob](../du04-pending-slot-oob/README.md) | 隔离程序复刻生产控制流，普通构建 + ASan 双证据 | CONFIRMED：订阅待响应槽位越界写（64 字节 memset） |

## 证据级别差异（审核时请注意）

- **SUBNEW1** 是真实组件证据：打到真实 `ric-plt-submgr:0.10.1` 容器的真实 RMR 端口，
  崩溃日志为容器原始输出。但只证明了“崩溃”这一事实，未证明可利用为代码执行。
- **DU04** 是源码定位 + 隔离复现：`du04_oob_test.c` 复刻了生产函数的检查与错误清理路径，
  并在 ASan 下取得 heap-buffer-overflow。**未**在完整 O-DU 运行实例上复现端到端崩溃，
  也未确认越界写入在真实结构体布局下损坏哪个字段。报告正文已如实标注该边界。

## 提交状态

两份均未向 CNVD 提交，无 CNVD 编号、无 CVE 编号。`appmgr/CNVD-draft.zh-CN.md` 是更早的
第三份草稿（appmgr 制品来源真实性），证据仅为源码审计，**不在本次两份之内**。

提交前需补的通用项：报告人姓名与联系方式、发现日期确认、是否已向其他平台/上游报送的声明、
以及按页面实际字段逐项映射（本目录给出的是字段建议，不是登录后表单的逐字说明）。
