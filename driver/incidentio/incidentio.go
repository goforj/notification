// Package incidentio submits correlated firing and recovery events to incident.io HTTP alert sources.
package incidentio

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/goforj/notification"
	"github.com/goforj/notification/internal/provider"
)

// Config fixes a trusted integration and optional ordinary Send correlation key.
type Config struct {
	Token    string
	SourceID string
	BaseURL  string
	Key      notification.IncidentKey
	HTTP     notification.HTTPConfig
}

// Driver submits native lifecycle events synchronously without polling or retries.
type Driver struct{ incident *provider.Incident }

// New validates configuration without contacting the integration.
func New(config Config) (*Driver, error) {
	base, err := provider.BaseURL(config.BaseURL, "https://api.incident.io")
	if err != nil || !provider.Token(config.Token) || !provider.Token(config.SourceID) {
		return nil, notification.ErrInvalidRequest
	}
	driver, err := provider.NewIncident("incidentio", config.HTTP, config.Key, func(action string, event notification.Incident) (provider.Request, error) {
		status := "firing"
		if action == "resolve" {
			status = "resolved"
		}
		title := event.Title
		if title == "" {
			title = "Notification resolved"
		}
		return provider.PostJSON(base+"/v2/alert_events/http/"+url.PathEscape(config.SourceID), http.Header{"Authorization": {"Bearer " + config.Token}}, struct {
			Title       string                   `json:"title"`
			Description string                   `json:"description,omitempty"`
			Key         notification.IncidentKey `json:"deduplication_key"`
			Status      string                   `json:"status"`
		}{title, event.Body, event.Key, status}), nil
	}, func(event notification.Incident, data []byte) error {
		var ack struct {
			Status string                   `json:"status"`
			Key    notification.IncidentKey `json:"deduplication_key"`
		}
		if json.Unmarshal(data, &ack) != nil || ack.Status != "success" || ack.Key != event.Key {
			return notification.ErrOutcomeUnknown
		}
		return nil
	}, http.StatusAccepted)
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
