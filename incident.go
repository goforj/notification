package notification

import "context"

// Severity is a portable incident importance, independent of any provider's priority scale.
type Severity string

const (
	// SeverityInfo reports an informational incident.
	SeverityInfo Severity = "info"
	// SeverityWarning reports a degraded condition requiring attention.
	SeverityWarning Severity = "warning"
	// SeverityError reports a service failure.
	SeverityError Severity = "error"
	// SeverityCritical reports an urgent service failure.
	SeverityCritical Severity = "critical"
)

// IncidentKey identifies an alert across trigger, acknowledge and resolve operations.
// Applications construct it from stable scoped identities, rather than display names.
type IncidentKey string

// Validate rejects unbounded keys and controls before a provider request.
func (k IncidentKey) Validate() error {
	if !validID(string(k)) || len(k) > 255 {
		return ErrInvalidRequest
	}
	return nil
}

// Incident contains human-facing content and explicit alert correlation.
// Empty Severity and Source use the configured driver's documented defaults.
type Incident struct {
	Key      IncidentKey
	Title    string
	Body     string
	Source   string
	Severity Severity
}

// Validate checks portable incident fields without assuming provider-specific workflow state.
func (i Incident) Validate() error {
	if err := i.Key.Validate(); err != nil {
		return err
	}
	if i.Title == "" || (Message{Title: i.Title, Body: i.Body}).Validate() != nil || (i.Source != "" && !validID(i.Source)) {
		return ErrInvalidMessage
	}
	switch i.Severity {
	case "", SeverityInfo, SeverityWarning, SeverityError, SeverityCritical:
		return nil
	default:
		return ErrInvalidMessage
	}
}

// IncidentSender is an optional provider capability for operational alert lifecycles.
// It does not add workflow state or incident requirements to ordinary Sender implementations.
type IncidentSender interface {
	Trigger(context.Context, Incident) error
	Acknowledge(context.Context, IncidentKey) error
	Resolve(context.Context, IncidentKey) error
}
