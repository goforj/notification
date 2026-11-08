// Package notificationtest provides reusable typed driver contract checks.
package notificationtest

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/goforj/notification"
)

// TestingT is the minimal reporting capability used by the conformance inventory.
type TestingT interface {
	Helper()
	Errorf(string, ...any)
}

// Fixture contains a fresh driver, a valid delivery, and expected success evidence.
// Factories register resource cleanup with their enclosing test.
type Fixture struct {
	Driver   notification.Driver[notification.TextPayload]
	Delivery notification.Delivery[notification.TextPayload]
	Status   notification.Status
}

// Driver runs acceptance, cancellation, deadlines, validation, and concurrent sends.
// The caller supplies a real local transport fixture, not a scripted success stub.
func Driver(t TestingT, factory func() Fixture) {
	t.Helper()
	f := factory()
	out, err := f.Driver.Deliver(context.Background(), f.Delivery)
	if out.Status != f.Status {
		t.Errorf("acceptance: unexpected status %s", out.Status)
	}
	if f.Status == notification.Accepted && err != nil {
		t.Errorf("acceptance: accepted delivery returned an error")
	}
	if f.Status == notification.Unknown && !errors.Is(err, notification.ErrOutcomeUnknown) {
		t.Errorf("acceptance: unknown delivery lost its classification")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = f.Driver.Deliver(ctx, f.Delivery)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("cancellation: identity was lost")
	}
	ctx, cancel = context.WithDeadline(context.Background(), time.Unix(0, 0))
	defer cancel()
	_, err = f.Driver.Deliver(ctx, f.Delivery)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("deadline: identity was lost")
	}
	invalid := f.Delivery
	invalid.Payload = notification.TextPayload{}
	_, err = f.Driver.Deliver(context.Background(), invalid)
	if !errors.Is(err, notification.ErrRender) {
		t.Errorf("validation: invalid payload was accepted")
	}
	var wg sync.WaitGroup
	outcomes := make(chan notification.Status, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			out, _ := f.Driver.Deliver(context.Background(), f.Delivery)
			outcomes <- out.Status
		}()
	}
	wg.Wait()
	close(outcomes)
	for status := range outcomes {
		if status != f.Status {
			t.Errorf("concurrency: unexpected status %s", status)
		}
	}
}
