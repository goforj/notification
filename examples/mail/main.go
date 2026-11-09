// Command mail sends through the optional Mail adapter with a fake transport.
package main

import (
	"context"
	"fmt"

	"github.com/goforj/mail"
	"github.com/goforj/mail/mailfake"
	"github.com/goforj/notification"
	notificationmail "github.com/goforj/notification/driver/mail"
)

// main demonstrates Mail defaults and a configured recipient without a plan.
func main() {
	// snippet:start
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
	// snippet:end
}
