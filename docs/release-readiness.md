# Release readiness

This record separates tested source, installable modules, hosted-provider verification and publication. Passing one gate does not imply the others passed. Earlier candidate-archive checks rewrote module requirements and therefore did not prove that the checked-in module graph could install publicly.

| Gate | Status and evidence |
| --- | --- |
| Consumer API | Message, Sender, construction, transport ownership and minimum Go 1.24 preserved. No public API or persisted-format migration introduced by these reviews. |
| Delivery evidence | WhatsApp encoded-body bounds, incident.io request-key correlation, Vonage multi-segment acceptance and ClickSend rejection classifications have direct regressions. Missing or partial evidence stops fallback. |
| Module requirements | The four runtime adapters, four example modules and integration module now pin an actual public revision instead of the nonexistent v0.0.0. Relative development replacements remain intact. |
| Public consumer | Passed at 6d077e3cda12 on Go 1.24: all five modules resolved through the public proxy and checksum database, all 51 drivers compiled together, and each of the five modules also compiled independently with its declared dependencies. Sender, invalid-destination validation and HTTP 408/500 uncertainty checks executed in all six consumers without replacements. The old eafcce8 revision failed this same check. |
| Source validation | Passed: 388 tests, exact 2146/2146 library statement coverage, all-module race/vet, pinned real ntfy/Gotify/Synapse tests, deterministic generation and no reachable vulnerabilities. Measured reports are linked below. |
| Hosted providers | Live hosted-provider tests and paired Signal verification remain pending, as recorded per driver. Native protocol fixtures are not live delivery evidence. |
| Publication | Version v0.1.0 is selected. Publication completes after the root and four optional module tags are independently verified through the public proxy and checksum database. |

The second SMS pass found that Vonage returns one result for each generated SMS segment. Requiring exactly one result incorrectly rejected successful multipart sends. The driver now verifies the declared count, every recipient and distinct accepted message IDs. Mixed accepted/rejected segments remain unknown to avoid replaying accepted parts. ClickSend now preserves documented throttling and credential errors; missing, unfamiliar or internal-error evidence cannot trigger fallback.

The packaging pass reproduced a real external-consumer failure: every optional module required an unpublished v0.0.0 root version. Retargeting temporary archives hid that fault. The initial repair used v0.0.0-20261109213702-eafcce898c23, independently resolved through the public proxy for each applicable module. Later behavioral minimums are recorded below. `make verify-consumer REVISION=<pushed-commit-or-tag>` now checks actual published source with a fresh cache, normal checksum verification, no workspace, no replacements and no local overlays. CI runs the same check on pushes.

See [unit evidence](unit-results.json), [native integration evidence](integration-results.json), [driver verification scope](drivers.md) and [publication sequence](development.md#independent-release). The root module remains standard-library-only. No hosted-delivery or public-release claim follows from passing local tests.

The README installation commands and version badge are generated from docs/release.json. Local replacements remain for repository development. Live hosted-provider verification remains pending separately from versioned publication.

## Further reliability and protocol passes

The next reliability pass reproduced a receiver committing work and then returning HTTP 500. A repeated journaled Send previously delivered it twice. HTTP 408 and server failures now retain transient advice while preserving unknown acceptance, including APNs through the shared transport, Web Push and SNS server faults. The journal retains uncertainty and fallback stops. Direct regressions cover the replay counter, transient status evidence and bounded advice. See [the runtime compatibility note](contracts.md#transient-errors-and-uncertain-acceptance).

The native response pass found that Better Stack action responses accepted an unrelated resource type when its ID matched. Acknowledge and Resolve now require the incident type as well as the requested ID. The input-boundary pass also rejects HTTP authorities with an empty hostname before any transport or resolver call.

A suspected Zulip form-encoding issue was ruled out against the actual [server parser](https://github.com/zulip/zulip/blob/main/zerver/actions/message_send.py): it explicitly supports raw channel names and JSON-encoded names or IDs. Existing configuration remains intact.

The stronger consumer check passed the combined and root-only cases at 881453f, then failed the isolated Mail case because its old transitive root accepted an invalid HTTP authority. Compiling the APIs had hidden that behavioral minimum mismatch. Runtime adapter, example and integration requirements now pin the public root correction v0.0.0-20261109224454-a1a26f8d3182. The public consumers additionally check HTTP 408/500 uncertainty, retry advice and invalid-destination rejection, using a local transport with no provider calls.

The cancellation pass reproduced a native acknowledgement interrupted after HTTP 200 headers: the outcome remained unknown but the caller's cancellation was lost. The provider boundary now preserves context cancellation without treating incomplete native evidence as acceptance. Deterministic response-reader regressions cover native and Slack acknowledgements and transient webhook/Web Push failures without network timing. Header-only generic acceptance remains successful.
