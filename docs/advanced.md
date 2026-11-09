# Advanced dispatch

Use `notification.NewDispatcher` when application policy needs scoped recipients, semantic content keys, channel-specific renderers, preferences, or route success rules. Ordinary messages use `notification.New(driver)` and need none of this configuration.

`Dispatcher.Send(ctx, Request)` returns a structured `Result` and an error. Each recipient must satisfy its route policy independently. Planners resolve trusted endpoints; typed adapters pair renderers with transports. Optional memory journaling retains identity and prevents repeated accepted channels during explicit partial retries.

## Fallback and identity

This program sends to real local HTTP endpoints. Permanent rejection advances from primary to backup. A duplicate request returns prior completion without invoking either destination again.

Run it from the repository root:

```sh
go run ./examples/fallback
```

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

driver, err := webhook.New(webhook.Config{
	AllowPrivate: true,
})
if err != nil {
	panic(err)
}
defer driver.Close()

recipient := notification.RecipientRef{
	Type: "system",
	ID:   "on_call",
}
plan := notification.Plan{}
for _, name := range []string{"primary", "backup"} {
	plan.Deliveries = append(plan.Deliveries, notification.PlannedDelivery{
		Recipient: recipient,
		Channel:   "webhook",
		Driver:    name,
		Endpoint: notification.Endpoint{
			Identity: name,
			Address:  notification.Address(provider.URL + "/" + name),
		},
		Locale:   "en",
		TimeZone: "UTC",
	})
}

scope := notification.Scope{
	Application: "helpdesk",
}
planner := notification.NewMemoryPlanner(map[string]notification.Plan{
	"ticket.updated": plan,
})
routes := []notification.Route{
	{
		Name:     "ticket.updated",
		Revision: "v1",
		Policy:   notification.FirstSuccess,
	},
}
renderer := notification.Literal("Ticket", "T-42 changed")
adapters := []notification.Adapter{
	notification.NewAdapter("webhook", "primary", "v1", renderer, driver),
	notification.NewAdapter("webhook", "backup", "v1", renderer, driver),
}

notifier, err := notification.NewDispatcher(
	scope,
	planner,
	routes,
	adapters,
	notification.WithJournal(notification.NewMemoryJournal()),
)
if err != nil {
	panic(err)
}

request := notification.Request{
	ID:         "update-42",
	Route:      "ticket.updated",
	Recipients: []notification.RecipientRef{recipient},
	Content: notification.Content{
		Key: "ticket.changed",
	},
}
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
