// Command mail composes a real GoForj Mail driver and a chat notification using local fixtures.
package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"

	"github.com/goforj/mail"
	"github.com/goforj/mail/mailresend"
	"github.com/goforj/notification"
	notificationmail "github.com/goforj/notification/driver/mail"
	"github.com/goforj/notification/driver/slack"
)

// main demonstrates one application-owned Mailer reused for notification composition.
func main() {
	var emails, chats atomic.Int64
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/email" {
			emails.Add(1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"id":"email-42"}`)
			return
		}
		chats.Add(1)
		_, _ = io.WriteString(w, "ok")
	}))
	defer provider.Close()
	// snippet:start
	transport, err := mailresend.New(mailresend.Config{
		APIKey:     "local-fixture-key",
		Endpoint:   provider.URL + "/email",
		HTTPClient: provider.Client(),
	})
	if err != nil {
		panic(err)
	}
	mailer := mail.New(transport, mail.WithDefaultFrom("sender@example.com", "Helpdesk"))
	email := notificationmail.New(mailer, "recipient@example.com")
	chat, err := slack.New(slack.Config{
		URL: provider.URL + "/slack",
		HTTP: notification.HTTPConfig{
			AllowPrivate: true,
		},
	})
	if err != nil {
		panic(err)
	}
	defer chat.Close()
	sender := notification.Fanout(email, chat)
	err = sender.Send(context.Background(), notification.Message{
		Title: "Ticket updated",
		Body:  "Ticket T-42 has a new owner.",
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("accepted email and chat:", emails.Load(), chats.Load())
	// accepted email and chat: 1 1
	// snippet:end
}
