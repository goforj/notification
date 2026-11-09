# Delivery contracts

`notification.New(driver)` constructs a synchronous notifier. Built-in message senders normalize a nil context to context.Background; application code should pass its caller context to preserve cancellation. `Send(ctx, Message{Title: ..., Body: ...})` validates plain text and calls the configured driver once, returning an error. It has no framework, database, Queue, Auth, or localization dependency.

## Common message path

A `Message` contains only `Title` and `Body`. Destinations are trusted driver configuration: webhook uses `Config.URL`, Mail uses recipient addresses passed to its constructor, and log uses a borrowed logger. Both fields must be valid UTF-8; title allows 256 bytes and rejects line breaks, and body must contain non-whitespace content and fit within 16 KiB. Title is optional for log and webhook; Mail requires a subject.

A nil error means the selected driver handled the message. For log this means local capture, for generic webhook it means HTTP 2xx acceptance, and for Mail it means legacy Mail success. It never proves recipient receipt. No retries, scheduling, fallback, deduplication, or background tasks are implicit in the common path. Errors retain safe classifications while discarding arbitrary collaborator prose. Cancellation during transport may leave provider acceptance uncertain; do not assume a canceled send was unsent.

Application services can depend on `Sender`. `Fanout` and `Fallback` already implement this interface and do not need a `New` wrapper. Use `New` to apply validation and safe errors around application-owned `SenderFunc` implementations; a function adapter itself invokes the supplied function directly. `notificationfake.New()` validates and captures messages without contacting providers. `Messages()` returns a copied slice, and `Respond(err)` scripts a test failure. No fake result implies provider delivery.

Applications own recipient authorization, endpoint verification, preferences, translations, and the choice to send a notification. The optional dispatcher provides the mechanisms for those policies.

## Sender composition and error evidence

`Fanout` and `Fallback` borrow one to 64 concurrency-safe senders and copy their configured lists. Invalid counts panic at construction. Fanout joins every started call and retains target errors in configuration order, with at most eight concurrent calls. Each call has its own concurrency bound; applications control aggregate admission. Fallback advances only after permanent rejection, unauthorized access or unsupported rendering. It stops on unknown acceptance, throttling, retryable failure, cancellation or invalid wiring. A nil error from a capture driver also ends fallback.

`DeliveryError` retains a bounded driver identity, rejection versus uncertainty and up to 24 hours of retry advice. `TargetError.Index` identifies the zero-based composition target. The facade preserves these types through `errors.As` and safe sentinels through `errors.Is`, including joined failures. Custom `Is` methods and standard wrapped causes retain all explicit safe classifications. Reaching the error traversal depth, width, or node budget retains unknown evidence instead of assuming that hidden causes prove rejection. Arbitrary provider prose, URLs and underlying network errors are removed. None of this advice schedules a retry.

## Incident capability

`IncidentSender` is optional. PagerDuty, JSM and legacy Opsgenie support explicit Trigger, Acknowledge and Resolve operations with a stable `IncidentKey`. Keys are bounded to 255 bytes and must use application-owned scoped identities. Display names must not determine correlation. Incident titles and bodies follow Message validation, with a required title and additional provider limits. Empty severity and source use configured defaults. Ordinary Send on these drivers requires a fixed configured key.

JSM resolves the alias to a native alert ID before acknowledgement or closure. Lookup and mutation are two separate network operations. Asynchronous request acceptance does not prove the alert has been created or changed. No driver polls or waits for workflow completion.

## Advanced intent and planning

Construct one `Dispatcher` with `NewDispatcher` per authoritative application and tenant scope. Scope never comes from request values or context. Requests contain opaque recipient references and typed scalar values. A trusted `Planner` resolves endpoints, locale, IANA time zone, and suppression or deferral. It must classify every requested recipient, cannot change scope, and cannot mix a disposition and immediate delivery for the same recipient.

