// Command mail demonstrates the optional Mail adapter using real Mail composition.
package main

import (
	"context"
	"fmt"

	"github.com/goforj/mail"
	"github.com/goforj/mail/mailfake"
	"github.com/goforj/notification"
	notificationmail "github.com/goforj/notification/driver/mail"
)

// main demonstrates acceptance through a Mail fake, without claiming real email delivery.
func main() {
	// snippet:start
	transport := mailfake.New()
	mailer := mail.New(transport, mail.WithDefaultFrom("sender@example.com", "Helpdesk"))
	driver := notificationmail.New(mailer)
	renderer := notification.RendererFunc[notificationmail.Payload](
		func(context.Context, notification.RenderRequest) (notificationmail.Payload, error) {
			return notificationmail.Payload{
				Subject: "Ticket updated",
				Text:    "Ticket T-42 changed.",
			}, nil
		},
	)

	recipient := notification.RecipientRef{
		Type: "user",
		ID:   "user-42",
	}
	planner := notification.NewMemoryPlanner(map[string]notification.Plan{
		"ticket.updated": {
			Deliveries: []notification.PlannedDelivery{
				{
					Recipient: recipient,
					Channel:   "email",
					Driver:    "mail",
					Endpoint: notification.Endpoint{
						Identity: "verified-email-42",
						Address:  notification.Address("recipient@example.com"),
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
			Policy:   notification.AllRequired,
		},
	}
	adapter := notification.NewAdapter("email", "mail", "v1", renderer, driver)

	notifier, err := notification.New(scope, planner, routes, []notification.Adapter{adapter})
	if err != nil {
		panic(err)
	}

	request := notification.Request{
		ID:         "mail-42",
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
	fmt.Println("adapter evidence:", result.Deliveries[0].Outcome.Status)
	// adapter evidence: accepted
	// snippet:end
}
