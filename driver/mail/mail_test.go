package mail

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/goforj/mail"
	"github.com/goforj/mail/mailfake"
	"github.com/goforj/notification"
)

// mailerFunc scripts legacy error-only Mail behavior.
type mailerFunc func(context.Context, mail.Message) error

// Send invokes the script without leaking provider prose through the adapter.
func (f mailerFunc) Send(ctx context.Context, m mail.Message) error { return f(ctx, m) }

// TestMail validates real Mail composition and conservative outcome mapping.
func TestMail(t *testing.T) {
	fake := mailfake.New()
	mailer := mail.New(fake, mail.WithDefaultFrom("sender@example.com", "Sender"))
	d := New(mailer)
	delivery := notification.Delivery[Payload]{Payload: Payload{Subject: "Ticket changed", Text: "T-42"}, Endpoint: notification.Endpoint{Address: notification.Address("recipient@example.com")}}
	out, err := d.Deliver(context.Background(), delivery)
	if err != nil || out.Status != notification.Accepted || out.IdempotencyUsed {
		t.Fatal(out, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := d.Deliver(ctx, delivery); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	invalid := delivery
	invalid.Payload = Payload{}
	if _, err := d.Deliver(context.Background(), invalid); !errors.Is(err, notification.ErrRender) {
		t.Fatal(err)
	}
	for _, cause := range []error{context.Canceled, context.DeadlineExceeded, errors.New("secret provider body")} {
		driver := New(mailerFunc(func(context.Context, mail.Message) error { return cause }))
		out, err := driver.Deliver(context.Background(), delivery)
		if out.Status != notification.Unknown || err == nil || strings.Contains(err.Error(), "secret") {
			t.Fatal(out, err)
		}
		if cause == context.Canceled || cause == context.DeadlineExceeded {
			if !errors.Is(err, cause) {
				t.Fatal("cancellation lost")
			}
		}
	}
	for _, p := range []Payload{{Subject: strings.Repeat("x", 257), Text: "body"}, {Subject: "title", Text: strings.Repeat("x", 32769)}, {Subject: "bad\r\nheader", Text: "body"}, {Subject: "title"}} {
		if p.Validate() == nil {
			t.Fatal("invalid email accepted")
		}
	}
}
