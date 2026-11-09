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
| ![Simple messages](docs/assets/badges/intent.svg) | Send a title and body through one configured driver. |
| ![Recipient policy](docs/assets/badges/policy.svg) | Add scoped recipient resolution and fallback when an application needs them. |
| ![Typed channels](docs/assets/badges/channels.svg) | Optional semantic templates pair typed renderers and transports. |
| ![Explicit identity](docs/assets/badges/identity.svg) | Optional process-local deduplication preserves accepted channels during partial retries. |
| ![Safe diagnostics](docs/assets/badges/diagnostics.svg) | Errors and built-in logs omit endpoints and message content. |
| ![Isolated testing](docs/assets/badges/testing.svg) | Capture messages through the same small Sender interface used by applications. |

## Quick Start

Requires Go 1.24+. This is an unreleased library. From the local checkout:

```sh
cd ~/code/notification
go run ./examples/quickstart
```

Configure a driver once, then send a message:

```go
driver := log.New(slog.New(slog.NewTextHandler(io.Discard, nil)))
notifier := notification.New(driver)

err := notifier.Send(context.Background(), notification.Message{
	Title: "Ticket updated",
	Body:  "Ticket T-42 has a new owner.",
})
if err != nil {
	panic(err)
}
fmt.Println("message captured")
// message captured
```

The log driver captures locally. A nil error means the driver handled the message; it never proves recipient delivery. Send performs no retries or background work.

## Drivers

| Driver | Configuration | Successful handling | Verification |
| ---: | --- | --- | --- |
| [![DingTalk](docs/assets/badges/driver-dingding.svg)](driver/dingtalk) | Webhook and optional signing secret | Native errcode; HMAC signature at send time | Local protocol tests; hosted verification pending |
| [![Discord](docs/assets/badges/driver-discord.svg)](driver/discord) | Webhook URL and optional thread | Native message acknowledgement; mentions disabled | Local protocol tests; hosted verification pending |
| [![Mail](docs/assets/badges/driver-email.svg)](driver/mail) | Borrowed Mailer and recipients | Mail acceptance | Mail composition tests; hosted verification pending |
| [![Feishu](docs/assets/badges/driver-feishu.svg)](driver/feishu) | V2 bot webhook and optional signing secret | Native code; timestamped HMAC when configured | Local protocol tests; hosted verification pending |
| [![Google Chat](docs/assets/badges/driver-googlechat.svg)](driver/googlechat) | Space webhook URL | Native message resource acknowledgement | Local protocol tests; hosted verification pending |
| [![Gotify](docs/assets/badges/driver-gotify.svg)](driver/gotify) | Server, application token and integer priority | Native message identity and content acknowledgement | Real Gotify 3.1.1 server verified |
| [![Jira Service Management](docs/assets/badges/driver-jiraservicemanagement.svg)](driver/jsm) | Cloud ID, email and API token | Asynchronous request acceptance; stable alias lookup for actions | Local protocol tests; hosted verification pending |
| [![LINE](docs/assets/badges/driver-line.svg)](driver/line) | Channel access token and recipient ID | Messaging API push acknowledgement; LINE Notify excluded | Local protocol tests; hosted verification pending |
| [![Log](docs/assets/badges/driver-log.svg)](driver/log) | Borrowed slog logger | Local capture | Local capture tests |
| [![ntfy](docs/assets/badges/driver-ntfy.svg)](driver/ntfy) | Topic, optional authentication and integer priority | Native message identity and topic acknowledgement | Real ntfy 2.37.2 server verified |
| [![Opsgenie](docs/assets/badges/driver-opsgenie.svg)](driver/opsgenie) | Legacy integration key and incident alias | Asynchronous request acceptance; legacy migration support | Local protocol tests; hosted verification pending |
| [![PagerDuty](docs/assets/badges/driver-pagerduty.svg)](driver/pagerduty) | Routing key and explicit incident key | Events v2 acknowledgement; trigger, acknowledge, resolve | Local protocol tests; hosted verification pending |
| [![Slack](docs/assets/badges/driver-slack.svg)](driver/slack) | Incoming webhook URL | HTTP 2xx and ok acknowledgement | Local protocol tests; hosted verification pending |
| [![Teams](docs/assets/badges/driver-teams.svg)](driver/teams) | Workflows webhook URL | Adaptive Card submitted; HTTP 2xx acceptance | Local protocol tests; hosted verification pending |
| [![Telegram](docs/assets/badges/driver-telegram.svg)](driver/telegram) | Bot token, chat and optional topic | Native message identity; JSON retry advice retained | Local protocol tests; hosted verification pending |
| [![Twilio](docs/assets/badges/driver-twilio.svg)](driver/twilio) | Account, token, E.164 recipient and sender or service | Native Message SID and accepted status | Local protocol tests; hosted verification pending |
| [![WAHA](docs/assets/badges/driver-waha.svg)](driver/waha) | Server, API key, existing session and chat | HTTP acceptance across engine response schemas | Local protocol tests; paired-session verification pending |
| [![Webhook](docs/assets/badges/driver-webhook.svg)](driver/webhook) | Trusted endpoint | HTTP 2xx acceptance | Local protocol tests; hosted verification pending |
| [![WeCom](docs/assets/badges/driver-wecom.svg)](driver/wecom) | Group robot webhook URL | Native errcode acknowledgement | Local protocol tests; hosted verification pending |
| [![Whapi](docs/assets/badges/driver-whapi.svg)](driver/whapi) | Channel token and recipient | Native sent acknowledgement | Local protocol tests; hosted verification pending |

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

