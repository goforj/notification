# Notification API review

Current status is recorded in [release readiness](release-readiness.md). Earlier recommendations and counts below are historical checkpoints, not the current release verdict.

Reviewed 2026-11-09 against baseline commit 023f2c0. This review covers the common sender API, sender composition, typed extension points, and standard Go interoperability. The earlier [provider review](provider-review.md) covers provider extraction and ecosystem research.

## Pass 1: Common application usage

Keep `Send(context.Context, Message) error` and the two-field Message. Applications can inject a single-method Sender, configure destinations once, and send content without planners, route revisions, journals, or generics. Returning an error supports ordinary Go control flow. A nil error means handling, which includes local capture; it cannot promise recipient delivery.

Keep `New(driver)` as a validation and safe-error boundary for custom implementations. Driver lifecycle stays with constructing code, and senders remain immutable once configured. Required dependencies should fail visibly when miswired. Nil guards, mutable service registries, implicit retries, and convenience overloads would complicate these contracts.

Decision: retain the public method sets and constructor signatures. Do not add title/body positional overloads, a universal provider-options map, or optional-message builders. Typed driver configuration already keeps content independent of delivery settings.

## Pass 2: Composition and failure semantics

Fanout and Fallback already return Sender. The composition example now uses them directly instead of wrapping each result in New. Single configured senders, application functions, compositions, and fakes can therefore satisfy the same consumer interface.

Retain joined errors in target order and the per-call concurrency limit. Fallback advances only on proved rejection. Retrying or advancing after uncertain acceptance can duplicate a message, so error normalization must preserve every explicit safe classification.

Three reproductions exposed incorrect behavior:

1. A custom error implementing Is for both permanent rejection and unknown acceptance retained only the first matching sentinel. Fallback could then invoke a backup.
2. A classified outer error could hide a wrapped cause identifying unknown acceptance or cancellation.
3. Truncating a group after 64 children could silently discard an unknown outcome in a later child and permit fallback.

Fixes retain all safe custom classifications and explicit wrapped evidence. A classified wrapper may still explain an opaque private cause without inventing uncertainty. Exhausted traversal budgets preserve an uncertainty marker. Provider prose remains absent from returned error chains. Regression tests exercise the public facade and actual fallback behavior, including cyclic and oversized errors.

Runtime behavior changes only where evidence was previously lost: errors.Is exposes additional safe matches, and fallback stops when unknown or throttled evidence is present or hidden by traversal bounds. No caller migration or public API change is required.

## Pass 3: Extension points and progressive complexity

SenderFunc is sufficient for adapting existing services and adding metrics, tracing, or admission control. The [decorator program](../examples/decorator/main.go) demonstrates a concurrency-safe counter, context/error propagation, and validation before the custom handler. It is executed by the shared example runner and projected into the README.

Retain IncidentSender as an optional capability. Trigger, acknowledge, and resolve require stable correlation and cannot be represented reliably by a title and body. They should not become requirements on chat, SMS, email, or capture senders.

Retain the typed Dispatcher API for scoped recipients, semantic content, application policy, and explicit outcomes. Do not automatically adapt an arbitrary Sender into an accepted typed transport: a nil error may mean local capture. Message and typed payloads serve different existing contracts, so merging their types or moving dispatcher APIs would introduce compatibility costs without simplifying ordinary sends.

Observer hooks are synchronous. Their GoDoc now states that they must return promptly; panic isolation does not provide scheduling or prevent a slow callback from delaying dispatch.

## Pass 4: Standard Go interoperability and ownership

A valid custom http.DefaultTransport implementing RoundTripper previously caused HTTP driver construction to panic because the code asserted *http.Transport. A reproduction now exercises that ordinary application instrumentation pattern. Construction clones a concrete default transport as before, preserving its TLS settings, and otherwise creates an owned standard transport before enforcing the same destination, proxy, redirect, and connection policy.

A second transport reproduction found that inherited DialTLS and DialTLSContext hooks could bypass destination validation for HTTPS. Both hooks are now cleared on the owned transport; HTTPS uses the same checked DialContext as HTTP. Regression tests cover both hooks, private-destination rejection, and preservation of TLS settings.

