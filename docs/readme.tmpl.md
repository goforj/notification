<p align="center">
  <img src="docs/assets/logo.png" width="300" alt="notification logo">
</p>

<p align="center">Send notifications in Go with a title, a body, and a configured driver.</p>

<p align="center">
  <img src="docs/assets/badges/status.svg" alt="Unreleased">
  <img src="docs/assets/badges/go.svg" alt="Go 1.24 or newer">
  <a href="LICENSE"><img src="docs/assets/badges/license.svg" alt="MIT license"></a>
  <a href="docs/unit-results.json"><img src="docs/assets/badges/coverage.svg" alt="Unit statement coverage"></a>
  <a href="docs/unit-results.json"><img src="docs/assets/badges/tests.svg" alt="Executed unit tests"></a>
</p>

<p align="center"><a href="#features">Features</a> · <a href="#quick-start">Quick Start</a> · <a href="#drivers">Drivers</a> · <a href="#runnable-examples">Examples</a> · <a href="#api-index">API</a></p>

## Features

| Feature | What you get |
| --- | --- |
| <img src="docs/assets/badges/intent.svg" width="22" height="22" alt="" align="middle"> Simple messages | Send a title and body through one configured driver. |
| <img src="docs/assets/badges/policy.svg" width="22" height="22" alt="" align="middle"> Recipient policy | Add scoped recipient resolution and fallback when an application needs them. |
| <img src="docs/assets/badges/channels.svg" width="22" height="22" alt="" align="middle"> Typed channels | Optional semantic templates pair typed renderers and transports. |
| <img src="docs/assets/badges/identity.svg" width="22" height="22" alt="" align="middle"> Explicit identity | Optional process-local deduplication preserves accepted channels during partial retries. |
| <img src="docs/assets/badges/diagnostics.svg" width="22" height="22" alt="" align="middle"> Safe diagnostics | Errors and built-in logs omit endpoints and message content. |
| <img src="docs/assets/badges/testing.svg" width="22" height="22" alt="" align="middle"> Isolated testing | Capture messages through the same small Sender interface used by applications. |

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

Hosted verification is pending unless explicitly stated above. Local protocol fixtures exercise native requests, rejection codes and acknowledgements. Real-server tests run immutable ntfy and Gotify images; see [integration evidence](docs/integration-results.json). HTTP or native acceptance never proves recipient delivery.

The [complete Uptime Gopher inventory](docs/providers/uptime-gopher.md) records all 88 IDs. All 20 distinct dedicated demo adapters have library implementations, with SMTP sharing Mail. The 67 compatibility aliases remain a protocol-review backlog, not native support claims. Opsgenie is legacy migration support. Teams uses Workflows; LINE uses the Messaging API.

The root module uses only the standard library. Mail is an optional module using goforj/mail v0.3.1. Drivers own their configured destinations; a `Message` contains only content. Errors preserve safe cancellation and provider classifications without returning provider prose. See [delivery contracts](docs/contracts.md) for bounds and ownership.

## Installation

Until tags are published, consumers can use the local checkout:

```sh
go mod edit -require=github.com/goforj/notification@v0.0.0
go mod edit -replace=github.com/goforj/notification=$HOME/code/notification
go mod tidy
```

HTTP providers and the log driver are included in the root module:

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

The optional Mail adapter needs an additional local module requirement:

```sh
go mod edit -require=github.com/goforj/notification/driver/mail@v0.0.0
go mod edit -replace=github.com/goforj/notification/driver/mail=$HOME/code/notification/driver/mail
go mod tidy
```

It borrows an ordinary `*mail.Mailer` and configured recipients. Mail continues to own sender defaults, email composition, and transport.

## Runnable Examples

| Program | Run from the repository root | Demonstrates |
| --- | --- | --- |
| [Quick start](examples/quickstart/main.go) | `go run ./examples/quickstart` | A title and body through the log driver |
| [Webhook](examples/webhook/main.go) | `go run ./examples/webhook` | A message sent to a configured local HTTP provider |
| [Mail](examples/mail/main.go) | `cd examples/mail && go run .` | Mail defaults and a configured recipient with a fake transport |
| [Fake](examples/fake/main.go) | `go run ./examples/fake` | Application testing through Sender |
| [Composition](examples/composition/main.go) | `go run ./examples/composition` | Fanout, conservative fallback and safe retry advice |
| [Incident](examples/incident/main.go) | `go run ./examples/incident` | Stable lifecycle keys with a local PagerDuty protocol fixture |
| [Advanced fallback](examples/fallback/main.go) | `go run ./examples/fallback` | Optional recipient planning, fallback, and identity |

The runner discovers and executes every program and verifies the expected output immediately after each print call. Child programs participate in coverage and race runs. Local tests verify webhook wire behavior and Mail composition; they make no hosted-provider delivery claim.

## API Index

{{api-index}}

## Webhook delivery

The URL is driver configuration. `AllowPrivate` below permits the local test server; the default destination policy rejects private and special addresses.

{{snippet:webhook}}

## Composition

Use ordinary senders for application-level fanout or conservative fallback. Fanout attempts every target with at most eight concurrent calls. Fallback advances only after permanent rejection, unauthorized access or unsupported rendering; uncertainty and transient failures stop it to avoid accidental duplicates. Both borrow one to 64 configured senders and never retry.

{{snippet:composition}}

## Incident lifecycle

PagerDuty, JSM and legacy Opsgenie additionally implement `IncidentSender`. Construct keys from stable scoped IDs and use the same key for trigger, acknowledge and resolve. Ordinary `Send` requires a fixed configured key on these drivers. JSM and Opsgenie return asynchronous acceptance, without polling for completion.

{{snippet:incident}}

## Mail delivery

Configure the recipient on the Mail adapter. This example uses Mail's fake, so it sends no real email.

{{snippet:mail}}

## Application testing

Application services depend on `notification.Sender`. The fake validates and captures messages without contacting providers. Use `fake.Respond(err)` to test a failure.

{{snippet:fake}}

## Advanced dispatch

Applications that need recipient lookup, preferences, localized templates, fallback, or deduplication can use `NewDispatcher`. These capabilities are optional and keep their explicit delivery results. See [advanced dispatch](docs/advanced.md) for the complete runnable example and migration from the earlier API.

## Development

```sh
make test
make test-race
make test-integration # Requires Docker.
make vet
make check-generated
make security
```

Python 3 is required for documentation generation. `make test` validates every module, executes the example programs, and enforces exact 100% library unit statement coverage. See [development and release](docs/development.md) for independent module tags and [delivery contracts](docs/contracts.md) for guarantees and limits.

The [original framework proposal](docs/designs/framework-proposal.md) records the larger roadmap. Durable Queue integration, SQL persistence, inboxes, and GoForj generation remain future work.
