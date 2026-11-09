// Command fallback sends through ordered trusted webhook destinations.
package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/goforj/notification"
	"github.com/goforj/notification/driver/webhook"
)

// main demonstrates real HTTP fallback and explicit process-local deduplication.
func main() {
	// snippet:start
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
	// snippet:end
}
