<p align="center">
  <img src="docs/assets/logo.png" width="300" alt="notification logo">
</p>

<p align="center">One Go API for chat, email, SMS, push, and on-call alerts.</p>

<p align="center">
  <a href="https://github.com/goforj/notification/actions/workflows/ci.yml"><img src="https://github.com/goforj/notification/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://github.com/goforj/notification/actions/workflows/security.yml"><img src="https://github.com/goforj/notification/actions/workflows/security.yml/badge.svg" alt="Security"></a>
  <img src="docs/assets/badges/status.svg" alt="Unreleased">
  <img src="docs/assets/badges/go.svg" alt="Go 1.24 or newer">
  <a href="LICENSE"><img src="docs/assets/badges/license.svg" alt="MIT license"></a>
  <a href="docs/unit-results.json"><img src="docs/assets/badges/coverage.svg" alt="Unit statement coverage"></a>
  <a href="docs/unit-results.json"><img src="docs/assets/badges/tests.svg" alt="Executed unit tests"></a>
{{integration-badge}}
</p>

<p align="center"><a href="#features">Features</a> · <a href="#quick-start">Quick Start</a> · <a href="#drivers">Drivers</a> · <a href="#runnable-examples">Examples</a> · <a href="#api-index">API</a></p>

## Features

{{features}}

## Quick Start

Requires Go 1.24+. This is an unreleased library. From the local checkout:

```sh
cd ~/code/notification
go run ./examples/quickstart
```

Configure a driver once, then send a message:

{{snippet:quickstart}}

The log driver captures locally. A nil error means the driver handled the message; it never proves recipient delivery. Send performs no retries or background work.

## Drivers

{{drivers}}

All drivers have local tests. ntfy, Gotify and Matrix also pass real-server tests; hosted and paired-session verification remains pending for other providers. See [driver configuration and delivery contracts](docs/drivers.md) for details and [integration evidence](docs/integration-results.json) for exact server versions.

Mail reuses your configured GoForj Mailer for SMTP, SendGrid, Resend, Mailgun, Postmark, or Amazon SES (a separate Mail module). Teams uses Workflows, LINE uses the Messaging API, and Opsgenie provides legacy integration support.

The root module uses only the standard library. Mail, APNs, SNS and Web Push are optional modules. Mail uses goforj/mail v0.3.1. Drivers own their configured destinations; a `Message` contains only content. Errors preserve safe cancellation and provider classifications without returning provider prose. See [delivery contracts](docs/contracts.md) for bounds and ownership.

## Installation

Install the verified unreleased revision through the public Go module proxy:

```sh
go get github.com/goforj/notification@v0.0.0-20261109215621-1a1b304b2d8a
```

Most HTTP providers and the log driver are included in the root module:

```go
import "github.com/goforj/notification"
import "github.com/goforj/notification/driver/log"
import "github.com/goforj/notification/driver/webhook"
```

For Slack-compatible transport, configure the trusted URL once:

```go
driver, err := webhook.New(webhook.Config{
    URL:    configuredWebhookURL,
    Format: webhook.Slack,
})
```

Install the optional Mail adapter at the same revision:

```sh
go get github.com/goforj/notification/driver/mail@v0.0.0-20261109215621-1a1b304b2d8a
```

It borrows an ordinary `*mail.Mailer` and configured recipients. Mail continues to own sender defaults, email composition, and transport. APNs, SNS and Web Push follow the same optional-module pattern: substitute `apns`, `sns` or `webpush` for `mail` in the command above. Their cryptographic or AWS SDK dependencies stay outside the root module. See [provider capabilities](docs/provider-capabilities.md) for supported operations and protocol limits.

## Runnable Examples

| Program | Run from the repository root | Demonstrates |
| --- | --- | --- |
| [Quick start](examples/quickstart/main.go) | `go run ./examples/quickstart` | A title and body through the log driver |
| [Webhook](examples/webhook/main.go) | `go run ./examples/webhook` | A message sent to a configured local HTTP provider |
| [Mail](examples/mail/main.go) | `cd examples/mail && go run .` | A real Mail Resend driver and Slack composed through Fanout against local fixtures |
| [Provider catalog](examples/providers/main.go) | `go run ./examples/providers` | Native root-provider configurations and a push/chat fanout |
| [WhatsApp templates](examples/whatsapp/main.go) | `go run ./examples/whatsapp` | Explicit approved-template sends against a local fixture |
| [APNs](examples/apns/main.go) | `cd examples/apns && go run .` | Signed alerts over real HTTP/2 and TLS |
| [SNS](examples/sns/main.go) | `cd examples/sns && go run .` | A borrowed AWS SDK client and a native local response |
| [Web Push](examples/webpush/main.go) | `cd examples/webpush && go run .` | Encrypted browser push and VAPID authentication |
| [Fake](examples/fake/main.go) | `go run ./examples/fake` | Application testing through Sender |
| [Composition](examples/composition/main.go) | `go run ./examples/composition` | Fanout, conservative fallback and safe retry advice |
| [Custom HTTP transport](examples/transport/main.go) | `go run ./examples/transport` | Caller-owned transport and TLS trust roots |
| [Decorator](examples/decorator/main.go) | `go run ./examples/decorator` | Application metrics through SenderFunc |
| [Incident](examples/incident/main.go) | `go run ./examples/incident` | Stable lifecycle keys with a local PagerDuty protocol fixture |
| [Advanced fallback](examples/fallback/main.go) | `go run ./examples/fallback` | Optional recipient planning, fallback, and identity |

