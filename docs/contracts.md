# Delivery contracts

`notification` is a standalone synchronous library. It has no framework, database, Queue, Auth, or localization dependency. Applications own recipient authorization, endpoint verification, preferences, translations, and the choice to send a notification.

## Intent and planning

Construct one `Notifier` per authoritative application and tenant scope. Scope never comes from request values or context. Requests contain opaque recipient references and typed scalar values. A trusted `Planner` resolves endpoints, locale, IANA time zone, and suppression or deferral. It must classify every requested recipient, cannot change scope, and cannot mix a disposition and immediate delivery for the same recipient.

`NewMemoryPlanner` copies fixed route plans and is useful for configured system destinations. It applies the notifier's scope to those trusted plans. User lookup and tenant authorization belong in an application `PlannerFunc`; the memory planner does not supply those checks.

Limits are 100 unique recipients, 400 planned deliveries, 64 values, 8 KiB per value, 32 KiB total scalar content, 16 unique tags, and 4 KiB per resolved address. UTF-8 text payloads allow a 256-byte title and a 16 KiB body. Mail payloads allow a 256-byte subject and 32 KiB combined text and HTML. Invalid input fails before drivers run.

## Per-recipient policies

Every recipient must satisfy its route policy. `AllRequired` requires every planned delivery to be accepted or delivered. `Any` attempts all planned channels and requires one acceptance per recipient. `FirstSuccess` follows plan order separately for each recipient; an unknown outcome stops fallback to avoid duplicating an ambiguous send. A retryable failure stops fallback unless `ContinueOnRetryable` explicitly allows advancement. `BestEffort` tolerates transport failures and preserves their evidence.

Suppression is acceptable only with `AllowSuppressed`. Deferral always leaves the request pending. None of these states proves provider acceptance. The local log driver returns unknown evidence, so a log-only route normally uses `BestEffort`. The capture fake returns no provider outcomes and never claims successful execution by default.

## Identity and retries

`Send` performs no retries, sleeps, scheduling, or background work. Applications own backoff and must respect bounded `RetryAfter` advice. Drivers and renderers are paired through `NewAdapter[P]`; incompatible payload types fail to compile.

Without a journal, each call is independent. `WithJournal(NewMemoryJournal())` enables process-local arbitration by application, tenant, and request ID. Concurrent identical requests return `ErrInProgress`. Different caller intent returns `ErrConflict`. Recipient order, tag order, and value order do not affect intent. Route and renderer revisions are execution snapshots, not caller intent.

The memory journal retains the original route, plan, adapter revisions, and results. While the route remains unsatisfied, an explicit repeat of `Send` retries transient rejected attempts and unattempted canceled work, preserving accepted and permanently failed deliveries. Unknown outcomes, including transport cancellation after possible dispatch, are never automatically retried. Unavailable pinned adapter revisions return `ErrConflict` before transport execution. A successful policy is terminal even when optional attempts failed transiently. Completed requests return copied prior outcomes.

The journal retains suppression and deferral classifications as the result of that planning snapshot. It does not reevaluate quiet hours or schedule deferred work. An application that needs future preference evaluation must own a separate scheduling workflow and follow-up request identity. Durable deferral recovery remains part of the framework proposal, not this release.

Memory state cannot survive restart and is not suitable for durable Queue jobs. `Forget(scope, id)` explicitly ends retention for a non-active identity and permits future delivery with that ID. The application must bound journal retention using its own policy. There is no exactly-once promise: provider acceptance can precede process failure, and generic webhooks and legacy Mail do not prove provider idempotency.

## Ownership and observations

Notifier catalogs, planner plans, request slices, journal results, and fake captures are copied. Callers must not mutate inputs concurrently with a call. Injected collaborators must support concurrent use and remain caller-owned. The library does not close borrowed drivers, loggers, or mailers. Close the webhook driver's idle connections when its owner shuts down.

Observer hooks must return promptly and be concurrency-safe. Dispatch recovers observer panics; it does not create a buffering goroutine. Observations expose only bounded route, channel, driver, status, and duration. Public errors discard collaborator prose and preserve safe cancellation identities. Endpoint wrappers redact ordinary formatting and JSON; `Reveal` is an intentional privileged accessor. Renderers and callers can inspect semantic values explicitly, so do not log request snapshots.

## HTTP behavior

The webhook driver accepts trusted planner-resolved HTTP or HTTPS URLs, disables redirects and environment proxies, and validates all DNS answers at connection time before dialing a resolved IP. Default policy rejects private and special addresses; `AllowPrivate` explicitly permits internal services and local tests. Configure it at construction, never from an untrusted request. Production credentials should travel over HTTPS.

Generic JSON sends `{ "title": "...", "body": "..." }` and classifies 2xx as provider acceptance. Slack-compatible mode sends plain text with `mrkdwn: false` and additionally requires the `ok` acknowledgement. 429 is throttled, 401/403 unauthorized, 5xx retryable, and other non-2xx responses permanent. Transport uncertainty remains unknown. Responses are bounded to 64 KiB plus one detection byte; provider bodies never enter errors. No generic idempotency header is advertised or silently assumed.

## Mail behavior

The optional `driver/mail` module delegates to `github.com/goforj/mail` v0.3.1. Mail owns sender defaults, recipients, header validation, providers, and transport. A nil legacy Mail error maps to accepted, never delivered, with idempotency support unproven. Legacy errors map to unknown because the error-only interface cannot reliably prove provider rejection. Cancellation identities remain inspectable.

## Implementation boundary

This repository implements the synchronous foundation of the [framework proposal](designs/framework-proposal.md), under the singular module name requested by the maintainer. It does not yet implement durable SQL journals, Queue submission or claims, generated GoForj components, inbox storage, localization adapters, provider packs, or demo migration. The proposal is historical planning context; these contracts describe the implemented API.
