# Private report: potential xapp_onboarder artifact authenticity issue

Prepared: 2026-10-28. Existing Jira reference: [RIC-1078](https://lf-o-ran-sc.atlassian.net/browse/RIC-1078).

This document consolidates prior local review notes. It is a preliminary security assessment, not a newly executed end-to-end reproduction or a maintainer-confirmed vulnerability. No formal severity, CVSS score or CVE assignment is asserted.

## Source baseline

- Project: O-RAN-SC RIC Platform Application Manager, repository `o-ran-sc/ric-plt-appmgr`.
- Component/package: `xapp_onboarder 1.0.0`.
- Previously audited tag: `0.5.10`.
- Pinned commit: `bdc6f07284d33aaad6a04e30fd5821776d7a8e18`.
- Prior notes also reviewed commit `a6fd863532bdee735c88904511dd72abf3050d8c`; it is not presented as today's latest master.
- Introduction, supported affected range and fixed version: unconfirmed.

## Source observations

The reviewed onboarding path accepts a descriptor, performs JSON Schema validation, generates a Helm chart, and invokes chart packaging/distribution. The descriptor controls the image registry, name and mutable tag, and may supply the container command and arguments. The reviewed component path does not itself verify a trusted publisher signature or bind the descriptor to an immutable image digest.

Pinned source references:

1. Input validation and onboarding flow:
   https://github.com/o-ran-sc/ric-plt-appmgr/blob/bdc6f07284d33aaad6a04e30fd5821776d7a8e18/xapp_orchestrater/dev/xapp_onboarder/xapp_onboarder/api/onboard.py
2. Descriptor schema:
   https://github.com/o-ran-sc/ric-plt-appmgr/blob/bdc6f07284d33aaad6a04e30fd5821776d7a8e18/xapp_orchestrater/dev/xapp_onboarder/xapp_onboarder/helm_controller/xapp_schema.py
3. Deployment image and command rendering:
   https://github.com/o-ran-sc/ric-plt-appmgr/blob/bdc6f07284d33aaad6a04e30fd5821776d7a8e18/xapp_orchestrater/dev/xapp_onboarder/xapp_onboarder/resources/xapp-std/templates/deployment.yaml
4. Chart linting, packaging and distribution:
   https://github.com/o-ran-sc/ric-plt-appmgr/blob/bdc6f07284d33aaad6a04e30fd5821776d7a8e18/xapp_orchestrater/dev/xapp_onboarder/xapp_onboarder/helm_controller/xApp_builder.py
5. Official API test and fixtures:
   https://github.com/o-ran-sc/ric-plt-appmgr/blob/bdc6f07284d33aaad6a04e30fd5821776d7a8e18/xapp_orchestrater/dev/xapp_onboarder/tests/test_api.py
   https://github.com/o-ran-sc/ric-plt-appmgr/blob/bdc6f07284d33aaad6a04e30fd5821776d7a8e18/xapp_orchestrater/dev/xapp_onboarder/tests/constants.py

The test code expects a successful response for its unsigned fixture. Inspection of that test is evidence of the test's expected behavior, not a claim that this report newly ran the service or demonstrated execution in a supported deployment.

## Security hypothesis and necessary preconditions

If a less-trusted actor can supply or modify onboarding input, or alter content behind a mutable image tag, and no mandatory downstream check establishes publisher and content identity, an untrusted artifact may enter a deployable chart. Actual execution additionally requires operator or automation deployment. Effects are limited by the xApp's granted identity, RBAC, credentials, service permissions, network reach, admission controls and actual image-pull behavior.

The key unresolved question is whether the supported architecture creates a trust boundary here. If onboarding is restricted to trusted administrators who already have equivalent deployment authority, or a mandatory external verification mechanism prevents untrusted artifacts from being deployed, the component-level absence of a signature check may be an architectural choice or hardening opportunity rather than a standalone security vulnerability.

No unauthorized Internet access, automatic cluster-administrator privilege, host compromise or real-network disruption has been established. Severity remains unassigned pending those checks.

## Proposed isolated validation — not claimed as completed

1. Record the supported onboarding architecture, API identity and permissions, admission policy, trusted image sources and any external signature checks.
2. In an isolated environment, use a benign local image and a schema-valid synthetic descriptor. Preserve raw requests, responses, service logs and generated chart contents.
3. Compare unsigned/modified input with a structurally invalid negative control to distinguish schema validation from authenticity checks.
4. Separately test chart generation and actual deployment. If testing a mutable tag, record both image digests, pull policy, caching and the container's actual image ID.
5. Establish whether the test actor lacks equivalent direct deployment authority. Record effective external rejection controls rather than treating them as a bypass.

A complete service/deployment reproduction and demonstrated privilege-boundary violation are not established in the attached evidence.

## Remediation discussion

Where the supported trust model requires trusted publishers, verify an authenticated manifest binding xApp identity/version, descriptor/schema hashes and each immutable image digest. Enforce verification at a mandatory point across onboarding and deployment; if delegated externally, document and test that the gate cannot be bypassed. Digest pinning establishes content identity; a trusted signature establishes publisher identity. They serve different purposes.

## Coordination and disclosure status

The current Jira description was checked on 2026-10-28 using the reporter's authenticated session. It contains a high-level summary and private-contact request, without detailed reproduction steps. This observation does not establish that the finding has never been disclosed elsewhere.

Earlier local notes list MITRE request tracking reference `CAN-2026-2039411`. The original receipt and current request status were not independently reverified during this email-preparation task. This is not presented as an assigned CVE.

Please confirm the supported trust model, affected versions, whether the project considers this a vulnerability, the responsible maintainer and preferred confidential evidence channel. The prior local documents and proposed reproduction outline remain available for private review.
