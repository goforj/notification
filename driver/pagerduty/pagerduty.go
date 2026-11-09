// Package pagerduty submits correlated Events API v2 incident operations.
package pagerduty

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/goforj/notification"
	"github.com/goforj/notification/internal/provider"
)

// Config supplies an integration routing key. Key is optional for explicit
// Trigger calls and required when using the ordinary Send message capability.
type Config struct {
	RoutingKey string
	URL        string
	Key        notification.IncidentKey
	Source     string
	Severity   notification.Severity
	HTTP       notification.HTTPConfig
}

// Driver owns an Events API transport and immutable integration configuration.
type Driver struct {
	client *provider.Client
	config Config
}

// New validates configuration without sending a test event.
func New(config Config) (*Driver, error) {
	if config.URL == "" {
		config.URL = "https://events.pagerduty.com/v2/enqueue"
	}
	if config.Source == "" {
		config.Source = "notification"
	}
	if config.Severity == "" {
		config.Severity = notification.SeverityWarning
	}
	key := config.Key
	if key == "" {
		key = "validation"
	}
	if len(config.RoutingKey) != 32 || !provider.Token(config.RoutingKey) || provider.Endpoint(config.URL) != nil || (notification.Incident{Key: key, Title: "validation", Body: "validation", Source: config.Source, Severity: config.Severity}).Validate() != nil {
		return nil, notification.ErrInvalidRequest
	}
	client, err := provider.New("pagerduty", config.HTTP)
	if err != nil {
		return nil, err
	}
	return &Driver{client, config}, nil
}

// Close releases idle connections; applications own all workflow scheduling.
func (d *Driver) Close() error { return d.client.Close() }

// Send triggers the configured fixed incident key with ordinary message content.
func (d *Driver) Send(ctx context.Context, message notification.Message) error {
	return d.Trigger(ctx, notification.Incident{Key: d.config.Key, Title: message.Title, Body: message.Body})
}

// Trigger submits an explicit stable key and requires its echo in the acknowledgement.
func (d *Driver) Trigger(ctx context.Context, incident notification.Incident) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := incident.Validate(); err != nil {
		return err
	}
	if incident.Source == "" {
		incident.Source = d.config.Source
	}
	if incident.Severity == "" {
		incident.Severity = d.config.Severity
	}
	payload := &eventPayload{incident.Title, incident.Source, incident.Severity, struct {
		Message string `json:"message"`
	}{incident.Body}}
	return d.event(ctx, "trigger", incident.Key, payload)
}

// Acknowledge submits an explicit provider acknowledgement action.
func (d *Driver) Acknowledge(ctx context.Context, key notification.IncidentKey) error {
	return d.event(ctx, "acknowledge", key, nil)
}

// Resolve resolves the open alert correlated by the same routing and incident keys.
func (d *Driver) Resolve(ctx context.Context, key notification.IncidentKey) error {
	return d.event(ctx, "resolve", key, nil)
}

// eventPayload keeps trigger-only fields out of acknowledge and resolve requests.
type eventPayload struct {
	Summary  string                `json:"summary"`
	Source   string                `json:"source"`
	Severity notification.Severity `json:"severity"`
	Details  struct {
		Message string `json:"message"`
	} `json:"custom_details"`
}

// event applies common cancellation, key and acknowledgement contracts to all actions.
func (d *Driver) event(ctx context.Context, action string, key notification.IncidentKey, payload *eventPayload) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := key.Validate(); err != nil {
		return err
	}
	body := provider.JSON(struct {
		RoutingKey string                   `json:"routing_key"`
		Action     string                   `json:"event_action"`
		Key        notification.IncidentKey `json:"dedup_key"`
		Payload    *eventPayload            `json:"payload,omitempty"`
	}{d.config.RoutingKey, action, key, payload})
	return d.client.Request(ctx, http.MethodPost, d.config.URL, http.Header{"Content-Type": []string{"application/json"}}, body, func(data []byte) error {
		var ack struct {
			Status string `json:"status"`
			Key    string `json:"dedup_key"`
		}
		if json.Unmarshal(data, &ack) != nil || ack.Status != "success" || ack.Key != string(key) {
			return notification.ErrOutcomeUnknown
		}
		return nil
	})
}
