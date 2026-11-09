// Package onesignal submits one explicitly targeted native OneSignal push notification.
package onesignal

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/goforj/notification"
	"github.com/goforj/notification/internal/provider"
)

// Config fixes a trusted onesignal destination and caller-owned connection policy.
type Config struct {
	AppID          string
	Token          string
	SubscriptionID string
	ExternalID     string
	BaseURL        string
	HTTP           notification.HTTPConfig
}

// Driver sends synchronously to the configured onesignal destination.
type Driver struct{ sender *provider.Sender }

// New validates configuration without contacting the provider.
func New(config Config) (*Driver, error) {
	base, err := provider.BaseURL(config.BaseURL, "https://api.onesignal.com")
	if err != nil || !provider.Token(config.AppID) || !provider.Token(config.Token) || (config.SubscriptionID == "") == (config.ExternalID == "") || (config.SubscriptionID != "" && !provider.Token(config.SubscriptionID)) || (config.ExternalID != "" && !provider.Token(config.ExternalID)) {
		return nil, notification.ErrInvalidRequest
	}
	sender, err := provider.NewSender("onesignal", config.HTTP, func(ctx context.Context, message notification.Message) (provider.Request, error) {
		var subscriptions []string
		var aliases map[string][]string
		if config.SubscriptionID != "" {
			subscriptions = []string{config.SubscriptionID}
		} else {
			aliases = map[string][]string{"external_id": {config.ExternalID}}
		}
		var headings map[string]string
		if message.Title != "" {
			headings = map[string]string{"en": message.Title}
		}
		request := provider.PostJSON(base+"/notifications", http.Header{"Authorization": {"Key " + config.Token}}, struct {
			AppID         string              `json:"app_id"`
			Channel       string              `json:"target_channel"`
			Subscriptions []string            `json:"include_subscription_ids,omitempty"`
			Aliases       map[string][]string `json:"include_aliases,omitempty"`
			Headings      map[string]string   `json:"headings,omitempty"`
			Contents      map[string]string   `json:"contents"`
		}{config.AppID, "push", subscriptions, aliases, headings, map[string]string{"en": message.Body}})
		if len(request.Body) > 4096 {
			return provider.Request{}, notification.ErrRender
		}
		return request, nil
	}, func(data []byte) error {
		var ack struct {
			ID         string          `json:"id"`
			Recipients *int            `json:"recipients"`
			Errors     json.RawMessage `json:"errors"`
		}
		if json.Unmarshal(data, &ack) != nil || ack.ID == "" {
			return notification.ErrOutcomeUnknown
		}
		if ack.Recipients != nil && *ack.Recipients <= 0 {
			return notification.ErrPermanent
		}
		if len(ack.Errors) > 0 && string(ack.Errors) != "null" && string(ack.Errors) != "[]" && string(ack.Errors) != "{}" {
			return notification.ErrOutcomeUnknown
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &Driver{sender: sender}, nil
}

// Send validates a message and performs one native operation without retries.
func (d *Driver) Send(ctx context.Context, message notification.Message) error {
	return d.sender.Send(ctx, message)
}

// Close releases only owned idle connections; injected transports remain caller-owned.
func (d *Driver) Close() error { return d.sender.Close() }
