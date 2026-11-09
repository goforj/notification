package notificationtest

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/goforj/notification"
)

// Sender runs the common message contract against a real configured driver.
// Its factory owns resources and registers cleanup with the enclosing test.
func Sender(t TestingT, factory func() notification.Sender) {
	t.Helper()
	driver := factory()
	message := notification.Message{
		Title: "Ticket",
		Body:  "T-42 changed",
	}
	if err := driver.Send(context.Background(), message); err != nil {
		t.Errorf("send: configured driver failed")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := driver.Send(ctx, message); !errors.Is(err, context.Canceled) {
		t.Errorf("cancellation: identity was lost")
	}
	ctx, cancel = context.WithDeadline(context.Background(), time.Unix(0, 0))
	defer cancel()
	if err := driver.Send(ctx, message); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("deadline: identity was lost")
	}
	if err := driver.Send(context.Background(), notification.Message{}); !errors.Is(err, notification.ErrInvalidMessage) {
		t.Errorf("validation: invalid message was accepted")
	}
	var wg sync.WaitGroup
	failures := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			failures <- driver.Send(context.Background(), message)
		}()
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		if err != nil {
			t.Errorf("concurrency: configured driver failed")
		}
	}
}
