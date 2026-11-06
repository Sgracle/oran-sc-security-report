# O-RAN SC Security Reports

O-RAN SC 安全研究与协调披露材料，按问题分别归档。整理日期：2026-11-06。

## 已归档问题

### Appmgr：镜像来源真实性

| 材料包 | 研究对象与问题 | 已有证据 | 尚待完成 | 协调记录 |
|---|---|---|---|---|
| [Appmgr：镜像来源真实性](appmgr/README.md) | `ric-plt-appmgr / xapp_onboarder`：描述符、发布者与镜像内容之间的信任绑定 | 固定提交的源码审计与官方测试预期 | 受支持架构的信任边界、真实服务及部署复现 | [RIC-1078](https://lf-o-ran-sc.atlassian.net/browse/RIC-1078)，MITRE 申请跟踪号 `CAN-2026-2039411` |

### Submgr：跨 E2 节点订阅归属

| 材料包 | 研究对象与问题 | 已有证据 | 尚待完成 | 协调记录 |
|---|---|---|---|---|
| [Submgr：跨 E2 节点订阅归属](submgr/OVERVIEW.zh-CN.md) | `ric-plt-submgr`：Delete Required 处理缺少来源节点与订阅所有者绑定 | 固定提交的源码审计、原函数隔离 mock、前后对照及本地补丁 | 真实 APER/SCTP/RMR 链路、发布镜像及完整原生回归 | [RIC-1079](https://lf-o-ran-sc.atlassian.net/browse/RIC-1079)，本仓库 [Issue #1](https://github.com/Sgracle/oran-sc-security-report/issues/1) |

### Crash Validation：三组件崩溃/资源缺陷（2026-11-06 隔离实测验证）

对象为 2026-10-27 对 `ric-plt-submgr`、`ric-plt-e2mgr`、`o-du-l2` 三仓库静态审查 9 项发现中的 7 项，
全部在隔离环境完成实测复现或证伪。**每个漏洞一个文件夹**：

| 目录 | 组件 | CWE | 结论 | 类型 |
|---|---|---|---|---|
| [du04-pending-slot-oob](du04-pending-slot-oob/README.md) | o-du-l2 | CWE-787 | CONFIRMED | 槽位越界写（64 字节 memset，计数只增不减、无需并发） |
| [du03-failure-response-leak](du03-failure-response-leak/README.md) | o-du-l2 | CWE-401 | CONFIRMED | 失败响应清理函数误用分配宏，每请求泄漏 6 次分配 |
| [mgr01-config-update-panic](mgr01-config-update-panic/README.md) | ric-plt-e2mgr | CWE-248 | CONFIRMED（panic 变体）/ REFUTED（写错条目变体） | 配置更新索引越界 panic，进程崩溃 DoS |
| [mgr02-global-map-race](mgr02-global-map-race/README.md) | ric-plt-e2mgr | CWE-362 | CONFIRMED | 跨锁竞态 + 无锁全局 map，Go 原生 fatal error |
| [sub01-state-map-race](sub01-state-map-race/README.md) | ric-plt-submgr | CWE-362 | CONFIRMED | 状态回调并发 map 读写，Go 原生 fatal error |
| [sub02-registry-lock-bypass](sub02-registry-lock-bypass/README.md) | ric-plt-submgr | CWE-362 | CONFIRMED | 断连清理绕过注册表锁 |
| [sub03-test-exit-route](sub03-test-exit-route/README.md) | ric-plt-submgr | CWE-489 | CONFIRMED | 生产启动注册测试路由，命中即 `os.Exit(1)` |

证据强度：Go 项在真实上游源码树内以 `go test -race` 验证，并发项均取得 Go runtime 原生
`fatal error`（进程级、非概率性）证据；C 项以复刻生产函数体与宏的隔离程序验证，GCC 直跑 +
ASan/LSan 双证据。静态审查中尚未实测的候选（E2-01、DU-01、DU-02）未归档，见根级报告附录 A。
截至 2026-11-06 本批漏洞尚未向 O-RAN SC 上游或任何 CNA 发送报告，无 CVE/CNVD 编号分配。

## 阅读入口

**Appmgr / xApp 制品真实性**

- [中文概览与状态](appmgr/README.md)
- [英文审计报告](appmgr/RIC-1078-appmgr-preliminary-report.md)
- [中文 CNVD 历史草稿](appmgr/CNVD-draft.zh-CN.md)：仍含待填项和未执行步骤，尚未向 CNVD 发送。

**Submgr / E2 订阅所有权**

- [中文概览](submgr/OVERVIEW.zh-CN.md) · [原始材料包说明](submgr/README.md)
- [英文漏洞报告](submgr/reports/submgr.md) · [规范与影响范围复核](submgr/reports/standards-and-scope.md)
- [证据摘要](submgr/evidence/SUMMARY.md) · [隔离测试说明](submgr/tests/README.md)
- [本地建议补丁](submgr/patches/ownership-fix.patch) · [协调记录](submgr/COORDINATION.md)

**Crash Validation / 三组件崩溃与资源缺陷**

- [根级验证记录（CVE 上报说明 + 复现命令与原始输出）](crash-validation-20261106.md)
- 各漏洞详情见上表对应目录的 README（成因、触发路径、实测证据、复现、修复建议）

## 目录

```text
appmgr/
  README.md                             中文概览、边界、状态
  RIC-1078-appmgr-preliminary-report.md  英文原始报告（2026-10-28）
  CNVD-draft.zh-CN.md                    中文历史草稿（2026-10-15）
submgr/
  OVERVIEW.zh-CN.md                      中文概览与阅读路线
  README.md                             原始材料包说明
  reports/                              英文报告、规范与范围
  evidence/                             历史证据及来源清单
  tests/                                baseline / fixed 独立测试模块
  patches/                              本地建议补丁
  tools/                                提取与校验工具
  verification/                         2026-11-03 打包复核结果
  LICENSES/                             保留的原始许可说明
  BUNDLE-MANIFEST.json                  原始 Submgr 文件哈希
  COORDINATION.md / PROVENANCE.md        协调与来源
du04-pending-slot-oob/README.md          o-du-l2 订阅待响应槽位越界写
du03-failure-response-leak/README.md     o-du-l2 失败响应清理误用分配宏
mgr01-config-update-panic/README.md      e2mgr 配置更新索引越界 panic
mgr02-global-map-race/README.md          e2mgr 全局 map 同步失效
sub01-state-map-race/README.md           submgr 状态回调并发 map 读写
sub02-registry-lock-bypass/README.md     submgr 断连清理绕过注册表锁
sub03-test-exit-route/README.md          submgr 生产暴露测试退出入口
crash-validation-20261106.md             Crash Validation 根级报告（CVE 说明 + 复现记录）
REORGANIZATION.md                        路径迁移及证据完整性说明
```

## 运行 Submgr 隔离测试

从仓库根目录进入 Submgr 包后，按[原始测试说明](submgr/tests/README.md)运行：

```sh
cd submgr
unshare -n -- bash tools/verify-bundle.sh
```

需要 Linux/WSL、Go 和创建网络命名空间的权限。基线用例预期失败，修复后用例预期通过。
本次仅整理目录、复核文件和链接，没有重新执行测试；历史结果仍保留原始日期。

## 归档与协调

各原始报告和材料包保持原内容；概览只提供导航和证据分层。详见[路径迁移说明](REORGANIZATION.md)。
`submgr/BUNDLE-MANIFEST.json` 仅覆盖原始 Submgr 包，不是整个仓库的总清单。

本仓库为私有工作记录。上传、整理或改名不等于向上游安全团队发送报告、
邀请维护者访问或获得漏洞确认。