`NewMemoryPlanner` copies fixed route plans and is useful for configured system destinations. It applies the notifier's scope to those trusted plans. User lookup and tenant authorization belong in an application `PlannerFunc`; the memory planner does not supply those checks.

Limits are 100 unique recipients, 400 planned deliveries, 64 values, 8 KiB per value, 32 KiB total scalar content, 16 unique tags, and 4 KiB per resolved address. UTF-8 text payloads allow a 256-byte title and a 16 KiB body. Mail payloads allow a 256-byte subject and 32 KiB combined text and HTML. Invalid input fails before drivers run.

## Per-recipient policies

Every recipient must satisfy its route policy. `AllRequired` requires every planned delivery to be accepted or delivered. `Any` attempts all planned channels and requires one acceptance per recipient. `FirstSuccess` follows plan order separately for each recipient; an unknown outcome stops fallback to avoid duplicating an ambiguous send. A retryable failure stops fallback unless `ContinueOnRetryable` explicitly allows advancement. `BestEffort` tolerates transport failures and preserves their evidence.

Suppression is acceptable only with `AllowSuppressed`. Deferral always leaves the request pending. None of these states proves provider acceptance. The local log driver returns unknown evidence, so a log-only route normally uses `BestEffort`. The capture fake returns no provider outcomes and never claims successful execution by default.

## Identity and retries

`Dispatcher.Send` performs no retries, sleeps, scheduling, or background work. Applications own backoff and must respect bounded `RetryAfter` advice. Drivers and renderers are paired through `NewAdapter[P]`; incompatible payload types fail to compile.

Without a journal, each call is independent. `WithJournal(NewMemoryJournal())` enables process-local arbitration by application, tenant, and request ID. Concurrent identical requests return `ErrInProgress`. Different caller intent returns `ErrConflict`. Recipient order, tag order, and value order do not affect intent. Route and renderer revisions are execution snapshots, not caller intent.

The memory journal retains the original route, plan, adapter revisions, and results. While the route remains unsatisfied, an explicit repeat of `Send` retries transient rejected attempts and unattempted canceled work, preserving accepted and permanently failed deliveries. Unknown outcomes, including transport cancellation after possible dispatch, are never automatically retried. Explicit unknown-acceptance error evidence overrides a failed outcome from a custom driver; conflicting evidence cannot enable fallback or journal retries. Unavailable pinned adapter revisions return `ErrConflict` before transport execution. A successful policy is terminal even when optional attempts failed transiently. Completed requests return copied prior outcomes.

The journal retains suppression and deferral classifications as the result of that planning snapshot. It does not reevaluate quiet hours or schedule deferred work. An application that needs future preference evaluation must own a separate scheduling workflow and follow-up request identity. Durable deferral recovery remains part of the framework proposal, not this release.

Memory state cannot survive restart and is not suitable for durable Queue jobs. `Forget(scope, id)` explicitly ends retention for a non-active identity and permits future delivery with that ID. The application must bound journal retention using its own policy. There is no exactly-once promise: provider acceptance can precede process failure, and generic webhooks and legacy Mail do not prove provider idempotency.

## Ownership and observations

Simple messages are passed by value. Dispatcher catalogs, planner plans, request slices, journal results, and fake captures are copied. Callers must not mutate inputs concurrently with a call. Injected collaborators must support concurrent use and remain caller-owned. The library does not close borrowed drivers, loggers, or mailers. Close the webhook driver's idle connections when its owner shuts down.

Observer hooks must return promptly and be concurrency-safe. Dispatch recovers observer panics; it does not create a buffering goroutine. Observations expose only bounded route, channel, driver, status, and duration. Public errors discard collaborator prose and preserve safe cancellation identities. Endpoint wrappers redact ordinary formatting and JSON; `Reveal` is an intentional privileged accessor. Renderers and callers can inspect semantic values explicitly, so do not log request snapshots.

## HTTP behavior

