package notification

import (
	"context"
	"errors"
	"sync"
)

// composedSender owns a copied list of borrowed, concurrency-safe senders.
type composedSender struct {
	senders  []Sender
	fallback bool
}

// Fanout delivers to every configured sender with at most eight concurrent calls.
// It waits for all started calls and returns every bounded target failure in input
// order. Configure one to 64 senders; invalid counts panic during construction.
func Fanout(senders ...Sender) Sender {
	return compose(false, senders)
}

// Fallback tries senders in order until one handles the message. Only a permanent
// rejection, unauthorized provider, or unsupported rendering advances fallback.
// Uncertain acceptance, transient failure, cancellation and invalid wiring stop it.
// Configure one to 64 senders; invalid counts panic during construction.
func Fallback(senders ...Sender) Sender {
	return compose(true, senders)
}

// compose copies construction-time routing without taking resource ownership.
func compose(fallback bool, senders []Sender) Sender {
	if len(senders) == 0 || len(senders) > 64 {
		panic(ErrInvalidRequest)
	}
	return &composedSender{senders: append([]Sender(nil), senders...), fallback: fallback}
}

// Send validates once before routing and never creates lifecycle goroutines or retries.
func (s *composedSender) Send(ctx context.Context, message Message) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := message.Validate(); err != nil {
		return err
	}
	if s.fallback {
		return s.sendFallback(ctx, message)
	}
	failures := make([]error, len(s.senders))
	jobs := make(chan int, len(s.senders))
	for index := range s.senders {
		jobs <- index
	}
	close(jobs)
	var workers sync.WaitGroup
	for worker := 0; worker < min(8, len(s.senders)); worker++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for index := range jobs {
				err := ctx.Err()
				if err == nil {
					err = s.senders[index].Send(ctx, message)
				}
				if err != nil {
					failures[index] = &TargetError{index, safeError(err, ErrOutcomeUnknown)}
				}
			}
		}()
	}
	workers.Wait()
	return errors.Join(failures...)
}

// sendFallback advances only on classified rejection and retains ordered failures.
func (s *composedSender) sendFallback(ctx context.Context, message Message) error {
	var failures []error
	for index, sender := range s.senders {
		if err := ctx.Err(); err != nil {
			return errors.Join(append(failures, err)...)
		}
		err := sender.Send(ctx, message)
		if err == nil {
			return nil
		}
		safe := safeError(err, ErrOutcomeUnknown)
		failures = append(failures, &TargetError{index, safe})
		if errors.Is(safe, ErrOutcomeUnknown) || errors.Is(safe, context.Canceled) || errors.Is(safe, context.DeadlineExceeded) || errors.Is(safe, ErrRetryable) || errors.Is(safe, ErrThrottled) {
			break
		}
		if !errors.Is(safe, ErrPermanent) && !errors.Is(safe, ErrUnauthorized) && !errors.Is(safe, ErrRender) {
			break
		}
	}
	return errors.Join(failures...)
}
