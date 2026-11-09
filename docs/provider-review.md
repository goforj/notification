# Provider coverage and Go notification API review

Reviewed 2026-11-08. This review covers Uptime Gopher's authoritative GoForj demo templates and five Go notification libraries. It recommends the next implementation boundary; the proposed additions below are not shipped APIs or verified provider integrations.

## Recommendation

Keep `New(driver).Send(ctx, Message)` as the common API. Broad provider coverage fits that interface well. Add lightweight sender composition, safe typed errors, and an optional incident lifecycle capability before expanding the driver catalog. The existing planner-based Dispatcher remains useful for application recipients and preferences, but configured monitoring destinations should not need it merely to send to Slack and Telegram together.

Treat every one of the demo's **88 provider IDs** as an explicit extraction target. Account for aliases, gateway adapters, Mail delegation, and service retirement individually. A provider is advertised as natively supported only after its actual protocol and applicable contract tests pass.

## What Uptime Gopher actually contains

The [complete generated inventory](providers/uptime-gopher.md) and [machine-readable snapshot](providers/uptime-gopher.json) contain every ID, constructor, source adapter, UI visibility, current library surface, source commit, and file hashes.

| Surface | Observed count | Meaning |
| --- | ---: | --- |
| Backend provider IDs | 88 | Names accepted by the backend catalog |
| Dedicated provider IDs | 21 | Names routed to provider-specific code |
| Dedicated adapters | 20 | Email and SMTP share an implementation |
| Generic compatibility aliases | 67 | All send the same provider/text/message JSON envelope |
| Frontend choices | 26 | All 21 dedicated IDs plus five compatibility aliases |

The five UI-visible generic aliases are Grafana OnCall, Google Sheets, Rocket.Chat, SendGrid, and Splunk. The other 62 backend IDs are absent from the current frontend selector. These counts describe source code, not a test result from deployed provider accounts.

The dedicated extraction targets are DingTalk (`dingding`), Discord, Email/SMTP, Feishu, Google Chat, Gotify, Jira Service Management, LINE, Log, ntfy, Opsgenie, PagerDuty, Slack, Teams, Telegram, Twilio, WAHA, generic webhook, WeCom, and Whapi. The inventory separately lists all 67 generic aliases, including regional SMS, push, incident, chat, gateway, and automation services.

Current `notification` building blocks cover Log, generic webhook, Slack-compatible webhook, and Mail composition for Email/SMTP. Sibling Mail packages also provide a route for SendGrid and Resend; that composition was identified from source, not newly integrated or certified by this review. Brevo would require a Mail-side implementation rather than a second email transport inside notification.

## Concrete extraction findings

