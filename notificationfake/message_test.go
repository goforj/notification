package notificationfake

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/goforj/notification"
)

// TestMessageFake checks capture ownership, configured failures and validation parity.
func TestMessageFake(t *testing.T) {
	fake := New()
	message := notification.Message{Title: "Ticket", Body: "T-42"}
	if err := fake.Send(nil, message); err != nil {
		t.Fatal(err)
	}
	message.Body = "changed"
	messages := fake.Messages()
	if messages[0].Body != "T-42" {
		t.Fatal("input changed capture")
	}
	messages[0].Body = "changed"
	if fake.Messages()[0].Body != "T-42" {
		t.Fatal("capture alias")
	}
	fake.Respond(notification.ErrThrottled)
	if err := fake.Send(context.Background(), message); !errors.Is(err, notification.ErrThrottled) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := fake.Send(ctx, message); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if err := fake.Send(nil, notification.Message{}); !errors.Is(err, notification.ErrInvalidMessage) {
		t.Fatal(err)
	}
	if len(fake.Messages()) != 2 {
		t.Fatal("invalid work captured")
	}
}

// TestConcurrentMessageCapture checks reads, response changes and sends together.
func TestConcurrentMessageCapture(t *testing.T) {
	fake := New()
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = fake.Send(nil, notification.Message{Body: "hello"})
			_ = fake.Messages()
			fake.Respond(nil)
		}()
	}
	wg.Wait()
	if len(fake.Messages()) != 16 {
		t.Fatal("missing captures")
	}
}