The runner discovers and executes every program and verifies the expected output immediately after each print call. Child programs participate in coverage and race runs. Local tests verify webhook wire behavior and Mail composition; they make no hosted-provider delivery claim.

## API Index

{{api-index}}

## Webhook delivery

The URL is driver configuration. `AllowPrivate` below permits the local test server; the default destination policy rejects private and special addresses.

{{snippet:webhook}}

## Custom HTTP transport

Set `HTTPConfig.Transport` to any concurrency-safe `http.RoundTripper`. Drivers borrow it unchanged; the caller owns its configuration and cleanup. This example uses a local TLS server and a certificate pool named `roots`:

{{snippet:transport}}

Webhook exposes the same option directly as `webhook.Config.Transport`. The library retains its client timeout, redirect rejection, response body bounds, and safe errors. With an injected transport, dialing, DNS, TLS, proxies, and retries belong to that transport; `AllowPrivate` and `Resolver` apply only to the default transport. See [transport ownership](docs/contracts.md#http-transport-ownership) for details.

## Composition

Use ordinary senders for application-level fanout or conservative fallback. Fanout attempts every target with at most eight concurrent calls. Fallback advances only after permanent rejection, unauthorized access or unsupported rendering; uncertainty and transient failures stop it to avoid accidental duplicates. Both return a `Sender` directly, borrow one to 64 configured senders, and never retry. For transient errors, check `ErrOutcomeUnknown` before considering a retry; server failure can follow provider acceptance. A `New` wrapper is optional around a composition; use it when wrapping a custom handler that needs common validation and safe errors.

{{snippet:composition}}

## Custom senders

Use `SenderFunc` to adapt an existing service or decorate a configured sender with application metrics or tracing. Preserve the caller's context and returned error. `New` applies validation and safe errors around the custom handler, so invalid messages do not reach it.

{{snippet:decorator}}

## Incident lifecycle

PagerDuty, JSM, legacy Opsgenie, SIGNL4, PagerTree and ilert additionally implement `IncidentSender`. Construct keys from stable scoped IDs and use the same key for trigger, acknowledge and resolve. Ordinary `Send` requires a fixed configured key on these drivers. JSM and Opsgenie return asynchronous acceptance, without polling for completion.

{{snippet:incident}}

## Mail delivery

Reuse the application-owned GoForj Mailer and configure notification recipients once. Mail owns provider credentials, sender defaults, MIME and transport. This example runs real Resend and Slack drivers against local HTTP fixtures, then composes them through Fanout. It sends no hosted email or chat message.

{{snippet:mail}}

See [Mail integration](docs/mail.md) for provider choices, rich email, lifecycle and verification details.

## Native provider configuration

All providers share the same `Sender` interface. Configure destinations once and compose senders with `Fanout` or `Fallback`. The runnable catalog includes each new root driver with readable configuration and local protocol fixtures.

{{snippet:providers}}

## Application push

APNs owns and refreshes its provider JWT, or borrows an explicitly configured certificate transport. SNS borrows an AWS SDK v2 publisher, preserving the SDK credential model. Web Push encrypts the standard browser subscription payload and signs VAPID authentication. These optional modules do not add dependencies to the root module.

{{snippet:apns}}

{{snippet:sns}}

{{snippet:webpush}}

FCM HTTP v1 accepts a caller-supplied `TokenSource` function for refreshed OAuth access tokens. OneSignal and Expo submit native push requests. Acceptance means the service accepted the request or ticket; the library does not poll delivery receipts.

## WhatsApp Cloud

Ordinary `Send` uses free-form text within the recipient's customer-service window. Use `SendTemplate` with an already approved template outside that window. Configure the Graph API version explicitly. The library does not create templates or verify service-window eligibility.

{{snippet:whatsapp}}

## Application testing

Application services depend on `notification.Sender`. The fake validates and captures messages without contacting providers. Use `fake.Respond(err)` to test a failure.

{{snippet:fake}}

## Advanced dispatch

Applications that need recipient lookup, preferences, localized templates, per-recipient fallback, or deduplication can use `NewDispatcher`. These capabilities are optional and keep their explicit delivery results. Use `Fallback` for ordered configured destinations; use `NewDispatcher` when fallback must follow application recipient policy. See [advanced dispatch](docs/advanced.md) for the complete runnable example and migration from the earlier API.

## Development

```sh
make test
make test-race
make test-integration # Requires Docker.
make vet
make check-generated
make security
```

Documentation is generated by the Go tools in `docs/`, using the runnable examples as the source of truth. `make test` validates every module, executes the example programs, and enforces exact 100% library unit statement coverage. See [development and release](docs/development.md) for independent module tags, [delivery contracts](docs/contracts.md) for guarantees and limits, and the [API review](docs/api-review.md) for the simplicity and composition decisions.

See the [current release gates](docs/release-readiness.md) for completed checks and outstanding verification.

The [original framework proposal](docs/designs/framework-proposal.md) records the larger roadmap. Durable Queue integration, SQL persistence, inboxes, and GoForj generation remain future work.
