# Crash Validation：三组件崩溃/资源缺陷的隔离实测验证

整理日期：2026-11-06。本材料包对应 2026-10-27 对 `ric-plt-submgr`、`ric-plt-e2mgr`、`o-du-l2`
三仓库的静态崩溃风险审查（9 项发现）的**本地隔离环境实测验证**阶段。

## 结论速览

| 编号 | 组件 | 结论 | 类型 |
|---|---|---|---|
| VUL-DU-04 | o-du-l2 | CONFIRMED | CWE-787 槽位越界写（64 字节 memset，计数只增不减、无需并发） |
| VUL-DU-03 | o-du-l2 | CONFIRMED | CWE-401 失败响应清理函数误用分配宏，每请求泄漏 6 次分配 |
| VUL-MGR-01 | ric-plt-e2mgr | CONFIRMED（panic 变体）/ REFUTED（写错条目变体） | CWE-248 更新列表长于现有配置时索引越界 panic，goroutine 无 recover |
| VUL-MGR-02 | ric-plt-e2mgr | CONFIRMED | CWE-362 跨锁竞态 + 无锁全局 map，原生 `fatal error` 杀进程 |
| VUL-SUB-01 | ric-plt-submgr | CONFIRMED | CWE-362 状态回调并发 map 读写，原生 `fatal error` |
| VUL-SUB-02 | ric-plt-submgr | CONFIRMED | CWE-362 断连清理绕过注册表锁 |
| VUL-SUB-03 | ric-plt-submgr | CONFIRMED | CWE-489 生产启动注册测试路由，命中即 `os.Exit(1)` |

静态审查中尚未实测的候选（E2-01、DU-01、DU-02）未纳入本批上报范围，见 cve 报告附录 A。

## 阅读入口

- [CVE 上报说明（中）](cve-report-20261106.md)：按漏洞逐项给出 CWE、成因、触发路径、实测证据与汇总表
- [验证实验记录（中）](validation-report-20261106.md)：复现命令、原始输出、对原评审文档的修正建议

## 证据边界

- Go 项（SUB-01/02/03、MGR-01/02）在真实上游源码树内以 `go test`（含 `-race`）验证；
  其中并发项均取得 Go runtime 原生 `fatal error`（进程级、非概率性）证据。
- C 项（DU-03/04）以复刻生产函数体与宏的隔离程序验证，GCC 直跑 + ASan/LSan 双证据。
- 全部验证于 WSL Ubuntu 隔离副本完成，未触碰运行中的 RIC 栈；复现环境与命令见验证实验记录。
- 本材料包记录的是验证结果与待上报内容；截至整理日尚未向 O-RAN SC 上游或任何 CNA 发送报告，
  亦无 CVE/CNVD 编号分配。
