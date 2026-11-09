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
	if len(err.(interface{ Unwrap() []error }).Unwrap()) != 65 || !errors.Is(err, ErrOutcomeUnknown) {
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

// multipleClassifications models a custom driver error with more than one safe classification.
type multipleClassifications struct{ classes []error }

// Error provides private diagnostics that must stay out of public errors.
func (multipleClassifications) Error() string { return "private multi-class provider response" }

// Is exposes independent classifications without requiring an unwrap chain.
func (e multipleClassifications) Is(target error) bool {
	for _, class := range e.classes {
		if target == class {
			return true
		}
	}
	return false
}

// TestCustomErrorPreservesAllClassifications keeps uncertainty and cancellation visible through the facade.
func TestCustomErrorPreservesAllClassifications(t *testing.T) {
	cause := multipleClassifications{[]error{ErrPermanent, ErrOutcomeUnknown, context.Canceled}}
	err := New(reject(cause)).Send(context.Background(), Message{Body: "public fixture"})
	for _, class := range cause.classes {
		if !errors.Is(err, class) {
			t.Fatalf("classification lost: %v", class)
		}
	}
	if strings.Contains(err.Error(), "private") {
		t.Fatal("provider prose disclosed")
	}
}

// classifiedWrapper supplies definite outer evidence alongside a standard wrapped cause.
type classifiedWrapper struct{ cause error }

// Error provides private context that is deliberately stripped from public diagnostics.
func (classifiedWrapper) Error() string { return "private wrapper" }

// Is supplies the wrapper's rejection classification independently of its cause.
func (classifiedWrapper) Is(target error) bool { return target == ErrPermanent }

// Unwrap permits errors.Is to discover more specific evidence in the cause.
func (e classifiedWrapper) Unwrap() error { return e.cause }

// classifiedGroup supplies outer evidence for an independently classified error group.
type classifiedGroup struct{ children []error }

// Error keeps provider diagnostics outside the public error chain.
func (classifiedGroup) Error() string { return "private classified group" }

// Is describes a rejected operation without hiding contradictory child evidence.
func (classifiedGroup) Is(target error) bool { return target == ErrPermanent }

// Unwrap exposes bounded child evidence to the sanitizer.
func (e classifiedGroup) Unwrap() []error { return e.children }

// TestClassifiedWrapping preserves explicit causes without turning explanatory prose into uncertainty.
func TestClassifiedWrapping(t *testing.T) {
	for _, cause := range []error{nil, errors.New("private explanation"), fmt.Errorf("private: %w", errors.New("private nested explanation")), classifiedGroup{[]error{errors.New("private explanation")}}} {
		err := safeError(classifiedWrapper{cause}, ErrOutcomeUnknown)
		if !errors.Is(err, ErrPermanent) || errors.Is(err, ErrOutcomeUnknown) || strings.Contains(err.Error(), "private") {
			t.Fatal("classified rejection changed")
		}
	}
	for _, cause := range []error{ErrOutcomeUnknown, fmt.Errorf("private: %w", ErrOutcomeUnknown), classifiedGroup{[]error{ErrOutcomeUnknown}}} {
		err := safeError(classifiedWrapper{cause}, ErrOutcomeUnknown)
		if !errors.Is(err, ErrPermanent) || !errors.Is(err, ErrOutcomeUnknown) || strings.Contains(err.Error(), "private") {
			t.Fatal("wrapped evidence lost")
		}
	}
	children := make([]error, 65)
	for i := range children {
		children[i] = ErrPermanent
	}
	for _, cause := range []error{classifiedWrapper{&errorCycle{}}, classifiedGroup{children}} {
		if err := safeError(cause, ErrOutcomeUnknown); !errors.Is(err, ErrOutcomeUnknown) {
			t.Fatal("truncation lost uncertainty")
		}
	}
}
