<p align="center">
  <img src="docs/assets/logo.png" width="300" alt="notification logo">
</p>

<p align="center">Semantic notifications for Go, with explicit recipients, channel policy, and delivery outcomes.</p>

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
| ![Semantic intent](docs/assets/badges/intent.svg) | Bounded content keys and typed scalar values keep provider payloads out of application services. |
| ![Recipient policy](docs/assets/badges/policy.svg) | Fanout and fallback evaluate every recipient independently. |
| ![Typed channels](docs/assets/badges/channels.svg) | Renderer and driver payload types must match at compile time. |
| ![Explicit identity](docs/assets/badges/identity.svg) | Scoped process-local deduplication preserves accepted channels during explicit partial retries. |
| ![Safe diagnostics](docs/assets/badges/diagnostics.svg) | Categorical observations and errors omit endpoints and message content. |
| ![Isolated testing](docs/assets/badges/testing.svg) | Validated capture fakes and a shared driver conformance inventory. |

## Quick Start

Requires Go 1.24+ and Python 3 for documentation generation. This is an unreleased library. From the local checkout:

```sh
cd ~/code/notification
go run ./examples/quickstart
```

The configured log driver captures the attempt. `BestEffort` succeeds while preserving unknown provider evidence.

{{snippet:quickstart}}

## Drivers

| Driver | Transport evidence | Dependencies |
| ---: | --- | --- |
| Log | Local capture only, reported as unknown | Standard library |
| Generic webhook | HTTP 2xx acceptance, with bounded responses and no redirect following | Standard library |
| Slack-compatible webhook | HTTP 2xx plus an `ok` acknowledgement | Standard library |
| Mail | Legacy Mail success means acceptance; errors remain unknown | Optional module using goforj/mail v0.3.1 |

Local HTTP tests execute generic and Slack-compatible wire contracts. No hosted Slack account or email provider delivery is claimed. See [delivery contracts](docs/contracts.md) for retry, destination, and ownership rules.

## Installation

The root module includes log and webhook packages:

```go
import "github.com/goforj/notification"
import "github.com/goforj/notification/driver/log"
import "github.com/goforj/notification/driver/webhook"
```

Run each included driver example from this checkout:

```sh
go run ./examples/quickstart # Log capture.
go run ./examples/fallback  # Generic webhook against a local HTTP provider.
```

Slack-compatible transport uses the included webhook package:

```go
driver, err := webhook.New(webhook.Config{Format: webhook.Slack})
```

The optional Mail adapter has its own module. Until independent module tags are published, consumers must use local replacements:

```sh
go mod edit -require=github.com/goforj/notification@v0.0.0
go mod edit -replace=github.com/goforj/notification=$HOME/code/notification
go mod edit -require=github.com/goforj/notification/driver/mail@v0.0.0
go mod edit -replace=github.com/goforj/notification/driver/mail=$HOME/code/notification/driver/mail
go mod tidy
```

A Mail adapter can borrow an ordinary `*mail.Mailer` through `driver/mail.New`. Its typed `Payload` carries subject, text, and HTML. Mail sender defaults and transport remain Mail-owned. The adapter module intentionally retains its relative root replacement so repository tests exercise unpublished root changes.

## Runnable Examples

| Program | Run from the repository root | Demonstrates |
| --- | --- | --- |
| [Quick start](examples/quickstart/main.go) | `go run ./examples/quickstart` | Scoped semantic intent and log capture |
| [Fallback](examples/fallback/main.go) | `go run ./examples/fallback` | Real local HTTP fallback and duplicate convergence |
| [Mail adapter](examples/mail/main.go) | `cd examples/mail && go run .` | Real Mail composition with a fake transport |
| [Fake](examples/fake/main.go) | `go run ./examples/fake` | Validated intent capture without provider acceptance |

The example runner discovers these programs, executes them, and checks the expected output immediately below each print call. Coverage and race runs instrument the child programs too.

## API Index

{{api-index}}

## Fallback and identity

A permanent rejection advances to the next configured destination. An explicit duplicate returns the prior journaled outcomes. `AllowPrivate` below is limited to the local test server.

{{snippet:fallback}}

## Mail adapter

The optional adapter uses Mail defaults and delegates transport. This program uses Mail's fake, so its accepted outcome tests adapter mapping without sending email.

{{snippet:mail}}

## Application testing

Depend on `notification.Sender` in application code. The capture fake validates requests and retains immutable snapshots without inventing delivery results.

{{snippet:fake}}

## Development

```sh
make test
make test-race
make vet
make check-generated
make security
```

`make test` runs the root and optional Mail module, executes examples, and enforces exact 100% library unit statement coverage. Local HTTP provider tests run with the unit suite; no Docker or external provider credentials are required. See [development and release](docs/development.md) for module inventory and independent tag planning.

The [delivery contracts](docs/contracts.md) describe guarantees and limits. The [original framework proposal](docs/designs/framework-proposal.md) records the larger roadmap. Durable Queue integration, SQL persistence, inboxes, and GoForj generation are future work.
