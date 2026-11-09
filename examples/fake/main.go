// Command fake captures messages without contacting a provider.
package main

import (
	"context"
	"fmt"

	"github.com/goforj/notification"
	"github.com/goforj/notification/notificationfake"
)

// main demonstrates testing application code through the Sender capability.
func main() {
	// snippet:start
	fake := notificationfake.New()
	var sender notification.Sender = fake

	err := sender.Send(context.Background(), notification.Message{
		Title: "Ticket updated",
		Body:  "Ticket T-42 has a new owner.",
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("captured messages:", len(fake.Messages()))
	// captured messages: 1
	// snippet:end
}