| API | Example |
| --- | --- |
| [New](#quick-start) | Quick Start |
| [Fanout](#composition) | Composition |
| [Fallback](#composition) | Composition |
| [DeliveryError.RetryAfter](#composition) | Composition |
| [pagerduty.Driver.Trigger](#incident-lifecycle) | Incident lifecycle |
| [pagerduty.Driver.Acknowledge](#incident-lifecycle) | Incident lifecycle |
| [pagerduty.Driver.Resolve](#incident-lifecycle) | Incident lifecycle |
| [Notifier.Send](#quick-start) | Quick Start |
| [webhook.New](#webhook-delivery) | Webhook delivery |
| [notificationfake.New](#application-testing) | Application testing |
| [Fake.Messages](#application-testing) | Application testing |
| [notificationmail.New](#mail-delivery) | Mail delivery |

## Webhook delivery

The URL is driver configuration. `AllowPrivate` below permits the local test server; the default destination policy rejects private and special addresses.

```go
provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusAccepted)
}))
defer provider.Close()

driver, err := webhook.New(webhook.Config{
	URL:          provider.URL,
	AllowPrivate: true,
})
if err != nil {
	panic(err)
}
defer driver.Close()

notifier := notification.New(driver)
err = notifier.Send(context.Background(), notification.Message{
	Title: "Ticket updated",
	Body:  "Ticket T-42 has a new owner.",
})
if err != nil {
	panic(err)
}
fmt.Println("message accepted")
// message accepted
```

## Composition

Use ordinary senders for application-level fanout or conservative fallback. Fanout attempts every target with at most eight concurrent calls. Fallback advances only after permanent rejection, unauthorized access or unsupported rendering; uncertainty and transient failures stop it to avoid accidental duplicates. Both borrow one to 64 configured senders and never retry.

```go
chat := notificationfake.New()
push := notificationfake.New()
sender := notification.New(notification.Fanout(chat, push))
message := notification.Message{
	Title: "Monitor down",
	Body:  "API health check failed.",
}
if err := sender.Send(context.Background(), message); err != nil {
	panic(err)
}
fmt.Println("captured targets:", len(chat.Messages())+len(push.Messages()))
// captured targets: 2

chat.Respond(notification.ErrPermanent)
backup := notification.New(notification.Fallback(chat, push))
if err := backup.Send(context.Background(), message); err != nil {
	panic(err)
}
fmt.Println("backup captures:", len(push.Messages()))
// backup captures: 2

push.Respond(notification.NewDeliveryError("push", notification.Outcome{
	Status:     notification.Failed,
	RetryAfter: 30 * time.Second,
}, notification.ErrThrottled))
err := notification.New(push).Send(context.Background(), message)
var failure *notification.DeliveryError
if errors.As(err, &failure) {
	fmt.Println("retry advice:", failure.RetryAfter())
	// retry advice: 30s
}
```

## Incident lifecycle

PagerDuty, JSM and legacy Opsgenie additionally implement `IncidentSender`. Construct keys from stable scoped IDs and use the same key for trigger, acknowledge and resolve. Ordinary `Send` requires a fixed configured key on these drivers. JSM and Opsgenie return asynchronous acceptance, without polling for completion.

```go
ctx := context.Background()
key := notification.IncidentKey("tenant:42:monitor:17")
err = driver.Trigger(ctx, notification.Incident{
	Key:      key,
	Title:    "API unavailable",
	Body:     "Health check failed.",
	Source:   "uptime-gopher",
	Severity: notification.SeverityCritical,
})
if err != nil {
	panic(err)
}
fmt.Println("trigger accepted by local fixture")
// trigger accepted by local fixture

if err := driver.Acknowledge(ctx, key); err != nil {
	panic(err)
}
if err := driver.Resolve(ctx, key); err != nil {
	panic(err)
}
fmt.Println("acknowledge and resolve use the same stable key")
// acknowledge and resolve use the same stable key
```

## Mail delivery

Configure the recipient on the Mail adapter. This example uses Mail's fake, so it sends no real email.

```go
transport := mailfake.New()
mailer := mail.New(transport, mail.WithDefaultFrom("sender@example.com", "Helpdesk"))
driver := notificationmail.New(mailer, "recipient@example.com")
notifier := notification.New(driver)

err := notifier.Send(context.Background(), notification.Message{
	Title: "Ticket updated",
	Body:  "Ticket T-42 has a new owner.",
})
if err != nil {
	panic(err)
}
fmt.Println("captured mail:", transport.SentCount())
// captured mail: 1
```

## Application testing

Application services depend on `notification.Sender`. The fake validates and captures messages without contacting providers. Use `fake.Respond(err)` to test a failure.

```go
fake := notificationfake.New()
var sender notification.Sender = fake

err := sender.Send(context.Background(), notification.Message{
	Title: "Ticket updated",
	Body:  "Ticket T-42 has a new owner.",
})
if err != nil {
	panic(err)
}
fmt.Println("captured messages:", len(fake.Messages()))
// captured messages: 1
```

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
