package notification

import (
	"context"
	"errors"
)

var (
	// ErrInvalidRequest indicates malformed or oversized caller intent.
	ErrInvalidRequest = errors.New("notification: invalid request")
	// ErrInvalidPlan indicates an unscoped, incomplete, or incompatible plan.
	ErrInvalidPlan = errors.New("notification: invalid plan")
	// ErrUnknownRoute indicates a route absent from the compiled catalog.
	ErrUnknownRoute = errors.New("notification: unknown route")
	// ErrConflict indicates reuse of a scoped ID with different intent.
	ErrConflict = errors.New("notification: conflicting request")
	// ErrInProgress indicates that another call owns this scoped request.
	ErrInProgress = errors.New("notification: in progress")
	// ErrNoDelivery indicates an unsatisfied recipient policy.
	ErrNoDelivery = errors.New("notification: no successful delivery")
	// ErrSuppressed indicates policy suppression of a recipient.
	ErrSuppressed = errors.New("notification: suppressed")
	// ErrDeferred indicates work requiring application-owned scheduling.
	ErrDeferred = errors.New("notification: deferred")
	// ErrRender indicates invalid or unsupported rendered content.
	ErrRender = errors.New("notification: render failed")
	// ErrRetryable indicates a transient transport failure.
	ErrRetryable = errors.New("notification: retryable failure")
	// ErrThrottled indicates provider rate limiting.
	ErrThrottled = errors.New("notification: throttled")
	// ErrUnauthorized indicates rejected provider credentials.
	ErrUnauthorized = errors.New("notification: unauthorized provider")
	// ErrPermanent indicates a non-retryable provider rejection.
	ErrPermanent = errors.New("notification: permanent failure")
	// ErrOutcomeUnknown indicates insufficient evidence about provider acceptance.
	ErrOutcomeUnknown = errors.New("notification: outcome unknown")
)

// safeError prevents untrusted collaborator prose from entering public errors.
func safeError(err error, fallback error) error {
	for _, safe := range []error{context.Canceled, context.DeadlineExceeded, ErrRender, ErrThrottled, ErrUnauthorized, ErrPermanent, ErrRetryable, ErrOutcomeUnknown} {
		if errors.Is(err, safe) {
			return safe
		}
	}
	return fallback
}
