// Command composition routes messages through ordinary configured senders.
package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/goforj/notification"
	"github.com/goforj/notification/notificationfake"
)

// main demonstrates opt-in routing without planners or background work.
func main() {
	// snippet:start
	chat := notificationfake.New()
	push := notificationfake.New()
	sender := notification.New(notification.Fanout(chat, push))
	message := notification.Message{
		Title: "Monitor down",
		Body:  "API health check failed.",
	}
	if err := sender.Send(context.Background(), message); err != nil {
		panic(err)
	}
	fmt.Println("captured targets:", len(chat.Messages())+len(push.Messages()))
	// captured targets: 2

	chat.Respond(notification.ErrPermanent)
	backup := notification.New(notification.Fallback(chat, push))
	if err := backup.Send(context.Background(), message); err != nil {
		panic(err)
	}
	fmt.Println("backup captures:", len(push.Messages()))
	// backup captures: 2

	push.Respond(notification.NewDeliveryError("push", notification.Outcome{
		Status:     notification.Failed,
		RetryAfter: 30 * time.Second,
	}, notification.ErrThrottled))
	err := notification.New(push).Send(context.Background(), message)
	var failure *notification.DeliveryError
	if errors.As(err, &failure) {
		fmt.Println("retry advice:", failure.RetryAfter())
		// retry advice: 30s
	}
	// snippet:end
}
