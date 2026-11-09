// Package firehydrant submits correlated OPEN and CLOSED events to FireHydrant Signals.
package firehydrant

import (
	"context"

	"github.com/goforj/notification"
	"github.com/goforj/notification/internal/provider"
)

// Config fixes a trusted integration and optional ordinary Send correlation key.
type Config struct {
	URL  string
	Key  notification.IncidentKey
	HTTP notification.HTTPConfig
}

// Driver submits native lifecycle events synchronously without polling or retries.
type Driver struct{ incident *provider.Incident }

// New validates configuration without contacting the integration.
func New(config Config) (*Driver, error) {
	if provider.Endpoint(config.URL) != nil {
		return nil, notification.ErrInvalidRequest
	}
	driver, err := provider.NewIncident("firehydrant", config.HTTP, config.Key, func(action string, event notification.Incident) (provider.Request, error) {
		summary := event.Title
		if summary == "" {
			summary = "Notification resolved"
		}
		status := "OPEN"
		if action == "resolve" {
			status = "CLOSED"
		}
		level := "INFO"
		switch event.Severity {
		case notification.SeverityWarning:
			level = "WARN"
		case notification.SeverityError, notification.SeverityCritical:
			level = "ERROR"
		}
		return provider.PostJSON(config.URL, nil, struct {
			Summary string                   `json:"summary,omitempty"`
			Body    string                   `json:"body,omitempty"`
			Key     notification.IncidentKey `json:"idempotency_key"`
			Status  string                   `json:"status"`
			Level   string                   `json:"level"`
		}{summary, event.Body, event.Key, status, level}), nil
	}, nil)
	if err != nil {
		return nil, err
	}
	return &Driver{incident: driver}, nil
}

// Send triggers the fixed configured incident key with ordinary message content.
func (d *Driver) Send(ctx context.Context, message notification.Message) error {
	return d.incident.Send(ctx, message)
}

// Trigger submits an explicit stable key independently of display names.
func (d *Driver) Trigger(ctx context.Context, event notification.Incident) error {
	return d.incident.Trigger(ctx, event)
}

// Resolve submits recovery for the same application-owned incident key.
func (d *Driver) Resolve(ctx context.Context, key notification.IncidentKey) error {
	return d.incident.Resolve(ctx, key)
}

// Close releases only owned idle connections; injected transports remain caller-owned.
func (d *Driver) Close() error { return d.incident.Close() }
