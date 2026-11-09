// Command quickstart captures a notification without an external service.
package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/goforj/notification"
	"github.com/goforj/notification/driver/log"
)

// main demonstrates the common message API with a local capture driver.
func main() {
	// snippet:start
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
	// snippet:end
}
