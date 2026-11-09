package notification

import "time"

// Status describes evidence about one planned delivery.
type Status string

const (
	// Accepted means the provider acknowledged work, without proving receipt.
	Accepted Status = "accepted"
	// Delivered requires synchronous transport evidence of actual delivery.
	Delivered Status = "delivered"
	// Failed means an attempt was rejected or could not be performed.
	Failed Status = "failed"
	// Unknown preserves uncertainty about provider acceptance.
	Unknown Status = "unknown"
	// Skipped means ordered fallback or cancellation prevented an attempt.
	Skipped Status = "skipped"
)

// Outcome contains bounded transport evidence without provider prose or IDs.
type Outcome struct {
	Status          Status
	RetryAfter      time.Duration
	IdempotencyUsed bool
}

// DeliveryResult identifies a delivery through an opaque key without recipient endpoints.
type DeliveryResult struct {
	Key     string
	Channel string
	Driver  string
	Outcome Outcome
	Error   error
}

// Result preserves all bounded outcomes when recipient policy fails.
type Result struct {
	RequestID  string
	Deliveries []DeliveryResult
	Suppressed int
	Deferred   int
	Succeeded  bool
}

// Clone isolates returned outcomes from journal and fake storage.
func (r Result) Clone() Result {
	r.Deliveries = append([]DeliveryResult(nil), r.Deliveries...)
	return r
}

// Observation exposes bounded categorical labels without content or identities.
type Observation struct {
	Route    string
	Channel  string
	Driver   string
	Status   Status
	Duration time.Duration
}

// Observer consumes observations synchronously and must be concurrency-safe.
// Observe must return promptly; the dispatcher isolates hook panics.
type Observer interface{ Observe(Observation) }

// ObserverFunc adapts a non-blocking observation hook.
type ObserverFunc func(Observation)

// Observe invokes the hook; the dispatcher isolates panics at the boundary.
func (f ObserverFunc) Observe(o Observation) { f(o) }

// observe isolates optional observer panics from delivery results.
func observe(observer Observer, o Observation) {
	if observer == nil {
		return
	}
	defer func() { _ = recover() }()
	observer.Observe(o)
}

// accepted identifies positive provider evidence independently of Go errors.
func accepted(s Status) bool { return s == Accepted || s == Delivered }
