# Notification API review

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

The library does not adopt arbitrary global wrappers as notification transports. That would let unrelated instrumentation bypass driver destination guarantees. Application-owned Sender decorators and advanced observers remain the current observation boundaries.

No public interfaces, exported configuration fields, persisted formats, module topology, dependencies, or minimum Go versions change in this review.

## Deliberate limits and future decisions

- HTTPConfig exposes timeouts, destination policy, and DNS resolution, but no per-driver HTTP client, custom CA, or mTLS injection. A future explicit construction seam should preserve destination checks and TLS ownership. Adding fields to exported configuration structs can break unkeyed literals, so this requires a separate compatibility decision and a concrete consumer case.
- Common Sender results remain error-only. Applications needing positive provider evidence and per-recipient results use typed transports and Dispatcher. A result-producing API for every sender would change the core contract.
- Rich attachments, arbitrary provider parameters, dynamic destination overrides, automatic retries, and durable deduplication are separate requirements. Existing typed renderers and application composition should be exercised before introducing another universal abstraction.

These limits prevent a claim that the API handles every possible provider feature. The common interface is intentionally small, and the extension points preserve materially different delivery semantics.

## Reference checks

The ecosystem comparison remains consistent with [nikoksr/notify's context-aware send interface](https://pkg.go.dev/github.com/nikoksr/notify) and [Shoutrrr's configured sender composition](https://containrrr.dev/shoutrrr/v0.5/getting-started/). These are API references, not adoption recommendations or executed compatibility tests.

Go's [error handling guidance](https://go.dev/blog/error-handling-and-go) supports ordinary error returns and typed inspection. The relevant interoperability contracts are [errors.Is and errors.As](https://pkg.go.dev/errors) and [http.RoundTripper](https://pkg.go.dev/net/http#RoundTripper).

Verification covers the changed regression cases, every repository module, executable examples, race checks, vet, exact library unit coverage, pinned real ntfy/Gotify integration, and deterministic generated documentation. Hosted provider verification remains outside this review's executed scope.
