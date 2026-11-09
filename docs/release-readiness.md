# Release readiness

This record separates tested source, installable modules, hosted-provider verification and publication. Passing one gate does not imply the others passed. Earlier candidate-archive checks rewrote module requirements and therefore did not prove that the checked-in module graph could install publicly.

| Gate | Status and evidence |
| --- | --- |
| Consumer API | Message, Sender, construction, transport ownership and minimum Go 1.24 preserved. No public API or persisted-format migration introduced by these reviews. |
| Delivery evidence | WhatsApp encoded-body bounds, incident.io request-key correlation, Vonage multi-segment acceptance and ClickSend rejection classifications have direct regressions. Missing or partial evidence stops fallback. |
| Module requirements | The four runtime adapters, four example modules and integration module now pin an actual public revision instead of the nonexistent v0.0.0. Relative development replacements remain intact. |
| Public consumer | The old eafcce8 revision failed real public resolution for all four optional modules. The new Go consumer verifier must pass on the pushed repair commit before this gate is closed. |
| Source validation | Exact unit coverage, all-module race/vet, native server tests, deterministic generation and vulnerability reachability checks are required. Measured reports are linked below. |
| Hosted providers | Live hosted-provider tests and paired Signal verification remain pending, as recorded per driver. Native protocol fixtures are not live delivery evidence. |
| Publication | No release version is selected or tagged by this review. Publish and verify the root, then pin and independently tag the four optional modules, then verify every actual version publicly. |

The second SMS pass found that Vonage returns one result for each generated SMS segment. Requiring exactly one result incorrectly rejected successful multipart sends. The driver now verifies the declared count, every recipient and distinct accepted message IDs. Mixed accepted/rejected segments remain unknown to avoid replaying accepted parts. ClickSend now preserves documented throttling and credential errors; missing, unfamiliar or internal-error evidence cannot trigger fallback.

The packaging pass reproduced a real external-consumer failure: every optional module required an unpublished v0.0.0 root version. Retargeting temporary archives hid that fault. The repaired requirements use v0.0.0-20261109213702-eafcce898c23, independently resolved through the public proxy for each applicable module. `make verify-consumer REVISION=<pushed-commit-or-tag>` now checks actual published source with a fresh cache, normal checksum verification, no workspace, no replacements and no local overlays. CI runs the same check on pushes.

See [unit evidence](unit-results.json), [native integration evidence](integration-results.json), [driver verification scope](drivers.md) and [publication sequence](development.md#independent-release). The root module remains standard-library-only. No hosted-delivery or public-release claim follows from passing local tests.
