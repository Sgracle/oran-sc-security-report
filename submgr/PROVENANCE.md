# Provenance and limitations

Baseline: O-RAN SC Submgr i-release commit
`956bb478233a4467393a4f3f67f4b4c04544a736`.
Fixed package: the local guard in `patches/ownership-fix.patch` applied to that
baseline, including the matcher unit test. Original source hashes and extracted
declaration lists are recorded in each package's `source-manifest.json`.

Original evidence in `evidence/` was collected in October 2026, then copied
without rerunning or rewriting those tests. A fresh packaging verification, if
performed, is recorded only in `verification/`. Historical review manifests
list additional original-workspace materials not distributed here; they are
provenance records, not a list of files promised by this repository.

Extracted production code is from https://github.com/o-ran-sc/ric-plt-submgr .
The original files carry Copyright (c) 2019 AT&T Intellectual Property and
Copyright (c) 2019 Nokia and Apache License 2.0 notices, retained in the extracted
files. Packaging adds that comment only; it does not change production function
bodies. See `LICENSES/O-RAN-SC-LICENSES.txt` and
https://www.apache.org/licenses/LICENSE-2.0 .

The mock harness and proposed patch have not been accepted upstream. This
private working record does not apply a new blanket license to the research
notes or third-party material. Full standards PDFs, operational logs, credentials,
emails, unrelated appmgr findings and full upstream repositories are omitted.
