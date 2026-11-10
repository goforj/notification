# Release readiness

This record separates tested source, installable modules, hosted-provider verification and publication. Passing one gate does not imply the others passed. Earlier candidate-archive checks rewrote module requirements and therefore did not prove that the checked-in module graph could install publicly.

| Gate | Status and evidence |
| --- | --- |
| Consumer API | Message, Sender, construction, transport ownership and minimum Go 1.24 preserved. No public API or persisted-format migration introduced by these reviews. |
| Delivery evidence | WhatsApp encoded-body bounds, incident.io request-key correlation, Vonage multi-segment acceptance and ClickSend rejection classifications have direct regressions. Missing or partial evidence stops fallback. |
| Module requirements | The four runtime adapters, four example modules and integration module now require v0.1.0. Relative development replacements remain intact. |
| Public consumer | The final candidate passed all six independent consumers in CI on Go 1.24. Published v0.1.0 verification is pending a cached SNS lookup failure in the public proxy and checksum service. Root, Mail, APNs and Web Push archives resolve with checksums; no checksum bypass is used. |
| Source validation | Passed: 388 tests, exact 2146/2146 library statement coverage, all-module race/vet, pinned real ntfy/Gotify/Synapse tests, deterministic generation and no reachable vulnerabilities. Measured reports are linked below. |
| Hosted providers | Live hosted-provider tests and paired Signal verification remain pending, as recorded per driver. Native protocol fixtures are not live delivery evidence. |
| Publication | Root v0.1.0 and all four driver module tags are published at the reviewed commits below. The coordinated GitHub release remains a draft until published SNS checksum verification succeeds. |

The second SMS pass found that Vonage returns one result for each generated SMS segment. Requiring exactly one result incorrectly rejected successful multipart sends. The driver now verifies the declared count, every recipient and distinct accepted message IDs. Mixed accepted/rejected segments remain unknown to avoid replaying accepted parts. ClickSend now preserves documented throttling and credential errors; missing, unfamiliar or internal-error evidence cannot trigger fallback.

The packaging pass reproduced a real external-consumer failure: every optional module required an unpublished v0.0.0 root version. Retargeting temporary archives hid that fault. The initial repair used v0.0.0-20261109213702-eafcce898c23, independently resolved through the public proxy for each applicable module. Later behavioral minimums are recorded below. `make verify-consumer REVISION=<pushed-commit-or-tag>` now checks actual published source with a fresh cache, normal checksum verification, no workspace, no replacements and no local overlays. CI runs the same check on pushes.

See [unit evidence](unit-results.json), [native integration evidence](integration-results.json), [driver verification scope](drivers.md) and [publication sequence](development.md#independent-release). The root module remains standard-library-only. No hosted-delivery or public-release claim follows from passing local tests.

The README installation commands and version badge are generated from docs/release.json. Local replacements remain for repository development. Live hosted-provider verification remains pending separately from versioned publication.

## Further reliability and protocol passes

The next reliability pass reproduced a receiver committing work and then returning HTTP 500. A repeated journaled Send previously delivered it twice. HTTP 408 and server failures now retain transient advice while preserving unknown acceptance, including APNs through the shared transport, Web Push and SNS server faults. The journal retains uncertainty and fallback stops. Direct regressions cover the replay counter, transient status evidence and bounded advice. See [the runtime compatibility note](contracts.md#transient-errors-and-uncertain-acceptance).

The native response pass found that Better Stack action responses accepted an unrelated resource type when its ID matched. Acknowledge and Resolve now require the incident type as well as the requested ID. The input-boundary pass also rejects HTTP authorities with an empty hostname before any transport or resolver call.

A suspected Zulip form-encoding issue was ruled out against the actual [server parser](https://github.com/zulip/zulip/blob/main/zerver/actions/message_send.py): it explicitly supports raw channel names and JSON-encoded names or IDs. Existing configuration remains intact.

The stronger consumer check passed the combined and root-only cases at 881453f, then failed the isolated Mail case because its old transitive root accepted an invalid HTTP authority. Compiling the APIs had hidden that behavioral minimum mismatch. That review pinned runtime adapter, example and integration requirements to the public root correction v0.0.0-20261109224454-a1a26f8d3182. Release requirements subsequently advanced to v0.1.0. The public consumers additionally check HTTP 408/500 uncertainty, retry advice and invalid-destination rejection, using a local transport with no provider calls.

The cancellation pass reproduced a native acknowledgement interrupted after HTTP 200 headers: the outcome remained unknown but the caller's cancellation was lost. The provider boundary now preserves context cancellation without treating incomplete native evidence as acceptance. Deterministic response-reader regressions cover native and Slack acknowledgements and transient webhook/Web Push failures without network timing. Header-only generic acceptance remains successful.

## v0.1.0 publication

| Module tag | Reviewed commit |
| --- | --- |
| v0.1.0 | 130ded513a5fe673a35dbca6a806e90a6d2d5aac |
| driver/mail/v0.1.0 | 0cceb76a42102452e0441c702a1a3a95919775f9 |
| driver/apns/v0.1.0 | 0cceb76a42102452e0441c702a1a3a95919775f9 |
| driver/sns/v0.1.0 | 0cceb76a42102452e0441c702a1a3a95919775f9 |
| driver/webpush/v0.1.0 | 0cceb76a42102452e0441c702a1a3a95919775f9 |

CI and Security passed for both reviewed commits. The root was published and checksum-verified before the optional module requirements advanced to v0.1.0. All four optional tags were then pushed atomically and their remote targets verified, including annotated-tag resolution.

At publication, the public proxy and checksum service returned a cached GitHub DNS failure for SNS v0.1.0. The proxy response expires at 2026-11-10 18:22:24 UTC. Root, Mail, APNs and Web Push downloads already returned public checksums. The GitHub release is held as a draft while the normal public consumer check awaits SNS propagation. Checksums, module requirements and tags are not bypassed or rewritten to conceal that external failure.
