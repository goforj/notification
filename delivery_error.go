package notification

import (
	"errors"
	"fmt"
	"time"
)

// DeliveryError retains safe provider evidence and retry advice without raw causes.
// Its fields are immutable; use errors.As to inspect it through a Notifier or fanout.
type DeliveryError struct {
	driver     string
	status     Status
	retryAfter time.Duration
	cause      error
}

// NewDeliveryError classifies a failed or uncertain transport operation. Driver
// names must be catalog identifiers; invalid names are replaced with "unknown".
// Provider prose and endpoint values are discarded from the public unwrap chain.
func NewDeliveryError(driver string, outcome Outcome, cause error) *DeliveryError {
	if !identifier.MatchString(driver) {
		driver = "unknown"
	}
	if outcome.Status != Failed && outcome.Status != Unknown {
		outcome.Status = Unknown
	}
	outcome, safe := normalize(outcome, cause)
	return &DeliveryError{
		driver:     driver,
		status:     outcome.Status,
		retryAfter: outcome.RetryAfter,
		cause:      safe,
	}
}

// Error formats only the bounded driver name and sanitized classification.
func (e *DeliveryError) Error() string {
	return fmt.Sprintf("notification: %s: %v", e.driver, e.cause)
}

// Unwrap exposes safe classifications for errors.Is and errors.As.
func (e *DeliveryError) Unwrap() error { return e.cause }

// Driver returns the bounded provider catalog identifier.
func (e *DeliveryError) Driver() string { return e.driver }

// Status distinguishes proved rejection from uncertain provider acceptance.
func (e *DeliveryError) Status() Status { return e.status }

// RetryAfter returns bounded provider scheduling advice without performing a retry.
func (e *DeliveryError) RetryAfter() time.Duration { return e.retryAfter }

// TargetError identifies a failed position in an immutable sender composition.
// The index contains no endpoint or recipient information.
type TargetError struct {
	index int
	cause error
}

// Error formats a target position and its sanitized failure.
func (e *TargetError) Error() string {
	return fmt.Sprintf("notification: target %d: %v", e.index, e.cause)
}

// Unwrap retains safe nested provider evidence.
func (e *TargetError) Unwrap() error { return e.cause }

// Index returns the zero-based position in the configured sender list.
func (e *TargetError) Index() int { return e.index }

// safeLeaf recognizes library classifications without following arbitrary unwrap chains.
func safeLeaf(err error) error {
	for _, candidate := range safeClasses {
		if err == candidate {
			return candidate
		}
	}
	if custom, ok := err.(interface{ Is(error) bool }); ok {
		for _, candidate := range safeClasses {
			if custom.Is(candidate) {
				return candidate
			}
		}
	}
	return nil
}

// sanitize bounds traversal of collaborator errors while preserving owned typed evidence.
func sanitize(err, fallback error, depth int, remaining *int) error {
	*remaining--
	if err == nil || depth > 16 || *remaining < 0 {
		return fallback
	}
	if safe := safeLeaf(err); safe != nil {
		return safe
	}
	switch e := err.(type) {
	case *DeliveryError:
		return &DeliveryError{e.driver, e.status, e.retryAfter, sanitize(e.cause, fallback, depth+1, remaining)}
	case *TargetError:
		return &TargetError{e.index, sanitize(e.cause, fallback, depth+1, remaining)}
	case interface{ Unwrap() []error }:
		children := e.Unwrap()
		if len(children) > 64 {
			children = children[:64]
		}
		var safe []error
		for _, child := range children {
			if child != nil {
				safe = append(safe, sanitize(child, fallback, depth+1, remaining))
			}
		}
		if len(safe) > 0 {
			return errors.Join(safe...)
		}
	case interface{ Unwrap() error }:
		return sanitize(e.Unwrap(), fallback, depth+1, remaining)
	}
	return fallback
}
