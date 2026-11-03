# Repository layout change — 2026-11-03

This repository contains two separate O-RAN SC research records. Its repository name is now
`oran-sc-security-report`; the previous name was `oran-sc-submgr-security-report`.

## Path mapping

| Previous path | Current path |
|---|---|
| `reports/appmgr/*` | `appmgr/*` |
| Original Submgr `README.md` | `submgr/README.md` |
| Original root `COORDINATION.md`, `PROVENANCE.md`, `BUNDLE-MANIFEST.json` | Same filenames under `submgr/` |
| Original `reports/submgr.md`, `reports/standards-and-scope.md` | Same paths under `submgr/` |
| Original `evidence/`, `tests/`, `patches/`, `tools/`, `verification/`, `LICENSES/` | Same directories under `submgr/` |

The root README is a new index. The Appmgr README is an updated navigation/status page.
`submgr/OVERVIEW.zh-CN.md` is a new Chinese overview. The original root
`.gitattributes` and `.gitignore` are retained and also included in the relocated original
Submgr package.

## Evidence preservation

All 58 files of the original Submgr bundle are moved as existing Git blobs with unchanged
contents and modes. The original manifest remains unchanged and its relative paths are resolved
from `submgr/`. The two dated Appmgr reports are also preserved as existing blobs.
The old Appmgr navigation page is superseded by the new overview and remains in Git history.

The Submgr manifest is not a repository-wide inventory. Newly written navigation files and the
Appmgr reports are outside its original scope. Test results have not been re-executed or
redated as part of this layout change.

Pre-reorganization revision: `0491836ea9338e72ed9de50192beec506b833782`.
Historical paths can be inspected at that revision. Current links should use the new locations.
No Git history is rewritten.

## Reading and running

Start at [the root index](README.md), then use [Appmgr](appmgr/README.md) or
[Submgr](submgr/OVERVIEW.zh-CN.md). For Submgr tools, first enter `submgr/`;
the scripts, modules and relative evidence paths remain grouped within that package.
