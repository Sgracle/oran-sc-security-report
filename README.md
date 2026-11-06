# O-RAN SC Security Reports

O-RAN SC 安全研究与协调披露材料，按问题分别归档。整理日期：2026-11-03。

## 两个问题

| 材料包 | 研究对象与问题 | 已有证据 | 尚待完成 | 协调记录 |
|---|---|---|---|---|
| [Appmgr：镜像来源真实性](appmgr/README.md) | `ric-plt-appmgr / xapp_onboarder`：描述符、发布者与镜像内容之间的信任绑定 | 固定提交的源码审计与官方测试预期 | 受支持架构的信任边界、真实服务及部署复现 | [RIC-1078](https://lf-o-ran-sc.atlassian.net/browse/RIC-1078)，MITRE 申请跟踪号 `CAN-2026-2039411` |
| [Submgr：跨 E2 节点订阅归属](submgr/OVERVIEW.zh-CN.md) | `ric-plt-submgr`：Delete Required 处理缺少来源节点与订阅所有者绑定 | 固定提交的源码审计、原函数隔离 mock、前后对照及本地补丁 | 真实 APER/SCTP/RMR 链路、发布镜像及完整原生回归 | [RIC-1079](https://lf-o-ran-sc.atlassian.net/browse/RIC-1079)，本仓库 [Issue #1](https://github.com/Sgracle/oran-sc-security-report/issues/1) |
| [Crash Validation：三组件崩溃/资源缺陷实测](crash-validation-20261106/README.md) | `ric-plt-submgr` / `ric-plt-e2mgr` / `o-du-l2`：2026-10-27 静态审查 9 项发现的隔离环境复现与证伪 | 真实源码树内 `go test -race`、Go 原生 `fatal error`、ASan/LSan 实测复现 | 向上游/CNA 发送报告及编号申请 | 无（截至 2026-11-06 尚未发送） |

两个问题的证据强度不同。Submgr 的 mock 测试和修复结果不构成 Appmgr 的验证证据。
目前未在这些材料中确认正式 CVE 分配、厂商认可或确定风险评分。
Appmgr 的 MITRE 状态最后核对于 2026-10-20，当时仍在审核；本次整理没有重新查询。

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
  BUNDLE-MANIFEST.json                   原始 Submgr 文件哈希
  COORDINATION.md / PROVENANCE.md        协调与来源
crash-validation-20261106/
  README.md                             三组件崩溃验证概览与证据边界
  cve-report-20261106.md                CVE 上报说明（7 项已验证漏洞）
  validation-report-20261106.md         隔离实测复现命令与原始输出
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

两份原始报告和 Submgr 原始材料包保持原内容；新的概览只提供导航和证据分层。
详见[路径迁移说明](REORGANIZATION.md)。`submgr/BUNDLE-MANIFEST.json` 仅覆盖原始
Submgr 包，不是整个仓库的总清单。

本仓库为私有工作记录。上传、整理或改名不等于向上游安全团队发送报告、
邀请维护者访问或获得漏洞确认。
