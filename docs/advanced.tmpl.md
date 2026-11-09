# Advanced dispatch

Use `notification.NewDispatcher` when application policy needs scoped recipients, semantic content keys, channel-specific renderers, preferences, or route success rules. Ordinary messages use `notification.New(driver)` and need none of this configuration.

`Dispatcher.Send(ctx, Request)` returns a structured `Result` and an error. Each recipient must satisfy its route policy independently. Planners resolve trusted endpoints; typed adapters pair renderers with transports. Optional memory journaling retains identity and prevents repeated accepted channels during explicit partial retries.

## Fallback and identity

This program sends to real local HTTP endpoints. Permanent rejection advances from primary to backup. A duplicate request returns prior completion without invoking either destination again.

Run it from the repository root:

```sh
go run ./examples/fallback
```

{{snippet:fallback}}

## Optional capabilities

- Implement `Planner` or `PlannerFunc` for application-owned recipient lookup, preferences, locale, and quiet hours.
- Supply typed `Renderer[P]` and `Driver[P]` pairs through `NewAdapter` for semantic templates or rich channel payloads.
- Select `AllRequired`, `Any`, `FirstSuccess`, or `BestEffort` per route.
- Add `WithJournal(NewMemoryJournal())` for explicitly process-local identity and partial retry convergence.
- Add `WithObserver` for categorical observations.
- Use `notificationfake.NewDispatcher()` for validated request capture and explicitly scripted results. Ordinary message tests use `notificationfake.New()`.

Webhook `Send` uses `Config.URL`. Its advanced `Deliver` method uses the planner's endpoint instead, so a transport used only by a dispatcher may omit the configured URL. Similarly, a Mail adapter used only through planned delivery may omit constructor recipients. Required collaborators remain caller-owned.

See [delivery contracts](contracts.md) for cancellation, unknown outcomes, replay limits, and retention.

## Migration from the initial unreleased API

The earlier `New(scope, planner, routes, adapters, ...)` constructor is now `NewDispatcher` and returns `*Dispatcher`. Existing request-based send calls keep their `(Result, error)` behavior. The old `Sender` interface is now `RequestSender`, and constructor `Option` is now `DispatchOption`. `WithJournal` and `WithObserver` keep their behavior.

For the common case, replace plan construction with a configured driver and call `New(driver).Send(ctx, Message)`. This returns only an error. Destinations belong in webhook configuration or Mail constructor recipients; the message has `Title` and `Body` fields. Simple sends have no implicit idempotency or routing.

For tests, `notificationfake.New()` now captures messages through `Messages()` and scripts errors with `Respond(err)`. The earlier request fake is available through `notificationfake.NewDispatcher()`, retaining `Requests()` and `Respond(Result, error)`.

This is a source API change to the unreleased library. No persisted format, configuration key, or minimum Go version changed.
