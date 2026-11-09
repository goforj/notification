// Package goalert submits and closes alerts through GoAlert Generic Incoming API.
package goalert

import (
	"context"
	"net/http"

	"github.com/goforj/notification"
	"github.com/goforj/notification/internal/provider"
)

// Config fixes a trusted integration and optional ordinary Send correlation key.
type Config struct {
	URL   string
	Token string
	Key   notification.IncidentKey
	HTTP  notification.HTTPConfig
}

// Driver submits native lifecycle events synchronously without polling or retries.
type Driver struct{ incident *provider.Incident }

// New validates configuration without contacting the integration.
func New(config Config) (*Driver, error) {
	base, err := provider.BaseURL(config.URL, "")
	if err != nil || !provider.Token(config.Token) {
		return nil, notification.ErrInvalidRequest
	}
	driver, err := provider.NewIncident("goalert", config.HTTP, config.Key, func(action string, event notification.Incident) (provider.Request, error) {
		if len(event.Body) > 6144 {
			return provider.Request{}, notification.ErrRender
		}
		operation := ""
		if action == "resolve" {
			operation = "close"
		}
		return provider.PostJSON(base+"/api/v2/generic/incoming", http.Header{"Authorization": {"Bearer " + config.Token}}, struct {
			Summary string                   `json:"summary,omitempty"`
			Details string                   `json:"details,omitempty"`
			Dedup   notification.IncidentKey `json:"dedup"`
			Action  string                   `json:"action,omitempty"`
		}{event.Title, event.Body, event.Key, operation}), nil
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
