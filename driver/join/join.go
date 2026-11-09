// Package join sends title and text through the Join device push API.
package join

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/goforj/notification"
	"github.com/goforj/notification/internal/provider"
)

// Config fixes a trusted join destination and caller-owned connection policy.
type Config struct {
	APIKey   string
	DeviceID string
	BaseURL  string
	HTTP     notification.HTTPConfig
}

// Driver sends synchronously to the configured join destination.
type Driver struct{ sender *provider.Sender }

// New validates configuration without contacting the provider.
func New(config Config) (*Driver, error) {
	base, err := provider.BaseURL(config.BaseURL, "https://joinjoaomgcd.appspot.com")
	if err != nil || !provider.Token(config.APIKey) || !provider.Token(config.DeviceID) {
		return nil, notification.ErrInvalidRequest
	}
	sender, err := provider.NewSender("join", config.HTTP, func(ctx context.Context, message notification.Message) (provider.Request, error) {
		endpoint := provider.Query(base+"/_ah/api/messaging/v1/sendPush", map[string]string{"apikey": config.APIKey, "deviceId": config.DeviceID, "title": message.Title, "text": message.Body})
		if len(endpoint) > 4096 {
			return provider.Request{}, notification.ErrRender
		}
		return provider.Request{Method: http.MethodGet, URL: endpoint, Header: http.Header{}}, nil
	}, func(data []byte) error {
		var ack struct {
			Success *bool `json:"success"`
		}
		if json.Unmarshal(data, &ack) != nil || ack.Success == nil {
			return notification.ErrOutcomeUnknown
		}
		if !*ack.Success {
			return notification.ErrPermanent
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
