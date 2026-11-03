#!/usr/bin/env bash
set -eu
if [ "${BUNDLE_NETNS:-}" != 1 ]; then
  exec unshare -n -- env BUNDLE_NETNS=1 bash "$0" "$@"
fi
cd "$(dirname "$0")/.."
ROOT_DIR=$PWD
mkdir -p verification
cat /proc/net/dev > verification/network-devices.txt
export GOPROXY=off GOSUMDB=off GOTOOLCHAIN=local
for variant in baseline fixed; do
  set +e
  (cd "tests/$variant" && go test -json -count=1 -timeout=30s ./...) > "verification/$variant-tests.jsonl" 2>&1
  result=$?
  set -e
  printf '%s\n' "$result" > "verification/$variant-exit-code.txt"
  printf '%s exit=%s\n' "$variant" "$result"
done
python3 tools/check-results.py