This is a runtime compatibility change for applications relying on global TLS dial hooks. Preserving those hooks would bypass the mandatory destination boundary. Move certificate, client-certificate, and trust-root settings into the concrete default transport's TLSClientConfig; those settings are cloned. Custom connection routing through global TLS dial hooks is intentionally unsupported and requires a separately designed construction seam if it becomes a concrete requirement.

The library does not adopt arbitrary global wrappers as notification transports. That would let unrelated instrumentation bypass driver destination guarantees. Application-owned Sender decorators and advanced observers remain the current observation boundaries.

The initial review changed no public interfaces, exported configuration fields, persisted formats, module topology, dependencies, or minimum Go versions. The subsequent maintainer-requested transport injection addition is recorded below.

## Deliberate limits and future decisions

- Resolved by the follow-up below: explicit per-driver transport injection now supports custom trust roots, mTLS, proxies, and transport instrumentation. A supplied RoundTripper owns connection policy; the library cannot enforce its default DNS policy inside arbitrary custom dialing code.
- Common Sender results remain error-only. Applications needing positive provider evidence and per-recipient results use typed transports and Dispatcher. A result-producing API for every sender would change the core contract.
- Rich attachments, arbitrary provider parameters, dynamic destination overrides, automatic retries, and durable deduplication are separate requirements. Existing typed renderers and application composition should be exercised before introducing another universal abstraction.

These limits prevent a claim that the API handles every possible provider feature. The common interface is intentionally small, and the extension points preserve materially different delivery semantics.

## Follow-up: explicit HTTP transport injection

The maintainer requested a first-class transport seam after the initial review. HTTPConfig.Transport and webhook.Config.Transport now accept standard http.RoundTripper implementations. Drivers borrow the exact supplied instance, preserve its TLS/proxy/dial settings, and never close it. The default path keeps the destination-checked owned transport.

Client timeouts, redirect rejection, bounded request and response bodies, native acknowledgements, and safe errors remain library-owned. Connection policy and any custom retries or logging belong to the injected transport. AllowPrivate and Resolver apply only to the default path. The runnable [TLS transport example](../examples/transport/main.go) uses an explicit certificate pool without changing global HTTP state.

This adds fields to two exported configuration structs. Keyed literals and constructor signatures are preserved; unkeyed HTTPConfig and webhook.Config literals must add the field or migrate to keyed literals. There are no new dependencies or minimum Go version changes. Tests cover all HTTP driver construction paths, shared transport lifetime, concrete transport immutability, concurrency, timeout/cancellation, redirects, response limits, and redaction.

## Reference checks

