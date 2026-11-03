# Appmgr / xApp image authenticity review (RIC-1078)

本目录归档此前讨论的“下载镜像/镜像来源校验”问题，实际审查对象为
O-RAN SC `ric-plt-appmgr` 的 `xapp_onboarder`。归档日期：2026-11-03。

## Reports

- [English preliminary report](RIC-1078-appmgr-preliminary-report.md), prepared 2026-10-28: pinned source evidence, trust-boundary prerequisites, proposed validation and remediation.
- [中文 CNVD 报送草稿](CNVD-draft.zh-CN.md), prepared 2026-10-15: product details, source references, proposed reproduction and submission guidance. This is an archived draft; placeholders and uncompleted validation steps remain.

## Scope and evidence

The reviewed component validates descriptor structure and generates/distributes Helm charts.
The recorded concern is the absence of component-level trusted-publisher signature verification
and immutable image-content binding. Actual exploitability depends on who can supply or modify
onboarding input and whether mandatory verification exists elsewhere in the supported deployment.

These documents record preliminary source observations. They do not establish a completed
end-to-end exploit, a privilege-boundary violation, maintainer acceptance, a formal severity,
or an assigned CVE. Proposed reproduction steps are not executed results.

This Appmgr finding is separate from the Submgr cross-E2-node ownership finding elsewhere in
this repository. The repository's Submgr tests, patch, pass counts and evidence logs do not
validate the Appmgr hypothesis. The existing root `BUNDLE-MANIFEST.json` describes the original
Submgr package and does not inventory this separately added documentation directory.

## Coordination references

- Project contact ticket: [RIC-1078](https://lf-o-ran-sc.atlassian.net/browse/RIC-1078).
- MITRE request tracking reference: `CAN-2026-2039411`, not a CVE identifier.
- The earlier portal check on 2026-10-20 showed the request was still under review, with no
  assigned CVE ID. Its status was not checked again while archiving these documents on
  2026-11-03. Coordination statements in each archived document refer to its preparation date.

Archiving here does not submit the CNVD draft, update the MITRE request, or send a report or
access invitation to O-RAN SC maintainers.

## Source provenance

The two reports were copied from the reporter's existing local documents, preserving their
dated statements and evidence limitations. No new experiment or current-upstream audit was
performed for this upload.
