package notification

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// reject scripts a classified configured target without copying provider prose.
func reject(err error) Sender {
	return SenderFunc(func(context.Context, Message) error { return err })
}

// TestFanout retains ordered failures, metadata and immutable sender configuration.
func TestFanout(t *testing.T) {
	provider := NewDeliveryError("telegram", Outcome{Status: Failed, RetryAfter: time.Minute}, ErrThrottled)
	senders := []Sender{reject(ErrUnauthorized), reject(fmt.Errorf("private: %w", provider)), reject(nil)}
	fanout := Fanout(senders...)
	senders[0] = reject(nil)
	err := New(fanout).Send(nil, Message{Body: "hello"})
	if !errors.Is(err, ErrUnauthorized) || !errors.Is(err, ErrThrottled) || strings.Contains(err.Error(), "private") {
		t.Fatal("aggregate classifications lost or prose leaked")
	}
	joined := err.(interface{ Unwrap() []error }).Unwrap()
	if len(joined) != 2 {
		t.Fatal("missing target failures")
	}
	for index, failure := range joined {
		var target *TargetError
		if !errors.As(failure, &target) || target.Index() != index {
			t.Fatal("target order lost")
		}
	}
	var delivery *DeliveryError
	if !errors.As(err, &delivery) || delivery.Driver() != "telegram" || delivery.Status() != Failed || delivery.RetryAfter() != time.Minute {
		t.Fatal("provider evidence lost")
	}
	if Fanout(reject(nil)).Send(nil, Message{Body: "hello"}) != nil {
		t.Fatal("successful target failed")
	}
}

// TestCompositionValidation rejects invalid inputs before collaborators run.
func TestCompositionValidation(t *testing.T) {
	for _, constructor := range []func(...Sender) Sender{Fanout, Fallback} {
		for _, count := range []int{0, 65} {
			func() {
				defer func() {
					if recover() != ErrInvalidRequest {
						t.Fatal("invalid configuration did not fail fast")
					}
				}()
				constructor(make([]Sender, count)...)
			}()
		}
		sender := constructor(reject(nil))
		if err := sender.Send(nil, Message{}); !errors.Is(err, ErrInvalidMessage) {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err := sender.Send(ctx, Message{Body: "hello"}); !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	}
}

// TestBoundedFanout proves the per-call concurrency limit and cancellation boundary.
func TestBoundedFanout(t *testing.T) {
	var calls atomic.Int64
	entered := make(chan struct{}, 8)
	release := make(chan struct{})
	senders := make([]Sender, 64)
	for i := range senders {
		senders[i] = SenderFunc(func(context.Context, Message) error {
			calls.Add(1)
			entered <- struct{}{}
			<-release
			return nil
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- Fanout(senders...).Send(ctx, Message{Body: "hello"}) }()
	for i := 0; i < 8; i++ {
		<-entered
	}
	if calls.Load() != 8 {
		t.Fatal("unbounded fanout")
	}
	cancel()
	close(release)
	if err := <-done; !errors.Is(err, context.Canceled) || calls.Load() != 8 {
		t.Fatal("canceled work was started")
	}
	failures := make([]Sender, 64)
	for i := range failures {
		failures[i] = reject(ErrPermanent)
	}
	err := New(Fanout(failures...)).Send(nil, Message{Body: "hello"})
	if len(err.(interface{ Unwrap() []error }).Unwrap()) != 64 {
		t.Fatal("facade truncated valid target failures")
	}
}

// TestFallback distinguishes rejected attempts from ambiguous acceptance and bad wiring.
func TestFallback(t *testing.T) {
	for _, cause := range []error{ErrPermanent, ErrUnauthorized, ErrRender} {
		if err := Fallback(reject(cause), reject(nil)).Send(nil, Message{Body: "hello"}); err != nil {
			t.Fatal(err)
		}
	}
	for _, cause := range []error{ErrRetryable, ErrThrottled, ErrOutcomeUnknown, context.Canceled, context.DeadlineExceeded, ErrInvalidRequest, ErrInvalidMessage, errors.New("private")} {
		calls := 0
		last := SenderFunc(func(context.Context, Message) error { calls++; return nil })
		err := Fallback(reject(cause), last).Send(nil, Message{Body: "hello"})
		if err == nil || calls != 0 || strings.Contains(err.Error(), "private") {
			t.Fatal("unsafe fallback")
		}
	}
	err := Fallback(reject(ErrPermanent), reject(ErrUnauthorized)).Send(nil, Message{Body: "hello"})
	if !errors.Is(err, ErrPermanent) || !errors.Is(err, ErrUnauthorized) {
		t.Fatal("fallback errors lost")
	}
	ctx, cancel := context.WithCancel(context.Background())
	first := SenderFunc(func(context.Context, Message) error { cancel(); return ErrPermanent })
	if err := Fallback(first, reject(nil)).Send(ctx, Message{Body: "hello"}); !errors.Is(err, context.Canceled) {
		t.Fatal("cancellation lost between attempts")
	}
}
