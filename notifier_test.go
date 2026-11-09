package notification

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

// TestMessage validates the common portable content boundary directly.
func TestMessage(t *testing.T) {
	for _, message := range []Message{
		{Body: "hello"},
		{Title: "Ticket updated", Body: "T-42 changed"},
		{Title: strings.Repeat("x", 256), Body: strings.Repeat("x", 16384)},
	} {
		if err := message.Validate(); err != nil {
			t.Fatal(err)
		}
	}
	for _, message := range []Message{
		{},
		{Body: " "},
		{Title: "bad\r\nheader", Body: "hello"},
		{Title: string([]byte{255}), Body: "hello"},
		{Body: string([]byte{255})},
		{Title: strings.Repeat("x", 257), Body: "hello"},
		{Body: strings.Repeat("x", 16385)},
	} {
		if !errors.Is(message.Validate(), ErrInvalidMessage) {
			t.Fatal("invalid message accepted")
		}
	}
}

// TestSimpleSend exercises validation, redaction and one-call transport semantics.
func TestSimpleSend(t *testing.T) {
	calls := 0
	driver := SenderFunc(func(ctx context.Context, message Message) error {
		calls++
		if ctx == nil || message.Title != "Ticket" || message.Body != "T-42" {
			t.Fatal("message changed")
		}
		return nil
	})
	notifier := New(driver)
	message := Message{Title: "Ticket", Body: "T-42"}
	if err := notifier.Send(nil, message); err != nil || calls != 1 {
		t.Fatal(err)
	}
	if err := notifier.Send(nil, Message{}); !errors.Is(err, ErrInvalidMessage) || calls != 1 {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := notifier.Send(ctx, message); !errors.Is(err, context.Canceled) || calls != 1 {
		t.Fatal(err)
	}
	for _, cause := range []error{ErrThrottled, ErrUnauthorized, ErrPermanent, ErrRetryable, ErrOutcomeUnknown, ErrRender, ErrInvalidRequest, ErrInvalidMessage, context.Canceled, context.DeadlineExceeded, errors.New("secret provider response")} {
		notifier := New(SenderFunc(func(context.Context, Message) error { return cause }))
		err := notifier.Send(nil, message)
		if err == nil || strings.Contains(err.Error(), "secret") {
			t.Fatal("driver prose leaked")
		}
		if cause.Error() != "secret provider response" && !errors.Is(err, cause) {
			t.Fatal("classification lost")
		}
	}
}

// TestConcurrentSimpleSend proves callers can share a notifier with a safe driver.
func TestConcurrentSimpleSend(t *testing.T) {
	var calls atomic.Int64
	notifier := New(SenderFunc(func(context.Context, Message) error { calls.Add(1); return nil }))
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := notifier.Send(nil, Message{Body: "hello"}); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if calls.Load() != 16 {
		t.Fatal("missing sends")
	}
}
