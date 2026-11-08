package notificationtest

import (
	"context"
	"testing"

	"github.com/goforj/notification"
)

// report captures expected failures to prove the inventory catches faulty drivers.
type report struct{ count int }

// Helper satisfies the reporting capability without an actual testing failure.
func (r *report) Helper() {}

// Errorf records intentionally faulty driver evidence without printing content.
func (r *report) Errorf(string, ...any) { r.count++ }

// TestInventory proves both conforming evidence and adversarial failures are checked.
func TestInventory(t *testing.T) {
	for _, status := range []notification.Status{notification.Accepted, notification.Unknown} {
		r := &report{}
		Driver(r, func() Fixture {
			return Fixture{Driver: notification.DriverFunc[notification.TextPayload](func(ctx context.Context, d notification.Delivery[notification.TextPayload]) (notification.Outcome, error) {
				if err := ctx.Err(); err != nil {
					return notification.Outcome{Status: notification.Failed}, err
				}
				if err := d.Payload.Validate(); err != nil {
					return notification.Outcome{Status: notification.Failed}, err
				}
				if status == notification.Unknown {
					return notification.Outcome{Status: status}, notification.ErrOutcomeUnknown
				}
				return notification.Outcome{Status: status}, nil
			}), Delivery: notification.Delivery[notification.TextPayload]{Payload: notification.TextPayload{Body: "hello"}}, Status: status}
		})
		if r.count != 0 {
			t.Fatal("valid driver rejected")
		}
	}
	r := &report{}
	Driver(r, func() Fixture {
		return Fixture{Driver: notification.DriverFunc[notification.TextPayload](func(context.Context, notification.Delivery[notification.TextPayload]) (notification.Outcome, error) {
			return notification.Outcome{Status: notification.Failed}, notification.ErrPermanent
		}), Status: notification.Accepted}
	})
	if r.count != 13 {
		t.Fatalf("faults detected: %d", r.count)
	}
	r = &report{}
	Driver(r, func() Fixture {
		return Fixture{Driver: notification.DriverFunc[notification.TextPayload](func(context.Context, notification.Delivery[notification.TextPayload]) (notification.Outcome, error) {
			return notification.Outcome{Status: notification.Unknown}, nil
		}), Status: notification.Unknown}
	})
	if r.count != 4 {
		t.Fatalf("unknown faults detected: %d", r.count)
	}
}
