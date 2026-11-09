package notification

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

// errorGroup supplies adversarial error trees beyond errors.Join's filtering.
type errorGroup []error

// Error supplies prose that must never be propagated.
func (errorGroup) Error() string { return "private endpoint" }

// Unwrap exposes controlled malformed or oversized error trees.
func (e errorGroup) Unwrap() []error { return e }

// errorCycle exercises the sanitizer's traversal budget.
type errorCycle struct{}

// Error supplies prose that must never be propagated.
func (*errorCycle) Error() string { return "private cycle" }

// Unwrap intentionally cycles to prove bounded sanitization.
func (e *errorCycle) Unwrap() error { return e }

// classifiedError models a provider's stable Is contract with private prose.
type classifiedError struct{ matches bool }

// Error supplies untrusted operational diagnostics.
func (classifiedError) Error() string { return "private provider text" }

// Is exposes only a stable public failure classification.
func (e classifiedError) Is(target error) bool { return e.matches && target == ErrThrottled }

// TestDeliveryError keeps evidence, clamps retry advice and never unwraps raw causes.
func TestDeliveryError(t *testing.T) {
	err := NewDeliveryError("telegram", Outcome{Status: Unknown, RetryAfter: 48 * time.Hour}, fmt.Errorf("private: %w", context.Canceled))
	if err.Driver() != "telegram" || err.Status() != Unknown || err.RetryAfter() != 24*time.Hour || !errors.Is(err, context.Canceled) || !errors.Is(err, ErrOutcomeUnknown) || strings.Contains(err.Error(), "private") {
		t.Fatal("unsafe delivery evidence")
	}
	invalid := NewDeliveryError("https://private.example", Outcome{Status: Accepted, RetryAfter: -1}, nil)
	if invalid.Driver() != "unknown" || invalid.Status() != Unknown || invalid.RetryAfter() != 0 || !errors.Is(invalid, ErrOutcomeUnknown) {
		t.Fatal("invalid evidence escaped")
	}
	if !errors.Is(NewDeliveryError("driver", Outcome{Status: Failed}, nil), ErrPermanent) {
		t.Fatal("unclassified rejection")
	}
}

// TestSanitizer bounds adversarial error shapes while preserving safe leaf contracts.
func TestSanitizer(t *testing.T) {
	for _, cause := range []error{nil, errorGroup{}, errorGroup{nil}, &errorCycle{}, classifiedError{}, fmt.Errorf("private: %w", errors.New("private raw"))} {
		err := safeError(cause, ErrOutcomeUnknown)
		if !errors.Is(err, ErrOutcomeUnknown) || strings.Contains(err.Error(), "private") {
			t.Fatal("unsafe cause")
		}
	}
	if !errors.Is(safeError(classifiedError{true}, ErrOutcomeUnknown), ErrThrottled) {
		t.Fatal("custom classification lost")
	}
	wide := make(errorGroup, 70)
	for i := range wide {
		wide[i] = ErrPermanent
	}
	err := safeError(wide, ErrOutcomeUnknown)
	if len(err.(interface{ Unwrap() []error }).Unwrap()) != 64 {
		t.Fatal("error width was not bounded")
	}
	var tree error = ErrPermanent
	for i := 0; i < 4; i++ {
		group := make(errorGroup, 8)
		for j := range group {
			group[j] = tree
		}
		tree = group
	}
	if strings.Contains(safeError(tree, ErrOutcomeUnknown).Error(), "private") {
		t.Fatal("budget overflow leaked")
	}
	if !errors.Is(safeError(errors.Join(ErrUnauthorized, ErrThrottled), ErrOutcomeUnknown), ErrUnauthorized) {
		t.Fatal("joined classification lost")
	}
}
