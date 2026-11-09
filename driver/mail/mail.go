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
type Driver struct {
	mailer     Mailer
	recipients []mail.Recipient
}

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

// New configures trusted recipients for Send without owning the Mailer lifecycle.
// Recipients may be omitted when only planner-resolved Deliver is used.
func New(mailer Mailer, recipients ...string) *Driver {
	to := make([]mail.Recipient, len(recipients))
	for i, address := range recipients {
		to[i] = mail.Recipient{Email: address}
	}
	return &Driver{mailer: mailer, recipients: to}
}

// Deliver maps nil-error legacy Mail success to acceptance with no idempotency claim.
func (d *Driver) Deliver(ctx context.Context, delivery notification.Delivery[Payload]) (notification.Outcome, error) {
	if err := ctx.Err(); err != nil {
		return notification.Outcome{Status: notification.Failed}, err
	}
	if err := delivery.Payload.Validate(); err != nil {
		return notification.Outcome{Status: notification.Failed}, err
	}
	p := delivery.Payload
	return d.deliver(ctx, mail.Message{To: []mail.Recipient{{Email: delivery.Endpoint.Address.Reveal()}}, Subject: p.Subject, Text: p.Text, HTML: p.HTML})
}

// deliver classifies the legacy Mail boundary without leaking provider prose.
func (d *Driver) deliver(ctx context.Context, message mail.Message) (notification.Outcome, error) {
	err := d.mailer.Send(ctx, message)
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

// Send delivers plain text to the recipients configured by New. Mail retains
// sender-default and transport ownership; nil means legacy provider acceptance.
func (d *Driver) Send(ctx context.Context, message notification.Message) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := message.Validate(); err != nil {
		return err
	}
	if len(d.recipients) == 0 || len(d.recipients) > 100 {
		return notification.ErrInvalidRequest
	}
	envelope := mail.Message{
		From:    &mail.Recipient{Email: "sender@example.com"},
		To:      append([]mail.Recipient(nil), d.recipients...),
		Subject: message.Title,
		Text:    message.Body,
	}
	if err := envelope.Validate(); err != nil {
		return notification.ErrInvalidMessage
	}
	envelope.From = nil
	outcome, err := d.deliver(ctx, envelope)
	if err != nil {
		return notification.NewDeliveryError("mail", outcome, err)
	}
	return nil
}
