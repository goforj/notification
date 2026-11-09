package webhook

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/goforj/notification"
)

// TestServerErrorDoesNotReplayAcceptedWork models a receiver committing a side effect before losing its success response.
func TestServerErrorDoesNotReplayAcceptedWork(t *testing.T) {
	for _, status := range []int{http.StatusInternalServerError, http.StatusBadGateway, http.StatusGatewayTimeout, http.StatusRequestTimeout} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			var recorded atomic.Int64
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				recorded.Add(1)
				w.Header().Set("Retry-After", "9")
				w.WriteHeader(status)
			}))
			defer server.Close()
			driver, err := New(Config{URL: server.URL, AllowPrivate: true})
			if err != nil {
				t.Fatal(err)
			}
			defer driver.Close()
			scope := notification.Scope{Application: "replay-check"}
			ref := notification.RecipientRef{Type: "user", ID: "one"}
			planned := notification.PlannedDelivery{Recipient: ref, Channel: "webhook", Driver: "primary", Endpoint: notification.Endpoint{Identity: "primary", Address: notification.Address(server.URL)}, Locale: "en", TimeZone: "UTC"}
			planner := notification.NewMemoryPlanner(map[string]notification.Plan{"alert": {Deliveries: []notification.PlannedDelivery{planned}}})
			dispatcher, err := notification.NewDispatcher(scope, planner, []notification.Route{{Name: "alert", Revision: "v1", Policy: notification.AllRequired}}, []notification.Adapter{notification.NewAdapter("webhook", "primary", "v1", notification.Literal("Alert", "Changed"), driver)}, notification.WithJournal(notification.NewMemoryJournal()))
			if err != nil {
				t.Fatal(err)
			}
			request := notification.Request{ID: "operation-42", Route: "alert", Recipients: []notification.RecipientRef{ref}, Content: notification.Content{Key: "alert.changed"}}

			unknown := true
			for i := 0; i < 2; i++ {
				result, err := dispatcher.Send(context.Background(), request)
				unknown = unknown && errors.Is(err, notification.ErrOutcomeUnknown) && errors.Is(err, notification.ErrRetryable) && len(result.Deliveries) == 1 && result.Deliveries[0].Outcome.Status == notification.Unknown && result.Deliveries[0].Outcome.RetryAfter == 9*time.Second
			}
			if recorded.Load() != 1 {
				t.Fatal("journal replayed possibly accepted work", recorded.Load())
			}
			if !unknown {
				t.Fatal("server error falsely proves rejection")
			}

			var fallback atomic.Int64
			sender := notification.Fallback(driver, notification.SenderFunc(func(context.Context, notification.Message) error { fallback.Add(1); return nil }))
			if err := sender.Send(nil, notification.Message{Body: "body"}); !errors.Is(err, notification.ErrOutcomeUnknown) || fallback.Load() != 0 {
				t.Fatal("server error enabled fallback", err)
			}
		})
	}
}
