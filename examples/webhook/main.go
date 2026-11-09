// Command webhook sends a message to a configured local HTTP provider.
package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/goforj/notification"
	"github.com/goforj/notification/driver/webhook"
)

// main demonstrates a configured destination without plans or renderers.
func main() {
	// snippet:start
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
	// snippet:end
}
