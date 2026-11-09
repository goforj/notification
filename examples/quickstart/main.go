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
	// snippet:end
}
