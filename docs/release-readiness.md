# Release readiness

This record separates tested source, installable modules, hosted-provider verification and publication. Passing one gate does not imply the others passed. Earlier candidate-archive checks rewrote module requirements and therefore did not prove that the checked-in module graph could install publicly.

| Gate | Status and evidence |
| --- | --- |
| Consumer API | Message, Sender, construction, transport ownership and minimum Go 1.24 preserved. No public API or persisted-format migration introduced by these reviews. |
| Delivery evidence | WhatsApp encoded-body bounds, incident.io request-key correlation, Vonage multi-segment acceptance and ClickSend rejection classifications have direct regressions. Missing or partial evidence stops fallback. |
| Module requirements | The four runtime adapters, four example modules and integration module now require v0.1.0. Relative development replacements remain intact. |
| Public consumer | Published v0.1.0 passed all six fresh public consumers on Go 1.24.13 and Go 1.27.3: all 51 drivers together, the root alone, and each optional module with its declared dependencies. Behavior checks pass with normal checksums, no workspace, no replacements and no source overlays. @latest selects v0.1.0 for all five modules. |
| Source validation | Passed: 388 tests, exact 2146/2146 library statement coverage, all-module race/vet, pinned real ntfy/Gotify/Synapse tests, deterministic generation and no reachable vulnerabilities. Measured reports are linked below. |
| Hosted providers | Live hosted-provider tests and paired Signal verification remain pending, as recorded per driver. Native protocol fixtures are not live delivery evidence. |
| Publication | Complete: root v0.1.0 and all four driver module tags are published and independently checksum-verified at the commits below. The [GitHub release](https://github.com/goforj/notification/releases/tag/v0.1.0) is public, is marked latest, and targets the reviewed root tag. |

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

The initial SNS lookup hit a cached GitHub DNS failure in the public proxy and checksum service. The release stayed in draft until that response expired and normal checksum verification succeeded. All five archives now resolve at v0.1.0, match the reviewed commits, include LICENSE, and carry verified module and go.mod checksums. All four optional modules declare root v0.1.0. No checksum, requirement or tag was bypassed or rewritten to conceal the external failure.

The release was published on 2026-11-10 after both toolchains passed all six public consumers. Each consumer executes Sender, invalid-destination validation and HTTP 408/500 uncertainty checks. README installation commands use @latest, which was separately verified to select v0.1.0 for every runtime module.

### Public module checksums

| Module | Version | Module checksum | go.mod checksum |
| --- | --- | --- | --- |
| github.com/goforj/notification | v0.1.0 | h1:AkeInbltueQSCIm+4M77TgPNfa9EobUiBJDjVGU4ixI= | h1:pWKJZS6hFR8ZWqBKN+6pnKBI7Jy5Sckz5xjGNj0opJ8= |
| github.com/goforj/notification/driver/mail | v0.1.0 | h1:kOBPBQjZoDxw8NxJ1eiOjy8OHCgvgHPIXKA9MeBNHqI= | h1:cu0vO+ENgBkAk5rBmqu/OkabwEMbOttc9eZ+b39saAQ= |
| github.com/goforj/notification/driver/apns | v0.1.0 | h1:Prx9LovCpFIk/KGWiBaN3UlGkv1pw2XZy+mhcXuk1Ko= | h1:3rB0w7Jsg1Vl8DrJmMer7oIwaIccoTMAcM4ZUTfhYz4= |
| github.com/goforj/notification/driver/sns | v0.1.0 | h1:tQZ8VDVCnJ1HTHQhVo/j81ME8LfnAE0wNFnkYRtU3ns= | h1:ujDq+BnhE9CSDGoEeZcLTPro8HMTA/4W7Q8+VAk4Y1g= |
| github.com/goforj/notification/driver/webpush | v0.1.0 | h1:MhU0QxOCNEBydRttOzeSUmuOsE8DAxhtxNxYHI7hcuo= | h1:d3ZV4L4GF0D9YDiFNhvkqGO1QH0mwpZ7WcT8STwuDhc= |
