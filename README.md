# O-RAN SC Submgr: private security review

Private working record for the cross-E2-node ownership finding in
`RIC Subscription Delete Required` handling. Prepared on 2026-11-03.

本仓库汇总本次 Submgr 跨节点订阅归属问题，供私下复核和跟踪。
已确认审查快照中的源码缺陷，并有隔离的原函数 mock 测试证据；
没有完成真实 APER/SCTP/RMR 链路、发布镜像和完整原生回归验证。

## Start here

- [English vulnerability report](reports/submgr.md): call chain, pinned commits,
  preconditions, observations, remediation and limitations.
- [Protocol and implementation review](reports/standards-and-scope.md): separates
  protocol requirements, implementation evidence and static-analysis candidates.
- [Before/after results](evidence/SUMMARY.md) and [machine-readable summary](evidence/summary.json).
- [Minimal proposed patch](patches/ownership-fix.patch): local proposal against
  i-release `956bb478233a4467393a4f3f67f4b4c04544a736`; not an upstream accepted fix.
- [Standalone synthetic tests](tests/README.md): baseline and fixed extracted packages.
- [Evidence and provenance](PROVENANCE.md).
- [Coordination status](COORDINATION.md).

## Evidence boundary

The original shared suite passed 9/20 cases before the guard and 20/20 afterward.
Nine extra identity matcher cases make the fixed total 29/29. These results use
extracted production functions, replacement DTOs, a JSON codec and mocked
RMR/routes/database/remote events. `RemoteActive` is synthetic state.
Failure/timeout cleanup is characterization, not a separately confirmed vulnerability.

No CVE, CVSS score, maintainer confirmation or deployed-image exploit is claimed.
Source snapshots and original test records date from October 2026; packaging this
repository does not establish the status of current upstream code.

Creating this private repository does not notify O-RAN SC maintainers or give them
access. An owner-authorized invitation or separate private security report is needed
for external coordination. Keep this repository private during review.
