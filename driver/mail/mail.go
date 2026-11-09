// Package mail adapts goforj/mail without duplicating any email transport.
package mail

import (
	"context"
	"errors"

	"github.com/goforj/mail"
	"github.com/goforj/notification"
)

// Mailer is the borrowed goforj/mail send capability, including Mail defaults.
type Mailer interface {
	Send(context.Context, mail.Message) error
}

// Driver delegates email transport to a required wired Mailer.
type Driver struct{ mailer Mailer }

// Payload contains the email presentation before one resolved recipient is applied.
type Payload struct {
	Subject string
	Text    string
	HTML    string
}

// Validate bounds content; Mail owns envelope, header and email-address validation.
func (p Payload) Validate() error {
	if len(p.Subject) > 256 || len(p.Text)+len(p.HTML) > 32768 {
		return notification.ErrRender
	}
	message := mail.Message{From: &mail.Recipient{Email: "sender@example.com"}, To: []mail.Recipient{{Email: "recipient@example.com"}}, Subject: p.Subject, Text: p.Text, HTML: p.HTML}
	if message.Validate() != nil {
		return notification.ErrRender
	}
	return nil
}

// New constructs an adapter without taking ownership of the Mailer lifecycle.
func New(mailer Mailer) *Driver { return &Driver{mailer} }

// Deliver maps nil-error legacy Mail success to acceptance with no idempotency claim.
func (d *Driver) Deliver(ctx context.Context, delivery notification.Delivery[Payload]) (notification.Outcome, error) {
	if err := ctx.Err(); err != nil {
		return notification.Outcome{Status: notification.Failed}, err
	}
	if err := delivery.Payload.Validate(); err != nil {
		return notification.Outcome{Status: notification.Failed}, err
	}
	p := delivery.Payload
	err := d.mailer.Send(ctx, mail.Message{To: []mail.Recipient{{Email: delivery.Endpoint.Address.Reveal()}}, Subject: p.Subject, Text: p.Text, HTML: p.HTML})
	if err == nil {
		return notification.Outcome{Status: notification.Accepted}, nil
	}
	// Legacy provider errors cannot safely prove rejection or retryability.
	for _, safe := range []error{context.Canceled, context.DeadlineExceeded} {
		if errors.Is(err, safe) {
			return notification.Outcome{Status: notification.Unknown}, safe
		}
	}
	return notification.Outcome{Status: notification.Unknown}, notification.ErrOutcomeUnknown
}
