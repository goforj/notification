// Package opsgenie supports existing Opsgenie alert integrations during migration to JSM.
// Opsgenie is a legacy service scheduled to shut down on April 5, 2027.
package opsgenie

import (
	"context"
	"encoding/json"
	"github.com/goforj/notification"
	"github.com/goforj/notification/internal/provider"
	"net/http"
	"net/url"
	"regexp"
	"unicode/utf8"
)

// Config supplies a legacy API integration key and optional ordinary Send defaults.
// BaseURL supports the EU endpoint or a trusted proxy.
type Config struct {
	Token    string
	BaseURL  string
	Key      notification.IncidentKey
	Source   string
	Priority string
	HTTP     notification.HTTPConfig
}

// Driver owns the legacy alert transport without polling asynchronous request state.
type Driver struct {
	client *provider.Client
	config Config
}

var priorityPattern = regexp.MustCompile(`^P[1-5]$`)

// New validates configuration without triggering an alert.
func New(config Config) (*Driver, error) {
	if config.Source == "" {
		config.Source = "notification"
	}
	if config.Priority == "" {
		config.Priority = "P3"
	}
	key := config.Key
	if key == "" {
		key = "validation"
	}
	if !provider.Token(config.Token) || !priorityPattern.MatchString(config.Priority) || utf8.RuneCountInString(config.Source) > 100 || (notification.Incident{Key: key, Title: "validation", Body: "validation", Source: config.Source}).Validate() != nil {
		return nil, notification.ErrInvalidRequest
	}
	base, err := provider.BaseURL(config.BaseURL, "https://api.opsgenie.com")
	if err != nil {
		return nil, err
	}
	config.BaseURL = base
	client, err := provider.New("opsgenie", config.HTTP)
	if err != nil {
		return nil, err
	}
	return &Driver{client, config}, nil
}

// Close releases owned idle connections.
func (d *Driver) Close() error { return d.client.Close() }

// Send triggers the explicitly configured stable alias.
func (d *Driver) Send(ctx context.Context, message notification.Message) error {
	return d.Trigger(ctx, notification.Incident{Key: d.config.Key, Title: message.Title, Body: message.Body})
}

// Trigger submits native alert fields with an immutable application correlation key.
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
	if utf8.RuneCountInString(incident.Title) > 130 || utf8.RuneCountInString(incident.Body) > 15000 || utf8.RuneCountInString(incident.Source) > 100 {
		return notification.NewDeliveryError("opsgenie", notification.Outcome{Status: notification.Failed}, notification.ErrRender)
	}
	priority := d.config.Priority
	switch incident.Severity {
	case notification.SeverityCritical:
		priority = "P1"
	case notification.SeverityError:
		priority = "P2"
	case notification.SeverityWarning:
		priority = "P3"
	case notification.SeverityInfo:
		priority = "P4"
	}
	body := provider.JSON(struct {
		Message     string                   `json:"message"`
		Alias       notification.IncidentKey `json:"alias"`
		Description string                   `json:"description"`
		Source      string                   `json:"source"`
		Priority    string                   `json:"priority"`
	}{incident.Title, incident.Key, incident.Body, incident.Source, priority})
	return d.request(ctx, d.config.BaseURL+"/v2/alerts", body)
}

// Acknowledge correlates by alias, preserving identity across display-name changes.
func (d *Driver) Acknowledge(ctx context.Context, key notification.IncidentKey) error {
	return d.action(ctx, key, "acknowledge")
}

// Resolve closes the alert with the same stable alias used by Trigger.
func (d *Driver) Resolve(ctx context.Context, key notification.IncidentKey) error {
	return d.action(ctx, key, "close")
}

// action uses Opsgenie's documented alias identifier instead of a display name.
func (d *Driver) action(ctx context.Context, key notification.IncidentKey, action string) error {
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
		Source string `json:"source"`
	}{d.config.Source})
	return d.request(ctx, d.config.BaseURL+"/v2/alerts/"+url.PathEscape(string(key))+"/"+action+"?identifierType=alias", body)
}

// request checks asynchronous acceptance without claiming the alert mutation completed.
func (d *Driver) request(ctx context.Context, endpoint string, body []byte) error {
	return d.client.Request(ctx, http.MethodPost, endpoint, http.Header{"Content-Type": {"application/json"}, "Authorization": {"GenieKey " + d.config.Token}}, body, func(data []byte) error {
		var ack struct {
			RequestID string `json:"requestId"`
		}
		if json.Unmarshal(data, &ack) != nil || ack.RequestID == "" {
			return notification.ErrOutcomeUnknown
		}
		return nil
	})
}
