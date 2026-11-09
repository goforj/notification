package notification

import (
	"context"
	"strings"
)

// Message is portable plain-text content for a driver's configured destination.
type Message struct {
	Title string
	Body  string
}

// Validate bounds plain text before any driver runs. Title is optional; drivers
// such as Mail can require it when their channel needs a subject.
func (m Message) Validate() error {
	if (TextPayload{Title: m.Title, Body: m.Body}).Validate() != nil || strings.ContainsAny(m.Title, "\r\n") {
		return ErrInvalidMessage
	}
	return nil
}

// Sender handles a message synchronously without retries or background work.
// A nil error means the configured driver handled the message, which may be
// local capture or provider acceptance. It never proves recipient delivery.
type Sender interface {
	Send(context.Context, Message) error
}

// SenderFunc adapts an application-owned message handler.
type SenderFunc func(context.Context, Message) error

// Send invokes the configured message handler.
func (f SenderFunc) Send(ctx context.Context, message Message) error {
	return f(ctx, message)
}

// Notifier validates messages and borrows one concurrency-safe driver.
// Driver configuration owns the destination and transport lifecycle.
type Notifier struct {
	driver Sender
}

// New constructs a notifier without I/O. The driver is a required wired
// dependency and remains caller-owned; construction starts no background tasks.
func New(driver Sender) *Notifier {
	return &Notifier{driver: driver}
}

// Send validates plain text and invokes the configured driver once. It strips
// arbitrary driver prose from errors while preserving safe classifications.
func (n *Notifier) Send(ctx context.Context, message Message) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := message.Validate(); err != nil {
		return err
	}
	if err := n.driver.Send(ctx, message); err != nil {
		return safeError(err, ErrOutcomeUnknown)
	}
	return nil
}
