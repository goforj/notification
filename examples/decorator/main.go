// Command decorator instruments the Sender interface without changing provider configuration.
package main

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/goforj/notification"
	"github.com/goforj/notification/notificationfake"
)

// main keeps application metrics outside transport and notification policy.
func main() {
	// snippet:start
	driver := notificationfake.New()
	var attempts atomic.Int64

	instrumented := notification.SenderFunc(func(ctx context.Context, message notification.Message) error {
		attempts.Add(1)
		return driver.Send(ctx, message)
	})
	notifier := notification.New(instrumented)

	err := notifier.Send(context.Background(), notification.Message{
		Title: "Ticket updated",
		Body:  "Ticket T-42 has a new owner.",
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("attempted sends:", attempts.Load())
	// attempted sends: 1

	err = notifier.Send(context.Background(), notification.Message{})
	fmt.Println("invalid message:", errors.Is(err, notification.ErrInvalidMessage))
	// invalid message: true
	fmt.Println("attempted sends:", attempts.Load())
	// attempted sends: 1
	// snippet:end
}
