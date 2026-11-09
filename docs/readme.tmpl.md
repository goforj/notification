<p align="center">
  <img src="docs/assets/logo.png" width="300" alt="notification logo">
</p>

<p align="center">Send notifications in Go with a title, a body, and a configured driver.</p>

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

All drivers have local tests. ntfy and Gotify also pass real-server tests; hosted and paired-session verification remains pending for other providers. See [driver configuration and delivery contracts](docs/drivers.md) for details and [integration evidence](docs/integration-results.json) for exact server versions.

Mail supports SMTP through its configured Mailer. Teams uses Workflows, LINE uses the Messaging API, and Opsgenie provides legacy integration support.

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
| [Decorator](examples/decorator/main.go) | `go run ./examples/decorator` | Application metrics through SenderFunc |
| [Incident](examples/incident/main.go) | `go run ./examples/incident` | Stable lifecycle keys with a local PagerDuty protocol fixture |
| [Advanced fallback](examples/fallback/main.go) | `go run ./examples/fallback` | Optional recipient planning, fallback, and identity |

The runner discovers and executes every program and verifies the expected output immediately after each print call. Child programs participate in coverage and race runs. Local tests verify webhook wire behavior and Mail composition; they make no hosted-provider delivery claim.

## API Index

{{api-index}}

## Webhook delivery

The URL is driver configuration. `AllowPrivate` below permits the local test server; the default destination policy rejects private and special addresses.

{{snippet:webhook}}

## Composition

Use ordinary senders for application-level fanout or conservative fallback. Fanout attempts every target with at most eight concurrent calls. Fallback advances only after permanent rejection, unauthorized access or unsupported rendering; uncertainty and transient failures stop it to avoid accidental duplicates. Both return a `Sender` directly, borrow one to 64 configured senders, and never retry. A `New` wrapper is optional around a composition; use it when wrapping a custom handler that needs common validation and safe errors.

{{snippet:composition}}

## Custom senders

Use `SenderFunc` to adapt an existing service or decorate a configured sender with application metrics or tracing. Preserve the caller's context and returned error. `New` applies validation and safe errors around the custom handler, so invalid messages do not reach it.

{{snippet:decorator}}

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

The [original framework proposal](docs/designs/framework-proposal.md) records the larger roadmap. Durable Queue integration, SQL persistence, inboxes, and GoForj generation remain future work.
