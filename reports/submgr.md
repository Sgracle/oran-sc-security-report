# [Security] Missing source-node ownership validation in RIC Subscription Delete Required handling

Repository: `o-ran-sc/ric-plt-submgr` / O-RAN SC RIC Platform Subscription Manager.

## Summary

The reviewed `handleE2TSubscriptionDeleteRequired()` implementation resolves each requested subscription using only its Instance ID in a registry shared across E2 Nodes. The handler receives the source E2 Node identity in `params.Meid`, and the selected subscription contains its owner in `subs.Meid`, but it does not verify that these identify the same node before scheduling deletion.

The downstream deletion path reconstructs the destination and subscription identifiers from the selected subscription. Consequently, a message identified as originating from E2 Node A can select a subscription owned by E2 Node B and cause the RIC to generate a deletion request addressed to B, followed by cleanup of B's local subscription state.

This is a source-level finding supported by an isolated test executing extracted, unchanged production function bodies with synthetic subscriptions and mocked external interfaces. A complete APER/SCTP/RMR end-to-end reproduction and deployed-image verification have **not** been completed.

## Reviewed source snapshots

| Branch snapshot | Commit | `container-tag.yaml` declaration | Evidence |
|---|---|---|---|
| h-release | `ad93392cb6c1180d9f4828baea4f5427c3163d4e` | 0.9.6 | Static source review |
| i-release | `956bb478233a4467393a4f3f67f4b4c04544a736` | 0.10.1 | Static review and isolated original-function tests |
| j-release / l-release | `ff822852aa2d4a81b5b6da18111f55a9b315603e` | 0.10.2 | Static source review; these branches share the same commit |
| m-release | `191c81a6c587cde692345b50da94e7cdae8c56d8` | 0.10.3 | Static source review |
| reviewed master | `98ebf666e797ee02a7b2946742ebcc4259252b4c` | 0.10.4 | Static source review |

The relevant handler, registry lookup, and five downstream deletion/sending functions have identical function text across these snapshots. These are reviewed branch heads and source tag declarations, not an exhaustive affected-version interval or verification of every published image. Please confirm introduction, supported affected versions, and any existing fix.

## Source evidence and call chain

