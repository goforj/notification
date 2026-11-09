// Package pagertree submits correlated events to PagerTree incoming integrations.
package pagertree

import (
	"context"
	"net/http"

	"github.com/goforj/notification"
	"github.com/goforj/notification/internal/provider"
)

// Config fixes a trusted integration and optional ordinary Send correlation key.
type Config struct {
	URL string
	// Token is the optional integration token configured in PagerTree.
	Token string
	Key   notification.IncidentKey
	HTTP  notification.HTTPConfig
}

// Driver submits native lifecycle events synchronously without polling or retries.
type Driver struct{ incident *provider.Incident }

// New validates configuration without contacting the integration.
func New(config Config) (*Driver, error) {
	if provider.Endpoint(config.URL) != nil || (config.Token != "" && !provider.Token(config.Token)) {
		return nil, notification.ErrInvalidRequest
	}
	headers := http.Header{}
	if config.Token != "" {
		headers.Set("pagertree-token", config.Token)
	}
	driver, err := provider.NewIncident("pagertree", config.HTTP, config.Key, func(action string, event notification.Incident) (provider.Request, error) {
		eventType := "create"
		switch action {
		case "acknowledge":
			eventType = "acknowledge"
		case "resolve":
			eventType = "resolve"
		}
		return provider.PostJSON(config.URL, headers, struct {
			ID          notification.IncidentKey `json:"Id"`
			Type        string                   `json:"event_type"`
			Title       string                   `json:"Title,omitempty"`
			Description string                   `json:"Description,omitempty"`
		}{event.Key, eventType, event.Title, event.Body}), nil
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

// Acknowledge submits the provider's native acknowledgement action.
func (d *Driver) Acknowledge(ctx context.Context, key notification.IncidentKey) error {
	return d.incident.Acknowledge(ctx, key)
}
