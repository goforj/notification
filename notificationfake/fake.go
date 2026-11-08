// Package notificationfake captures validated intent without implying provider delivery.
package notificationfake

import (
	"context"
	"sync"

	"github.com/goforj/notification"
)

// Fake is a concurrency-safe application Sender with independent captured snapshots.
// It validates caller input but does not invent plans or provider acceptance.
type Fake struct {
	mu       sync.Mutex
	requests []notification.Request
	result   notification.Result
	err      error
}

// New constructs an isolated capture fake with a default empty successful result.
func New() *Fake { return &Fake{} }

// Respond configures explicit test outcomes copied independently from caller storage.
func (f *Fake) Respond(result notification.Result, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.result = result.Clone()
	f.err = err
}

// Send validates and records intent without rendering or transport execution.
func (f *Fake) Send(ctx context.Context, r notification.Request) (notification.Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return notification.Result{}, err
	}
	if err := r.Validate(); err != nil {
		return notification.Result{}, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests = append(f.requests, r.Clone())
	result := f.result.Clone()
	result.RequestID = r.ID
	return result, f.err
}

// Requests returns immutable copies for deliberate inspection of safe fixture data.
func (f *Fake) Requests() []notification.Request {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]notification.Request, len(f.requests))
	for i, r := range f.requests {
		out[i] = r.Clone()
	}
	return out
}