1. [`control.go`, Required handler](https://github.com/o-ran-sc/ric-plt-submgr/blob/98ebf666e797ee02a7b2946742ebcc4259252b4c/pkg/control/control.go#L1686) decodes the list and calls `GetSubscriptionFirstMatch()` with only `subsTobeRemove.RequestId.InstanceId`. It checks an ongoing deletion count but does not check the source owner, complete incoming Request ID, or RAN Function ID against the selected subscription.
2. [`registry.go`, lookup](https://github.com/o-ran-sc/ric-plt-submgr/blob/98ebf666e797ee02a7b2946742ebcc4259252b4c/pkg/control/registry.go#L606) returns a subscription from the global `register` map using the provided Instance ID.
3. [`control.go`, `SendSubscriptionDeleteReq()`](https://github.com/o-ran-sc/ric-plt-submgr/blob/98ebf666e797ee02a7b2946742ebcc4259252b4c/pkg/control/control.go#L1572) builds a request from the selected object, sets the on-wire Requestor ID to `123`, takes the Function ID from `subs.SubReqMsg`, and sets the destination MEID from `subs.Meid`.
4. `SendSubscriptionDeleteReqToE2T()` checks the selected destination's connection and schedules `handleSubscriptionDelete()`. That check does not establish that the original sender owns the selected subscription.
5. [`control.go`, delete handling](https://github.com/o-ran-sc/ric-plt-submgr/blob/98ebf666e797ee02a7b2946742ebcc4259252b4c/pkg/control/control.go#L1218) and [`registry.go`, cleanup](https://github.com/o-ran-sc/ric-plt-submgr/blob/98ebf666e797ee02a7b2946742ebcc4259252b4c/pkg/control/registry.go#L514) can invalidate the subscription, remove its endpoints/routes/database record, and release its Instance ID.

The important transition is that the received identity remains A, while the selected object and generated output identify B. A globally unique allocation policy for Instance IDs does not itself authorize A to act on B's subscription.

## Expected behavior

A Delete Required message from A must be resolved only against subscriptions belonging to A. If the listed subscription is owned by B, the handler should not schedule B's deletion, generate an output to B, or modify B's subscription, route, database, or ID-allocation state. Valid messages concerning the sender's own subscriptions should continue to work.

## Isolated reproduction and observations

The local fixture creates two connected logical nodes and pre-populated subscription state:

| Field | Node A subscription | Node B subscription |
|---|---|---|
| Owner RanName | A | B |
| Instance ID | 101 | 102 |
| On-wire Requestor ID | 123 | 123 |
| Function ID | 2 | 2 |
| Endpoint / route / database record | Present | Present |

The stored xApp-facing Requestor ID is deliberately `900` to distinguish it from the current implementation's on-wire value `123`.

The test supplies source metadata `params.Meid.RanName = "A"` and a decoded-list equivalent identifying synthetic subscription `(Requestor=123, Instance=102, Function=2)` to the original Required handler. No SCTP socket, actual E2 Node, or real RMR/SDL/route service is involved.

The fixture automatically extracts the original handler, deletion functions, registry cleanup, subscription locking, and tracker/transaction implementations from the specified local revision. It replaces the package/import context and external types/interfaces. JSON substitutes for ASN.1, RMR outputs are captured in memory, and simulated success/failure/timeout events are delivered to transactions. Therefore this validates the internal control flow, not full native integration or protocol reachability.

Observed before the fix:

- An output addressed to **B** is generated with `(Requestor=123, Instance=102, Function=2)`.
- B/102 is invalidated and removed from the registry.
- The mock route and E2-subscription database entries for B/102 are deleted.
- Instance ID 102 is returned to the available-ID pool.
- A/101 remains unchanged.

After a local ownership/identity guard is added before scheduling deletion, the same cross-node input produces no deletion output or route/database cleanup calls, and B's original state is preserved. A-own and B-own deletion controls still pass.

The shared behavior suite changed from 9/20 passing before the fix to 20/20 afterward; nine additional matcher cases passed in the extracted test package, for 29/29 after the fix. A 50-run randomized-order race test passed for the covered paths. These are **not** results of the repository's complete native test suite.

The complete native `go test ./...` could not compile because the isolated environment lacked cached Go dependencies (`xapp-frame`, `sdlgo`, `nodeb-rnib`, and others), with dependency downloads disabled. Full native regression and end-to-end protocol tests remain outstanding. No tests were sent to production or third-party networks.

## Security impact and preconditions

A malicious or compromised, connected E2 Node that can deliver such a message and reference another live subscription in the same Submgr registry may cause unauthorized changes to that subscription's availability and local management state. The actual effect depends on target connection state, subscription validity, policy flags, endpoint/transaction state, and deployment details.

The handler is not specific to KPM. Other subscriptions using this lifecycle should also be reviewed. This report does not claim user-plane disruption, arbitrary RIC code execution, cross-RIC impact, or successful exploitation of deployed images.

## Protocol context

[ETSI TS 104 039 V4.0.0 / O-RAN E2AP v04.00](https://www.etsi.org/deliver/etsi_ts/104000_104099/104039/04.00.00_60/ts_104039v040000p.pdf), sections 8.2.2A.1-3 and 9.2.7-8, describes deletion of subscriptions in the initiating E2 Node and node-scoped identifiers. Earlier v02.03 text also limits the procedure to that node's existing subscriptions. This supports keeping the source node, target subscription and deletion destination associated.

The generated code in the audited SC baseline cites v02.00. The exact v02.00 prose was not obtained, so this is not a claim of an exhaustive clause-by-clause conformance assessment for that version. The concrete finding is an implementation-level ownership binding failure, which can be fixed without a wire-format change.

## Suggested remediation and regression coverage

Before adding a matched subscription to the deletion list or changing any state:

- Bind the trusted source E2 Node identity to the stored subscription owner, using the project's canonical node identity.
- Verify the actual on-wire complete Request ID and the associated RAN Function ID.
- Reject missing or mismatched identity without scheduling deletion or cleanup.
- Preserve handling of legitimate entries and add cross-node, mixed-batch, missing-identity and concurrent-different-subscription tests.

A node-scoped lookup, composite key, or owner check after global lookup can enforce this. Do not blindly compare the incoming Requestor ID to the stored xApp-facing ID: the reviewed sending code replaces it with `123` on E2. That value is implementation-specific, not an E2AP standard constant.

A local minimal guard and isolated regression harness are available for coordinated maintainer review. No claim is made that the full native build, merged multi-endpoint deletion, same-subscription races, reconnection, ID reuse or other message handlers have been verified.

## Maintainer coordination

This private working copy was packaged on 2026-11-03. Existing Jira: [RIC-1079](https://lf-o-ran-sc.atlassian.net/browse/RIC-1079). The reporter states there has been no response; Jira was not re-read during this packaging task.


Please confirm the supported security-reporting channel, affected/fixed versions and preferred regression-test integration. If this report should be handled privately, please provide or use the project's security workflow for further evidence and coordination.
