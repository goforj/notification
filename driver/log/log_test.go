package log

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/goforj/notification"
	"github.com/goforj/notification/notificationtest"
)

// TestDriver verifies capture evidence and portable cancellation semantics.
func TestDriver(t *testing.T) {
	notificationtest.Driver(t, func() notificationtest.Fixture {
		var output bytes.Buffer
		return notificationtest.Fixture{Driver: New(slog.New(slog.NewJSONHandler(&output, nil))), Delivery: notification.Delivery[notification.TextPayload]{Payload: notification.TextPayload{Body: "secret"}, Endpoint: notification.Endpoint{Address: notification.Address("secret")}}, Status: notification.Unknown}
	})
	var out bytes.Buffer
	d := New(slog.New(slog.NewJSONHandler(&out, nil)))
	result, err := d.Deliver(context.Background(), notification.Delivery[notification.TextPayload]{Payload: notification.TextPayload{Body: "secret"}, Key: "secret", Endpoint: notification.Endpoint{Address: notification.Address("secret")}})
	if result.Status != notification.Unknown || !errors.Is(err, notification.ErrOutcomeUnknown) || strings.Contains(out.String(), "secret") {
		t.Fatal("log evidence or redaction")
	}
}

// TestSender verifies simple capture semantics without exposing message content.
func TestSender(t *testing.T) {
	var out bytes.Buffer
	driver := New(slog.New(slog.NewJSONHandler(&out, nil)))
	notificationtest.Sender(t, func() notification.Sender { return driver })
	if strings.Contains(out.String(), "T-42") || strings.Contains(out.String(), "Ticket") {
		t.Fatal("captured content leaked")
	}
}
