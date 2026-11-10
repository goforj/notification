# Changelog

## v0.1.0

Send chat, email, SMS, push, and on-call notifications through one small Go Sender interface.

## What's included

- 51 drivers, including Slack, Discord, Telegram, PagerDuty, ntfy, Gotify, APNs, FCM, SNS, and Web Push.
- Synchronous sends, bounded fanout, conservative fallback, and explicit incident lifecycles.
- Optional recipient planning, typed rendering, and process-local deduplication.
- Caller-owned HTTP transports, application testing fakes, and executable examples for every driver constructor.
- A standard-library-only root module. Mail reuses GoForj Mail; Mail, APNs, SNS, and Web Push are independent optional modules.

## Try it

Requires Go 1.24 or newer.

```sh
go get github.com/goforj/notification@v0.1.0
```

Install only the optional adapters you use:

```sh
go get github.com/goforj/notification/driver/mail@v0.1.0
go get github.com/goforj/notification/driver/apns@v0.1.0
go get github.com/goforj/notification/driver/sns@v0.1.0
go get github.com/goforj/notification/driver/webpush@v0.1.0
```

See the [README](https://github.com/goforj/notification#quick-start) for the quick start and API examples. Each optional module has its own `driver/<name>/v0.1.0` tag.

## Verification scope

Protocol fixtures cover all drivers. ntfy, Gotify, and Matrix also have real-server verification. Live hosted-provider and paired Signal verification remains pending where marked in the [driver guide](https://github.com/goforj/notification/blob/main/docs/drivers.md). Provider acceptance does not prove recipient delivery.

The [release record](https://github.com/goforj/notification/blob/main/docs/release-readiness.md) tracks module publication and detailed validation evidence.
