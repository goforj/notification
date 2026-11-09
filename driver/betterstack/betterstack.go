// Package betterstack creates incidents and applies lifecycle actions using their native IDs.
package betterstack

import (
	"context"
	"encoding/json"
	"net/http"
	"net/mail"
	"net/url"

	"github.com/goforj/notification"
	"github.com/goforj/notification/internal/provider"
)

// Config fixes Better Stack authentication and explicit escalation choices.
// RequesterEmail may be omitted when the supplied token's permissions allow it.
// The provider controls escalation; configure a policy or the desired alert channels.
type Config struct {
	Token          string
	RequesterEmail string
	PolicyID       string
	TeamName       string
	Email          bool
	SMS            bool
	Call           bool
	Push           bool
	BaseURL        string
	HTTP           notification.HTTPConfig
}

// Driver borrows connection policy and does not invent correlation keys for native incident IDs.
type Driver struct {
	client  *provider.Client
	config  Config
	headers http.Header
}

// New validates configuration without creating an incident or provisioning an escalation policy.
func New(config Config) (*Driver, error) {
	base, err := provider.BaseURL(config.BaseURL, "https://incidents.betterstack.com")
	if err != nil || !provider.Token(config.Token) || (config.PolicyID != "" && !provider.Token(config.PolicyID)) || (config.TeamName != "" && !provider.Token(config.TeamName)) {
		return nil, notification.ErrInvalidRequest
	}
	if config.RequesterEmail != "" {
		address, err := mail.ParseAddress(config.RequesterEmail)
		if err != nil || address.Address != config.RequesterEmail {
			return nil, notification.ErrInvalidRequest
		}
	}
	config.BaseURL = base
	client, err := provider.New("betterstack", config.HTTP)
	if err != nil {
		return nil, err
	}
	return &Driver{client: client, config: config, headers: http.Header{"Authorization": {"Bearer " + config.Token}}}, nil
}

// Send creates a new incident. Use Create when subsequent actions need its returned ID.
func (d *Driver) Send(ctx context.Context, message notification.Message) error {
	_, err := d.Create(ctx, message)
	return err
}

// Create returns the native incident ID only after a correlated creation acknowledgement.
// This API does not offer deduplication; repeating Create can create another incident.
func (d *Driver) Create(ctx context.Context, message notification.Message) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := message.Validate(); err != nil {
		return "", err
	}
	summary := message.Title
	if summary == "" {
		summary = message.Body
	}
	request := provider.PostJSON(d.config.BaseURL+"/api/v3/incidents", d.headers, struct {
		Summary     string `json:"summary"`
		Name        string `json:"name,omitempty"`
		Description string `json:"description"`
		Requester   string `json:"requester_email,omitempty"`
		Policy      string `json:"policy_id,omitempty"`
		Team        string `json:"team_name,omitempty"`
		Email       bool   `json:"email"`
		SMS         bool   `json:"sms"`
		Call        bool   `json:"call"`
		Push        bool   `json:"push"`
	}{summary, message.Title, message.Body, d.config.RequesterEmail, d.config.PolicyID, d.config.TeamName, d.config.Email, d.config.SMS, d.config.Call, d.config.Push})
	id := ""
	err := d.client.Request(ctx, request.Method, request.URL, request.Header, request.Body, func(data []byte) error {
		var response struct {
			Data struct {
				ID   string `json:"id"`
				Type string `json:"type"`
			} `json:"data"`
		}
		if json.Unmarshal(data, &response) != nil || !provider.Token(response.Data.ID) || len(response.Data.ID) > 256 || response.Data.Type != "incident" {
			return notification.ErrOutcomeUnknown
		}
		id = response.Data.ID
		return nil
	}, http.StatusCreated)
	return id, err
}

// Acknowledge acts on a native ID returned by Create, not an application correlation key.
func (d *Driver) Acknowledge(ctx context.Context, id string) error {
	return d.action(ctx, id, "acknowledge")
}

// Resolve closes a native ID returned by Create without searching by mutable titles.
func (d *Driver) Resolve(ctx context.Context, id string) error { return d.action(ctx, id, "resolve") }

// action preserves native identity and refuses malformed or mismatched acknowledgements.
func (d *Driver) action(ctx context.Context, id, action string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if !provider.Token(id) || len(id) > 256 {
		return notification.ErrInvalidRequest
	}
	request := provider.PostJSON(d.config.BaseURL+"/api/v3/incidents/"+url.PathEscape(id)+"/"+action, d.headers, struct{}{})
	return d.client.Request(ctx, request.Method, request.URL, request.Header, request.Body, func(data []byte) error {
		var response struct {
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		if json.Unmarshal(data, &response) != nil || response.Data.ID != id {
			return notification.ErrOutcomeUnknown
		}
		return nil
	}, http.StatusOK)
}

// Close releases only owned idle connections; injected transports remain caller-owned.
func (d *Driver) Close() error { return d.client.Close() }
