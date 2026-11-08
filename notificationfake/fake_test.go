package notificationfake

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/goforj/notification"
)

// TestFake verifies validation, isolation and explicit outcome scripting.
func TestFake(t *testing.T) {
	f := New()
	r := notification.Request{ID: "one", Route: "ticket.updated", Recipients: []notification.RecipientRef{{Type: "user", ID: "one"}}, Content: notification.Content{Key: "ticket.changed", Values: []notification.Value{notification.String("ticket", "T-42")}}}
	out, err := f.Send(nil, r)
	if err != nil || out.RequestID != "one" || len(out.Deliveries) != 0 || out.Succeeded {
		t.Fatal("capture claimed execution")
	}
	r.Recipients[0].ID = "changed"
	snapshot := f.Requests()
	if snapshot[0].Recipients[0].ID != "one" {
		t.Fatal("input alias")
	}
	snapshot[0].Recipients[0].ID = "other"
	if f.Requests()[0].Recipients[0].ID != "one" {
		t.Fatal("output alias")
	}
	scripted := notification.Result{Deliveries: []notification.DeliveryResult{{Channel: "email"}}}
	f.Respond(scripted, notification.ErrDeferred)
	scripted.Deliveries[0].Channel = "changed"
	out, err = f.Send(context.Background(), r)
	if !errors.Is(err, notification.ErrDeferred) || out.Deliveries[0].Channel != "email" {
		t.Fatal("response alias")
	}
	out.Deliveries[0].Channel = "changed"
	out, _ = f.Send(nil, r)
	if out.Deliveries[0].Channel != "email" {
		t.Fatal("return alias")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := f.Send(ctx, r); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	r.ID = ""
	if _, err := f.Send(nil, r); !errors.Is(err, notification.ErrInvalidRequest) {
		t.Fatal(err)
	}
}

// TestConcurrentCapture proves independent concurrent test use.
func TestConcurrentCapture(t *testing.T) {
	f := New()
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = f.Send(nil, notification.Request{ID: "one", Route: "ticket.updated", Recipients: []notification.RecipientRef{{Type: "user", ID: "one"}}, Content: notification.Content{Key: "ticket.changed"}})
			_ = f.Requests()
			f.Respond(notification.Result{}, nil)
		}()
	}
	wg.Wait()
	if len(f.Requests()) != 16 {
		t.Fatal("missing captures")
	}
}
