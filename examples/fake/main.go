// Command fake captures intent without reporting execution or provider acceptance.
package main

import (
	"context"
	"fmt"

	"github.com/goforj/notification"
	"github.com/goforj/notification/notificationfake"
)

// main demonstrates isolated application tests through the Sender capability.
func main() {
	// snippet:start
	fake := notificationfake.New()
	var sender notification.Sender = fake
	result, err := sender.Send(context.Background(), notification.Request{
		ID: "assignment-42", Route: "ticket.updated", Recipients: []notification.RecipientRef{{Type: "user", ID: "user-42"}},
		Content: notification.Content{Key: "ticket.changed", Values: []notification.Value{notification.String("ticket", "T-42")}},
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("captured requests:", len(fake.Requests()))
	// captured requests: 1
	fmt.Println("provider outcomes:", len(result.Deliveries))
	// provider outcomes: 0
	// snippet:end
}
