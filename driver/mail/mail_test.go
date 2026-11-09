package mail

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/goforj/mail"
	"github.com/goforj/mail/mailfake"
	"github.com/goforj/notification"
	"github.com/goforj/notification/notificationtest"
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

// TestSender verifies real Mail composition through the simple configured-recipient path.
func TestSender(t *testing.T) {
	fake := mailfake.New()
	mailer := mail.New(fake, mail.WithDefaultFrom("sender@example.com", "Sender"))
	driver := New(mailer, "recipient@example.com")
	notificationtest.Sender(t, func() notification.Sender { return driver })
	if err := driver.Send(nil, notification.Message{Title: "Ticket", Body: "T-42 changed"}); err != nil {
		t.Fatal("nil context should use background", err)
	}
	message := notification.Message{Title: "Ticket", Body: "T-42 changed"}
	for _, addresses := range [][]string{nil, {"invalid"}, make([]string, 101)} {
		driver := New(mailer, addresses...)
		if err := driver.Send(context.Background(), message); err == nil {
			t.Fatal("invalid recipients accepted")
		}
	}
	if err := driver.Send(context.Background(), notification.Message{Body: "no subject"}); !errors.Is(err, notification.ErrInvalidMessage) {
		t.Fatal(err)
	}
}

// TestSenderOwnership prevents a borrowed Mailer from mutating configured recipients.
func TestSenderOwnership(t *testing.T) {
	calls := 0
	addresses := []string{"recipient@example.com"}
	driver := New(mailerFunc(func(_ context.Context, message mail.Message) error {
		calls++
		if message.From != nil || len(message.To) != 1 || message.To[0].Email != "recipient@example.com" || message.Subject != "Ticket" || message.Text != "hello" {
			t.Fatal("Mail composition changed")
		}
		message.To[0].Email = "changed@example.com"
		return errors.New("private provider prose")
	}), addresses...)
	addresses[0] = "changed@example.com"
	for i := 0; i < 2; i++ {
		if err := driver.Send(context.Background(), notification.Message{Title: "Ticket", Body: "hello"}); !errors.Is(err, notification.ErrOutcomeUnknown) {
			t.Fatal(err)
		}
	}
	if calls != 2 {
		t.Fatal("missing Mail calls")
	}
}
