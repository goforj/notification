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

```go
recipient := notification.RecipientRef{Type: "system", ID: "on_call"}
planner := notification.NewMemoryPlanner(map[string]notification.Plan{
	"ticket.updated": {Deliveries: []notification.PlannedDelivery{{
		Recipient: recipient, Channel: "log", Driver: "local",
		Endpoint: notification.Endpoint{Identity: "local", Address: notification.Address("local")},
		Locale:   "en", TimeZone: "UTC",
	}}},
})
driver := log.New(slog.New(slog.NewTextHandler(io.Discard, nil)))
notifier, err := notification.New(notification.Scope{Application: "helpdesk"}, planner,
	[]notification.Route{{Name: "ticket.updated", Revision: "v1", Policy: notification.BestEffort}},
	[]notification.Adapter{notification.NewAdapter("log", "local", "v1", notification.Literal("Ticket updated", "Ticket T-42 has a new owner."), driver)},
)
if err != nil {
	panic(err)
}
result, err := notifier.Send(context.Background(), notification.Request{
	ID: "assignment-42", Route: "ticket.updated", Recipients: []notification.RecipientRef{recipient},
	Content: notification.Content{Key: "ticket.assignment.changed", Values: []notification.Value{notification.String("ticket", "T-42")}},
})
if err != nil {
	panic(err)
}
fmt.Println("route succeeded:", result.Succeeded)
// route succeeded: true
fmt.Println("provider evidence:", result.Deliveries[0].Outcome.Status)
// provider evidence: unknown
```

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

| API | Example |
| --- | --- |
| [New](#quick-start) | Quick Start |
| [Notifier.Send](#quick-start) | Quick Start |
| [NewMemoryPlanner](#quick-start) | Quick Start |
| [NewAdapter](#quick-start) | Quick Start |
| [WithJournal](#fallback-and-identity) | Fallback and identity |
| [NewMemoryJournal](#fallback-and-identity) | Fallback and identity |
| [webhook.New](#fallback-and-identity) | Fallback and identity |
| [notificationfake.New](#application-testing) | Application testing |
| [Fake.Requests](#application-testing) | Application testing |
| [notificationmail.New](#mail-adapter) | Mail adapter |

## Fallback and identity

A permanent rejection advances to the next configured destination. An explicit duplicate returns the prior journaled outcomes. `AllowPrivate` below is limited to the local test server.

```go
calls := 0
provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	calls++
	if r.URL.Path == "/primary" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}))
defer provider.Close()
driver, err := webhook.New(webhook.Config{AllowPrivate: true})
if err != nil {
	panic(err)
}
defer driver.Close()
recipient := notification.RecipientRef{Type: "system", ID: "on_call"}
plan := notification.Plan{}
for _, name := range []string{"primary", "backup"} {
	plan.Deliveries = append(plan.Deliveries, notification.PlannedDelivery{
		Recipient: recipient, Channel: "webhook", Driver: name, Locale: "en", TimeZone: "UTC",
		Endpoint: notification.Endpoint{Identity: name, Address: notification.Address(provider.URL + "/" + name)},
	})
}
notifier, err := notification.New(notification.Scope{Application: "helpdesk"}, notification.NewMemoryPlanner(map[string]notification.Plan{"ticket.updated": plan}),
	[]notification.Route{{Name: "ticket.updated", Revision: "v1", Policy: notification.FirstSuccess}},
	[]notification.Adapter{
		notification.NewAdapter("webhook", "primary", "v1", notification.Literal("Ticket", "T-42 changed"), driver),
		notification.NewAdapter("webhook", "backup", "v1", notification.Literal("Ticket", "T-42 changed"), driver),
	}, notification.WithJournal(notification.NewMemoryJournal()),
)
if err != nil {
	panic(err)
}
request := notification.Request{ID: "update-42", Route: "ticket.updated", Recipients: []notification.RecipientRef{recipient}, Content: notification.Content{Key: "ticket.changed"}}
result, err := notifier.Send(context.Background(), request)
if err != nil {
	panic(err)
}
fmt.Println("primary:", result.Deliveries[0].Outcome.Status)
// primary: failed
fmt.Println("backup:", result.Deliveries[1].Outcome.Status)
// backup: accepted
if _, err := notifier.Send(context.Background(), request); err != nil {
	panic(err)
}
fmt.Println("HTTP calls after duplicate:", calls)
// HTTP calls after duplicate: 2
```

## Mail adapter

The optional adapter uses Mail defaults and delegates transport. This program uses Mail's fake, so its accepted outcome tests adapter mapping without sending email.

```go
transport := mailfake.New()
mailer := mail.New(transport, mail.WithDefaultFrom("sender@example.com", "Helpdesk"))
driver := notificationmail.New(mailer)
renderer := notification.RendererFunc[notificationmail.Payload](func(context.Context, notification.RenderRequest) (notificationmail.Payload, error) {
	return notificationmail.Payload{Subject: "Ticket updated", Text: "Ticket T-42 changed."}, nil
})
recipient := notification.RecipientRef{Type: "user", ID: "user-42"}
planner := notification.NewMemoryPlanner(map[string]notification.Plan{"ticket.updated": {Deliveries: []notification.PlannedDelivery{{
	Recipient: recipient, Channel: "email", Driver: "mail", Locale: "en", TimeZone: "UTC",
	Endpoint: notification.Endpoint{Identity: "verified-email-42", Address: notification.Address("recipient@example.com")},
}}}})
notifier, err := notification.New(notification.Scope{Application: "helpdesk"}, planner,
	[]notification.Route{{Name: "ticket.updated", Revision: "v1", Policy: notification.AllRequired}},
	[]notification.Adapter{notification.NewAdapter("email", "mail", "v1", renderer, driver)},
)
if err != nil {
	panic(err)
}
result, err := notifier.Send(context.Background(), notification.Request{ID: "mail-42", Route: "ticket.updated", Recipients: []notification.RecipientRef{recipient}, Content: notification.Content{Key: "ticket.changed"}})
if err != nil {
	panic(err)
}
fmt.Println("adapter evidence:", result.Deliveries[0].Outcome.Status)
// adapter evidence: accepted
```

## Application testing

Depend on `notification.Sender` in application code. The capture fake validates requests and retains immutable snapshots without inventing delivery results.

```go
fake := notificationfake.New()
var sender notification.Sender = fake
result, err := sender.Send(context.Background(), notification.Request{
	ID: "assignment-42", Route: "ticket.updated", Recipients: []notification.RecipientRef{{Type: "user", ID: "user-42"}},
	Content: notification.Content{Key: "ticket.changed", Values: []notification.Value{notification.String("ticket", "T-42")}},
})
if err != nil {
	panic(err)
}
fmt.Println("captured requests:", len(fake.Requests()))
// captured requests: 1
fmt.Println("provider outcomes:", len(result.Deliveries))
// provider outcomes: 0
```

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
