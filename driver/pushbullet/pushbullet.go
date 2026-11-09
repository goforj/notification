// Package pushbullet submits native note pushes to Pushbullet.
package pushbullet

import (
	"context"
	"encoding/json"
	"github.com/goforj/notification"
	"github.com/goforj/notification/internal/provider"
	"net/http"
)

// Config fixes a trusted pushbullet destination and caller-owned connection policy.
type Config struct {
	Token    string
	DeviceID string
	Email    string
	Channel  string
	BaseURL  string
	HTTP     notification.HTTPConfig
}

// Driver sends synchronously to the configured pushbullet destination.
type Driver struct{ sender *provider.Sender }

// New validates configuration without contacting the provider.
func New(config Config) (*Driver, error) {
	base, err := provider.BaseURL(config.BaseURL, "https://api.pushbullet.com")
	targets := 0
	for _, value := range []string{config.DeviceID, config.Email, config.Channel} {
		if value != "" {
			targets++
			if !provider.Token(value) {
				return nil, notification.ErrInvalidRequest
			}
		}
	}
	if err != nil || !provider.Token(config.Token) || targets > 1 {
		return nil, notification.ErrInvalidRequest
	}
	sender, err := provider.NewSender("pushbullet", config.HTTP, func(ctx context.Context, message notification.Message) (provider.Request, error) {
		return provider.PostJSON(base+"/v2/pushes", http.Header{"Access-Token": {config.Token}}, struct {
			Type    string `json:"type"`
			Title   string `json:"title"`
			Body    string `json:"body"`
			Device  string `json:"device_iden,omitempty"`
			Email   string `json:"email,omitempty"`
			Channel string `json:"channel_tag,omitempty"`
		}{"note", message.Title, message.Body, config.DeviceID, config.Email, config.Channel}), nil
	}, func(data []byte) error {
		var ack struct {
			ID string `json:"iden"`
		}
		if json.Unmarshal(data, &ack) != nil || ack.ID == "" {
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
