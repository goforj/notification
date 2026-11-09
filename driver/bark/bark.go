// Package bark publishes native notifications to a configured Bark server.
package bark

import (
	"context"
	"encoding/json"

	"github.com/goforj/notification"
	"github.com/goforj/notification/internal/provider"
)

// Config fixes a trusted bark destination and caller-owned connection policy.
type Config struct {
	URL       string
	DeviceKey string
	Group     string
	HTTP      notification.HTTPConfig
}

// Driver sends synchronously to the configured bark destination.
type Driver struct{ sender *provider.Sender }

// New validates configuration without contacting the provider.
func New(config Config) (*Driver, error) {
	base, err := provider.BaseURL(config.URL, "https://api.day.app")
	if err != nil || !provider.Token(config.DeviceKey) || len(config.DeviceKey) > 256 || (config.Group != "" && !provider.Token(config.Group)) {
		return nil, notification.ErrInvalidRequest
	}
	sender, err := provider.NewSender("bark", config.HTTP, func(ctx context.Context, message notification.Message) (provider.Request, error) {
		return provider.PostJSON(base+"/push", nil, struct {
			DeviceKey string `json:"device_key"`
			Title     string `json:"title"`
			Body      string `json:"body"`
			Group     string `json:"group,omitempty"`
		}{config.DeviceKey, message.Title, message.Body, config.Group}), nil
	}, func(data []byte) error {
		var ack struct {
			Code *int `json:"code"`
		}
		if json.Unmarshal(data, &ack) != nil || ack.Code == nil {
			return notification.ErrOutcomeUnknown
		}
		if *ack.Code != 200 {
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
