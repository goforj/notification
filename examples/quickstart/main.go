// Command quickstart captures a configured system notification without a service.
package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/goforj/notification"
	"github.com/goforj/notification/driver/log"
)

// main demonstrates semantic intent and honest local capture evidence.
func main() {
	// snippet:start
	recipient := notification.RecipientRef{
		Type: "system",
		ID:   "on_call",
	}
	planner := notification.NewMemoryPlanner(map[string]notification.Plan{
		"ticket.updated": {
			Deliveries: []notification.PlannedDelivery{
				{
					Recipient: recipient,
					Channel:   "log",
					Driver:    "local",
					Endpoint: notification.Endpoint{
						Identity: "local",
						Address:  notification.Address("local"),
					},
					Locale:   "en",
					TimeZone: "UTC",
				},
			},
		},
	})

	scope := notification.Scope{
		Application: "helpdesk",
	}
	routes := []notification.Route{
		{
			Name:     "ticket.updated",
			Revision: "v1",
			Policy:   notification.BestEffort,
		},
	}
	driver := log.New(slog.New(slog.NewTextHandler(io.Discard, nil)))
	renderer := notification.Literal("Ticket updated", "Ticket T-42 has a new owner.")
	adapter := notification.NewAdapter("log", "local", "v1", renderer, driver)

	notifier, err := notification.New(scope, planner, routes, []notification.Adapter{adapter})
	if err != nil {
		panic(err)
	}

	request := notification.Request{
		ID:         "assignment-42",
		Route:      "ticket.updated",
		Recipients: []notification.RecipientRef{recipient},
		Content: notification.Content{
			Key: "ticket.assignment.changed",
			Values: []notification.Value{
				notification.String("ticket", "T-42"),
			},
		},
	}
	result, err := notifier.Send(context.Background(), request)
	if err != nil {
		panic(err)
	}
	fmt.Println("route succeeded:", result.Succeeded)
	// route succeeded: true
	fmt.Println("provider evidence:", result.Deliveries[0].Outcome.Status)
	// provider evidence: unknown
	// snippet:end
}