1. **A catalog alias is not a native adapter.** `NewGenericWebhookCompatProvider` sends one application-defined envelope regardless of the provider name. Twilio, Matrix, Web Push, Nostr, and the other services require their own authentication, wire format, and acknowledgement rules. A configured intermediary may translate that envelope, but that is gateway compatibility.
2. **Dedicated code also needs protocol review.** ntfy and Gotify priorities are stored as strings and placed directly into JSON. ntfy documents an integer priority; Gotify's request model also uses an integer. This is a source/schema mismatch to fix during extraction, not evidence from a live failure. [ntfy JSON contract](https://docs.ntfy.sh/publish/#publish-as-json), [Gotify request model](https://github.com/gotify/server/blob/master/model/message.go).
3. **Recovery identity must survive renaming.** Opsgenie and JSM prefer monitor names over monitor IDs for their alert aliases. Renaming a monitor between trigger and recovery can change the correlation key. Use stable scoped monitor/incident identity; display names belong in presentation. PagerDuty explicitly requires consistent correlation between trigger, acknowledge, and resolve operations. [PagerDuty alert events](https://docs.pagerduty.com/developer/send-alert-event).
4. **HTTP status is insufficient for every service.** The shared priority client accepts any 2xx response and does not decode acknowledgement bodies. Each extracted driver needs native success/error-body classification and malformed-success tests. Keep provider bodies out of public errors.
5. **Preserve app behavior deliberately.** The demo formats monitor fields, selects database channel IDs, stores attempt history, and owns Queue retries. The library owns transports. Extraction needs an application projection into Message or incident operations, with compatibility fixtures for stored IDs, credentials/configuration aliases, subject prefixes, destination selection, and historical statuses. Generic webhook payload shape is also a persisted integration contract.
6. **Acceptance needs a migration projection.** The current manager records `delivered` for every nil provider error, including local logging. Preserve historical rows while introducing precise new acceptance/capture classifications through an explicit application migration.

Source evidence is in the inventory's linked constructor and adapter paths, particularly `manager.go.tmpl`, `generic_webhook_compat_provider.go.tmpl`, `priority_provider_common.go.tmpl`, `ntfy_provider.go.tmpl`, `gotify_provider.go.tmpl`, `opsgenie_provider.go.tmpl`, and `jira_service_management_provider.go.tmpl`.

## Go ecosystem comparison

These are source reviews, not adoption endorsements or executed conformance results. Exact commits, inspected paths, and release metadata are recorded in [research sources](providers/research-sources.json). Release requirements and branch requirements are kept separate.

| Library | Useful design evidence | Implication for notification |
| --- | --- | --- |
| [nikoksr/notify](https://github.com/nikoksr/notify) | Configured services share a small send interface that takes context; a notifier fans out to multiple services. The inspected implementation waits for services but exposes the first errgroup error. | Keep the simple interface and typed provider construction. Preserve all safe target failures when adding fanout, rather than copying its aggregate error behavior. |
| [containrrr/shoutrrr](https://github.com/containrrr/shoutrrr) | Service URLs make transport selection configuration-driven. The inspected original top-level API has no caller context. | Configuration-driven selection is useful for the demo, but credentials in connection strings should not become the default Go API. Keep context explicit. |
| [nicholas-fedor/shoutrrr](https://github.com/nicholas-fedor/shoutrrr) | The fork adds context-aware sends, aligned per-target errors, resource closure, transport injection, timeout policy, and schema registration. | Adopt the ideas of target-level diagnostics, shared connection policy, and provider metadata. Keep typed configs as the primary interface. Its broader machinery need not enter the root sender. |
| [d1n-go/notify](https://github.com/d1n-go/notify) | A message-based interface composes with Multi and Fallback wrappers. It also offers explicit async and throttling wrappers. | Lightweight composition is the clearest missing ergonomic piece. Preserve our cancellation, uncertainty, and Queue ownership rules instead of importing its async behavior. |
| [gopackx/go-notification](https://github.com/gopackx/go-notification) | Laravel-style Notifiable/Via routing and channel-specific message builders support user notifications. Its default config enables asynchronous dispatch and retries. | Useful reference for the optional application-recipient layer. These lifecycle and policy requirements should not return to the common monitoring send path. |

The latest GitHub releases observed were nikoksr/notify v1.6.0, original Shoutrrr v0.8.0, and the fork v0.23.0. Their release go.mod requirements were Go 1.25.10, 1.18, and 1.27 respectively. The last two reviewed repositories had no GitHub Release entry; that does not mean they have no tags. Our root remains Go 1.24. Wrapping a newer library would require an explicitly isolated adapter and a concrete minimum-version decision, not a silent root dependency increase. [notify release module](https://github.com/nikoksr/notify/blob/v1.6.0/go.mod), [original Shoutrrr release module](https://github.com/containrrr/shoutrrr/blob/v0.8.0/go.mod), [fork release module](https://github.com/nicholas-fedor/shoutrrr/blob/v0.23.0/go.mod).

## API direction

### Preserve the common message API

A title and body remain sufficient for ordinary delivery. Provider authentication, recipients, topics, channel/thread IDs, and fixed transport options belong in typed driver configuration. Prefer provider-specific packages such as `driver/telegram` and `driver/discord`; do not grow `webhook.Format` into a switch over dozens of services.

Keep existing small constructors where they are clear. New constructors validate configuration without contacting the provider. Raw string maps remain an application configuration boundary, not the public sending API. Rich channel payloads stay in typed provider APIs or the advanced renderer seam. Add portable Message fields only when representative drivers prove their semantics; do not add a universal provider-options map.

### Add simple sender composition

Implemented shape:

```go
sender := notification.New(notification.Fanout(
    slackDriver,
    telegramDriver,
))
```

Fanout should bound concurrency, join per-call work, attempt all configured targets unless canceled, and retain all safe failures. A separate Fallback wrapper should preserve deterministic order and stop on uncertain acceptance. Retryable advancement must be deliberate. Neither wrapper should require opaque recipient references, locale, revisions, a journal, or a planner merely to use configured destinations.

The error-only Sender contract means nil is handled, which can include log capture. Composition must document that distinction and must not claim provider acceptance from nil alone. Per-recipient policy and delivery-state recovery remain Dispatcher responsibilities.

### Preserve structured errors through the facade

The simple path now preserves Outcome.RetryAfter and bounded joined failures through DeliveryError and TargetError. These retain provider/target classification and retry advice. They support errors.Is and errors.As, and retain multiple failures without exposing tokens, URLs, recipient addresses, or server bodies. The ordinary Send signature can remain error-only.

### Model incident lifecycle as an optional capability

Plain messages cannot express trigger versus resolve, stable incident correlation, severity, and source. PagerDuty, Opsgenie, and JSM need those concepts. Use a small domain-neutral incident capability alongside Sender, with typed incident data and explicit trigger/acknowledge/resolve operations. It should not require every email/chat/SMS driver to implement an incident workflow or make monitoring fields mandatory in Message.

The implementation exposes IncidentSender alongside Sender and exercises it against PagerDuty and JSM protocol fixtures. Map monitor down/up to explicit incident actions in the demo, using server-owned stable scope and keys. Keep provider deduplication identity separate from a notification request ID.

### Share transport mechanics and provider facts

Extract an internal bounded HTTP transport layer from the current webhook mechanism so new drivers reuse destination policy, timeouts, redirects/proxies, response bounds, safe classification, and cancellation. Provider adapters still own authentication, encoding, and acknowledgement parsing. Mail continues to own all email transport. Isolate SDK dependencies only where weight, maintenance, or Go requirements justify another module.

Maintain one provider descriptor catalog for stable IDs, aliases, configuration/secret fields, capabilities, lifecycle status, and verified test scope. Generate documentation and demo UI choices from that catalog during integration. This makes the current 88-versus-26 difference explicit and maintainable; it does not require exposing every legacy option in the UI. Do not load arbitrary provider code from runtime data.

## Current service lifecycle constraints

- Teams work should target Workflows and current message/card schemas. Microsoft documents retirement of connector-based incoming webhooks in May 2026. [Microsoft Teams webhook documentation](https://learn.microsoft.com/en-us/microsoftteams/platform/webhooks-and-connectors/how-to/add-incoming-webhook).
- Opsgenie is an existing-customer compatibility target, with shutdown/end of support scheduled for April 5, 2027. Prioritize JSM migration alongside it. [Atlassian migration notice](https://www.atlassian.com/software/opsgenie/migration).
- Grafana OnCall OSS is archived as of March 24, 2026. Preserve its stored ID and distinguish legacy self-hosted compatibility from a current cloud/IRM integration. [Grafana OnCall notice](https://grafana.com/docs/oncall/latest/intro/).
- LINE Notify was terminated on March 31, 2025. The demo already targets LINE Messaging API, which should remain the explicit protocol. [LINE termination and replacement guidance](https://developers.line.biz/en/news/2025/04/01/line-notify/).

Exhaustive coverage means accounting for these IDs and their migration paths. It cannot mean promising delivery through a shut-down upstream service.

## Implementation sequence

1. Land safe typed errors, sender composition, a shared transport seam, and a PagerDuty/JSM incident prototype, while preserving New(driver).Send(ctx, Message).
2. Extract the 20 dedicated demo adapters in small tested groups: chat/webhooks, push, messaging/SMS, incident systems, and Mail composition. Correct protocol mismatches rather than freezing them into compatibility fixtures.
3. Implement every remaining catalog ID against its real protocol or an explicitly named gateway adapter. Track each of the 67 aliases individually. Review retirement and native support before labeling it supported.
4. Prove each driver with the shared Sender contract, native request/acknowledgement fixtures, authentication and failure cases, cancellation, redaction, and applicable real-backend/provider integration coverage. Record hosted checks as pending when no authorized sandbox is available.
5. Integrate the library into Uptime Gopher through an explicit application adapter and configuration/history migration. Generate the frontend provider list and validate the largest generated composition under /tmp.

Composition, safe errors, shared HTTP mechanics, incident lifecycles and all 20 distinct dedicated demo adapters are now implemented. See the [driver catalog](providers/drivers.json), the [generated support matrix](../README.md#drivers), and [real-server evidence](integration-results.json). Hosted providers and paired WhatsApp sessions have local protocol tests with external verification pending, as explicitly authorized by the maintainer.

The 67 generic aliases remain individually inventoried protocol work. Framework integration and persisted configuration/history migration remain separate work; no demo configuration or stored provider ID was changed here.