The ecosystem comparison remains consistent with [nikoksr/notify's context-aware send interface](https://pkg.go.dev/github.com/nikoksr/notify) and [Shoutrrr's configured sender composition](https://containrrr.dev/shoutrrr/v0.5/getting-started/). These are API references, not adoption recommendations or executed compatibility tests.

Go's [error handling guidance](https://go.dev/blog/error-handling-and-go) supports ordinary error returns and typed inspection. The relevant interoperability contracts are [errors.Is and errors.As](https://pkg.go.dev/errors) and [http.RoundTripper](https://pkg.go.dev/net/http#RoundTripper).

Verification covers the changed regression cases, every repository module, executable examples, race checks, vet, exact library unit coverage, pinned real ntfy/Gotify integration, and deterministic generated documentation. Hosted provider verification remains outside this review's executed scope.

## Release candidate review

Reviewed 2026-11-09 after explicit HTTP transport injection, against baseline 077b5b0. This is a readiness assessment of the implemented library, not a claim that every future provider or payload feature belongs in the first release.

### Consumer ergonomics and compatibility

The primary workflow remains typed driver construction followed by Send(context.Context, Message) error. Destinations, credentials and connection policy stay in configuration; title and body stay in the message. Applications can depend on Sender without importing a specific driver. Fanout, Fallback, SenderFunc and the capture fake use the same interface. No new builders, positional overloads, configuration maps or framework-specific adapters are needed.

Message senders now consistently normalize nil contexts, including direct use of log, webhook, ntfy, Gotify and Mail. Previously those paths could panic while other built-in senders and the facade accepted nil. Regression checks exercise direct calls, not just calls through New. Application code should still propagate its real context. Typed Deliver and user-supplied function adapters retain their existing context contracts.

This review adds no exported names, struct fields, dependencies, module boundaries or minimum Go requirements. It preserves ordinary keyed configuration and constructor usage. It changes runtime behavior only for nil message contexts and contradictory failure evidence described below.

### Failure evidence and recovery

A custom typed transport could return Failed together with ErrOutcomeUnknown. The dispatcher retained the sentinel but trusted the failed status when deciding FirstSuccess fallback and journal retries. The regression reproduced a backup send after an ambiguous primary attempt.

Normalization now promotes a failed outcome to Unknown when the sanitized error contains unknown acceptance, including unclassified errors normalized to that sentinel. Unknown evidence blocks ordered fallback even with ContinueOnRetryable and prevents journal replay. Tests submit the same request twice with plain, retryable, canceled and opaque contradictory errors. Positive Accepted and Delivered evidence keeps its existing semantics. The same normalization protects DeliveryError status inspection.

### Driver protocols and resources

The common Sender suite and provider-specific fixtures cover the implemented drivers, including configured destinations, native acknowledgements, malformed successes, bounds, authentication, cancellation, concurrent use and safe failures. Incident operations retain explicit stable correlation. ntfy and Gotify have separate pinned real-server verification. Hosted providers and paired sessions retain the maintainer-approved local-protocol scope, with external verification pending.

An explicit RoundTripper remains borrowed unchanged and caller-owned. Default transports retain destination checks. Client timeouts, redirects and response body bounds remain library-controlled. The manual HTTPX interoperability check exercised a Slack send and confirmed full request and response body dumping through the injected transport. This check does not certify HTTPX middleware, client-level retry options or all future HTTPX versions.

### Security and documentation

Review covered public error redaction, endpoint and scalar formatting, request bounds, response bounds, default DNS and TLS routing, and ownership at borrowed-resource boundaries. Debugging through a supplied transport remains an explicit caller policy and can expose raw content. Root runtime dependencies remain standard-library-only; Mail and test/tooling dependencies remain isolated in their existing modules.

Executable examples own the README snippets. The release checks execute them, verify output, instrument child processes for coverage and races, and enforce generation parity. The HTTP behavior contract now explicitly distinguishes the default destination policy from caller-supplied transport policy. The README keeps the simple path visible and labels hosted verification honestly.

### Release operations and limits

All five modules are inventoried. Only the root and driver/mail modules are independently published libraries. Tidy parity preserves their intentional local replacements. The release planner reports separate root and Mail tags; there are no published tags at review time. Publication still requires selecting a version, publishing the root, updating Mail's root requirement from v0.0.0, publishing Mail independently, and testing both from an external consumer with GOWORK=off and no replacements. These are publication steps, not API design defects.

Codecov upload configuration could not be inspected with the available credential: the repository variable and secret metadata endpoints returned HTTP 403. Exact unit coverage enforcement is verified independently; a successful Codecov upload is not claimed. This review does not change repository administration settings.

Final checks passed across all five modules: 263 executed unit tests, exact 1251/1251 library statement coverage, race detection, vet, tidy and module-inventory parity, pinned real ntfy/Gotify integration, deterministic documentation generation, and vulnerability scans on Go 1.27.3. The implementation is suitable for an initial release with the documented verification scope. Hosted verification, the inventoried compatibility aliases, durable journals and framework integration remain explicit future work. No release tag or GitHub release is created by this review.

## Provider expansion review

The approved expansion adds 31 native integrations without adding fields or methods to Message or Sender. Provider-specific authentication, fixed destinations and optional native capabilities remain on their driver Config and Driver. Existing keyed configuration and constructor signatures remain valid. The root remains standard-library-only on Go 1.24. APNs, SNS and Web Push introduce three independent optional modules alongside Mail; consumers that do not import them incur no new SDK or cryptographic dependencies.

The ergonomics pass preserved a single synchronous Send for ordinary messages. FCM borrows a token-refresh callback, SNS borrows the AWS SDK publisher, and Mail borrows the existing GoForj Mailer. WhatsApp approved templates use an explicit method rather than hidden message metadata. Partial on-call lifecycles expose only supported actions. Better Stack returns its native incident ID from Create instead of presenting an application key as a provider ID.

The native-protocol pass checks literal paths, authentication and payload field names independently of driver serialization. It corrected PagerTree's case-sensitive Id, Title and Description fields, incident.io's required recovery title and HTTP 202 success acknowledgement, and Matrix's support for domainless room-version-12 IDs. A real Synapse fixture verifies authenticated room sends and stored history. Cryptographic fixtures independently decrypt Web Push messages and verify VAPID; APNs exercises real HTTP/2/TLS and client-certificate authentication. SNS exercises the actual SDK's SigV4 and XML handling with one attempt per send.

The ownership and diagnostics pass checks borrowed transports, publishers and Mailers, bounded payloads, cancellation, safe provider errors and immutable signing material. No credential, endpoint or provider response prose is returned through DeliveryError. Native acceptance does not imply delivery receipts or human acknowledgement. Hosted and paired Signal verification remains pending.

The composition and operations pass compiles all 51 drivers in one temporary external consumer with GOWORK=off and explicit local replacements for unpublished notification modules. This proves composition of the source state, not published module availability. All eleven repository modules participate in validation, and the release planner lists separate tags for all five runtime modules. Current unit and native integration evidence supersedes the counts in earlier review checkpoints. No release is published by this expansion.


## Release readiness follow-up

Four further passes covered consumer compatibility, native delivery evidence, security/resources and publication. The resulting source is a candidate for an initial v0.1.0 release within the documented verification scope. This is not a claim that every hosted provider has been tested live.

The API pass preserved Message, Sender, constructors, configuration, transport ownership and Go 1.24. No public API or persisted-format migration is required. The optional dispatcher and provider-specific capabilities remain separate from ordinary sends.

The native evidence pass reproduced and fixed two failures. WhatsApp previously bounded raw template parameters while JSON escaping and envelope metadata could exceed its documented 16 KiB request limit. It now measures the final serialized request before transport. Regression tests accept exactly 16384 bytes and reject an escaped overflow without making another request. incident.io previously accepted a success acknowledgement containing a different valid incident key. It now compares the response with the specific request key using a per-call closure. Tests cover mismatched trigger/recovery responses and concurrent calls with different keys. These changes tighten runtime handling of oversized templates and contradictory provider evidence without changing valid consumer configuration.

The security and resources pass checked cancellation, bounded inputs/responses, borrowed collaborators, signing material, safe errors and container cleanup. The vulnerability scanner found no reachable vulnerabilities in the repository modules. Module-level advisories in unused dependency packages remain separate from reachable findings; this review does not claim every transitive module is advisory-free. Strict unit coverage is enforced locally and in CI. Codecov setup visibility remains unavailable with the current credential, so a successful upload is not claimed.

The publication pass inventoried all eleven modules and verified the five runtime module tag names. It built isolated candidate archives at a proposed v0.1.0, substituted that root requirement into the four optional archives, and compiled all 51 drivers together from a temporary consumer using GOWORK=off, a fresh module cache and no replacements. This checks candidate packaging and composition. It does not prove public proxy availability or select a release version for the maintainer.

Actual publication still requires the chosen root version, corresponding optional-module requirements/tags, release-facing generated documentation and a clean consumer check through the public module proxy. Those publication changes are intentionally not made by a readiness review. Hosted-provider verification and paired Signal verification remain explicitly pending. Current measured unit and integration results are in unit-results.json and integration-results.json; earlier counts in this document describe historical checkpoints.

Final local verification passed 328 tests with exact 2124/2124 library statement coverage, all eleven modules in race and vet checks, pinned real ntfy/Gotify/Synapse integration, deterministic generated documentation, module inventory and vulnerability scanning. The updated root also passed on Go 1.24.13.
