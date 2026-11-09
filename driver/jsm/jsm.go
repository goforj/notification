// Package jsm submits Jira Service Management Operations alert lifecycles.
package jsm

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/mail"
	"net/url"
	"regexp"
	"unicode/utf8"

	"github.com/goforj/notification"
	"github.com/goforj/notification/internal/provider"
)

// Config supplies JSM Operations credentials and optional fixed Send defaults.
type Config struct {
	CloudID  string
	Email    string
	Token    string
	BaseURL  string
	Key      notification.IncidentKey
	Source   string
	Priority string
	HTTP     notification.HTTPConfig
}

// Driver owns an immutable JSM Operations integration.
type Driver struct {
	client *provider.Client
	config Config
	auth   string
}

var priorityPattern = regexp.MustCompile(`^P[1-5]$`)

// New validates configuration without looking up or creating an alert.
func New(config Config) (*Driver, error) {
	email, err := mail.ParseAddress(config.Email)
	if err != nil || email.Address != config.Email || email.Name != "" || !provider.Token(config.Token) || !provider.Token(config.CloudID) || len(config.CloudID) > 256 {
		return nil, notification.ErrInvalidRequest
	}
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
	if !priorityPattern.MatchString(config.Priority) || utf8.RuneCountInString(config.Source) > 100 || (notification.Incident{Key: key, Title: "validation", Body: "validation", Source: config.Source}).Validate() != nil {
		return nil, notification.ErrInvalidRequest
	}
	base, err := provider.BaseURL(config.BaseURL, "https://api.atlassian.com/jsm/ops/api/"+url.PathEscape(config.CloudID)+"/v1")
	if err != nil {
		return nil, err
	}
	config.BaseURL = base
	client, err := provider.New("jsm", config.HTTP)
	if err != nil {
		return nil, err
	}
	return &Driver{client, config, "Basic " + base64.StdEncoding.EncodeToString([]byte(config.Email+":"+config.Token))}, nil
}

// Close releases only owned idle connections; injected transports remain caller-owned.
func (d *Driver) Close() error { return d.client.Close() }

// Send triggers the configured fixed incident key.
func (d *Driver) Send(ctx context.Context, message notification.Message) error {
	return d.Trigger(ctx, notification.Incident{Key: d.config.Key, Title: message.Title, Body: message.Body})
}

// Trigger creates an alert using a stable alias and explicit source and severity.
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
	source := incident.Source
	if source == "" {
		source = d.config.Source
	}
	if utf8.RuneCountInString(incident.Title) > 130 || utf8.RuneCountInString(incident.Body) > 15000 || utf8.RuneCountInString(source) > 100 {
		return notification.NewDeliveryError("jsm", notification.Outcome{Status: notification.Failed}, notification.ErrRender)
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
	}{incident.Title, incident.Key, incident.Body, source, priority})
	return d.client.Request(ctx, http.MethodPost, d.config.BaseURL+"/alerts", d.headers(), body, accepted)
}

// Acknowledge applies acknowledgement to the alert identified by the stable alias.
func (d *Driver) Acknowledge(ctx context.Context, key notification.IncidentKey) error {
	return d.action(ctx, key, "acknowledge")
}

// Resolve closes the alert identified by the same stable alias used for its trigger.
func (d *Driver) Resolve(ctx context.Context, key notification.IncidentKey) error {
	return d.action(ctx, key, "close")
}

// headers constructs an independent authenticated request header map.
func (d *Driver) headers() http.Header {
	return http.Header{"Content-Type": []string{"application/json"}, "Authorization": []string{d.auth}, "Accept": []string{"application/json"}}
}

// accepted requires JSM's asynchronous acceptance request identity.
func accepted(data []byte) error {
	var ack struct {
		RequestID string `json:"requestId"`
	}
	if json.Unmarshal(data, &ack) != nil || ack.RequestID == "" {
		return notification.ErrOutcomeUnknown
	}
	return nil
}

// action resolves the stable alias to a native alert ID before mutating alert state.
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
	var id string
	endpoint := d.config.BaseURL + "/alerts/alias?alias=" + url.QueryEscape(string(key))
	err := d.client.Request(ctx, http.MethodGet, endpoint, d.headers(), nil, func(data []byte) error {
		var alert struct {
			ID string `json:"id"`
		}
		if json.Unmarshal(data, &alert) != nil || !provider.Token(alert.ID) || len(alert.ID) > 256 {
			return notification.ErrOutcomeUnknown
		}
		id = alert.ID
		return nil
	})
	if err != nil {
		return err
	}
	body := provider.JSON(struct {
		Source string `json:"source"`
	}{d.config.Source})
	return d.client.Request(ctx, http.MethodPost, d.config.BaseURL+"/alerts/"+url.PathEscape(id)+"/"+action, d.headers(), body, accepted)
}