Built-in HTTP drivers use trusted configured HTTP or HTTPS URLs. The advanced webhook adapter also accepts trusted planner-resolved endpoints. The default transport disables environment proxies and validates all DNS answers at connection time before dialing a resolved IP. The client disables redirects even when a custom transport is supplied. Default policy rejects private and special addresses; `AllowPrivate` explicitly permits internal services and local tests. Configure it at construction, never from an untrusted request. Production credentials should travel over HTTPS.

Generic JSON sends `{ "title": "...", "body": "..." }` and classifies 2xx as provider acceptance. Slack-compatible mode sends plain text with `mrkdwn: false` and additionally requires the `ok` acknowledgement. 429 is throttled, 401/403 unauthorized, 5xx retryable, and other non-2xx responses permanent. Transport uncertainty remains unknown. Responses are bounded to 64 KiB plus one detection byte; provider bodies never enter errors. No generic idempotency header is advertised or silently assumed.

## Mail behavior

The optional `driver/mail` module configures recipients with `New(mailer, addresses...)` and delegates to `github.com/goforj/mail` v0.3.1. Mail owns sender defaults, recipients, header validation, providers, and transport. A nil legacy Mail error maps to accepted, never delivered, with idempotency support unproven. Legacy errors map to unknown because the error-only interface cannot reliably prove provider rejection. Cancellation identities remain inspectable.

## Implementation boundary

This repository implements the synchronous foundation of the [framework proposal](designs/framework-proposal.md), under the singular module name requested by the maintainer. It does not yet implement durable SQL journals, Queue submission or claims, generated GoForj components, inbox storage, localization adapters, the 67 remaining compatibility-alias protocols, or demo migration. The common Message API and optional NewDispatcher boundary supersede the proposal's application-facing constructor. The proposal is historical planning context; these contracts describe the implemented API.

## Application decorators and observers

Implement `Sender` or use `SenderFunc` to add application-owned metrics, admission control, or tracing around a configured sender. Preserve the returned error and caller context. Decorators do not need a new library option or an advanced dispatcher. Keep message text and raw collaborator errors out of logs.

Advanced `Observer` hooks run synchronously and must return promptly. Panic isolation does not create asynchronous execution or prevent a slow hook from delaying the caller.

## HTTP transport ownership

Every HTTP provider accepts `notification.HTTPConfig.Transport`, a standard `http.RoundTripper`. Webhook's existing flat configuration exposes `webhook.Config.Transport`. A nil field selects the library-owned default. A supplied transport is borrowed unchanged, can be shared across drivers, and must support concurrent calls and request cancellation. Construction performs no I/O and does not clone or mutate it. Driver Close never closes a borrowed transport or calls its CloseIdleConnections method. The caller owns its complete lifetime.

The library still creates its own http.Client with the configured timeout and redirect rejection. Request URL syntax, request and response body bounds, provider acknowledgement parsing, cancellation handling, and error redaction remain active. The library adds no retries. An injected transport's own retries and logging are application policy, including any logging of credentials or provider metadata.

Connection-time destination checks belong to the default transport. It clones a concrete global *http.Transport, retaining TLS certificate settings while replacing dial hooks, disabling proxies, and enforcing connection and response-header bounds. If the global default is another RoundTripper, it creates an owned standard transport. It never adopts a global wrapper implicitly.

An explicitly injected transport owns all dialing, DNS, private-address policy, TLS, and proxy behavior. Transport-level settings such as response-header limits also remain caller-owned. AllowPrivate and Resolver configure only the library-owned default; they do not constrain an opaque custom RoundTripper. In particular, the library cannot claim DNS-rebinding protection for a connection made by arbitrary caller code. This is an explicit trusted construction choice, never a message-controlled setting. Wrap or configure the supplied transport to enforce the destination policy your application needs.

The new fields preserve keyed configuration literals and all constructor signatures. Unkeyed HTTPConfig and webhook.Config literals need the additional field or should migrate to keyed literals. Module requirements and minimum Go versions are unchanged.
