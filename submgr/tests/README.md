# Isolated original-function tests

`baseline/` and `fixed/` are standalone Go modules with no third-party modules.
They contain selected production declarations plus local DTOs and mocks.
For attribution, the original SC copyright/license comment was added above
`extracted.go`; function bodies were not changed during packaging.

Run from a local Linux environment with Go and permission to create a network
namespace (for example an isolated WSL test environment):

```sh
unshare -n -- bash tools/verify-bundle.sh
```

The script refuses to run outside the new network namespace. It disables Go
dependency downloads. Baseline exit 1 and fixed exit 0 are expected. It saves
fresh logs in `verification/` without overwriting the October evidence.

Manual execution within a separately isolated environment:

```sh
cd tests/baseline
GOPROXY=off GOSUMDB=off GOTOOLCHAIN=local go test -count=1 -timeout=30s ./...
# expected failure: security invariants are not satisfied
cd ../fixed
GOPROXY=off GOSUMDB=off GOTOOLCHAIN=local go test -count=1 -timeout=30s ./...
# expected success
```

`tools/extract/main.go` documents the AST-based extraction from the reviewed
local SC source. Re-extraction requires that source checkout. These standalone
modules do not require it. `native_ownership_test.go` is the original matcher
test with imports replaced for this harness; it is not a full native build.

The historical `go vet` unreachable-code finding and full native build's
missing offline dependencies are recorded separately. No E2 nodes or services
are contacted by these tests.
