// Package notificationfake captures messages without contacting providers.
package notificationfake

import (
	"context"
	"sync"

	"github.com/goforj/notification"
)

// Fake is an isolated, concurrency-safe Sender that validates and captures messages.
// A successful capture makes no claim about provider acceptance or delivery.
type Fake struct {
	mu       sync.Mutex
	messages []notification.Message
	err      error
}

// New constructs an empty capture fake.
func New() *Fake {
	return &Fake{}
}

// Respond configures the error returned by subsequent valid sends.
func (f *Fake) Respond(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.err = err
}

// Send validates and captures a value copy without invoking a transport.
func (f *Fake) Send(ctx context.Context, message notification.Message) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := message.Validate(); err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.messages = append(f.messages, message)
	return f.err
}

// Messages returns independent captured values for deliberate fixture inspection.
func (f *Fake) Messages() []notification.Message {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]notification.Message(nil), f.messages...)
}
