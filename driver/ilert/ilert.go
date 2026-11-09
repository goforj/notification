// Package ilert submits native ALERT, ACCEPT and RESOLVE events to ilert.
package ilert

import (
	"context"
	"net/http"

	"github.com/goforj/notification"
	"github.com/goforj/notification/internal/provider"
)

// Config fixes a trusted integration and optional ordinary Send correlation key.
type Config struct {
	IntegrationKey string
	URL            string
	Key            notification.IncidentKey
	HTTP           notification.HTTPConfig
}

// Driver submits native lifecycle events synchronously without polling or retries.
type Driver struct{ incident *provider.Incident }

// New validates configuration without contacting the integration.
func New(config Config) (*Driver, error) {
	if config.URL == "" {
		config.URL = "https://api.ilert.com/api/events"
	}
	if provider.Endpoint(config.URL) != nil || !provider.Token(config.IntegrationKey) {
		return nil, notification.ErrInvalidRequest
	}
	driver, err := provider.NewIncident("ilert", config.HTTP, config.Key, func(action string, event notification.Incident) (provider.Request, error) {
		eventType := "ALERT"
		switch action {
		case "acknowledge":
			eventType = "ACCEPT"
		case "resolve":
			eventType = "RESOLVE"
		}
		return provider.PostJSON(config.URL, nil, struct {
			Key      string                   `json:"integrationKey"`
			Type     string                   `json:"eventType"`
			AlertKey notification.IncidentKey `json:"alertKey"`
			Summary  string                   `json:"summary,omitempty"`
			Details  string                   `json:"details,omitempty"`
		}{config.IntegrationKey, eventType, event.Key, event.Title, event.Body}), nil
	}, nil, http.StatusAccepted)
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

// Acknowledge submits the provider's native acknowledgement action.
func (d *Driver) Acknowledge(ctx context.Context, key notification.IncidentKey) error {
	return d.incident.Acknowledge(ctx, key)
}
