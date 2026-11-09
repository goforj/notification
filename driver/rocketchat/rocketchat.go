// Package rocketchat sends plain text through Rocket.Chat incoming integrations.
package rocketchat

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/goforj/notification"
	"github.com/goforj/notification/internal/provider"
)

// Config fixes a trusted rocketchat destination and caller-owned connection policy.
type Config struct {
	URL  string
	HTTP notification.HTTPConfig
}

// Driver sends synchronously to the configured rocketchat destination.
type Driver struct{ sender *provider.Sender }

// New validates configuration without contacting the provider.
func New(config Config) (*Driver, error) {
	if provider.Endpoint(config.URL) != nil {
		return nil, notification.ErrInvalidRequest
	}
	sender, err := provider.NewSender("rocketchat", config.HTTP, func(ctx context.Context, message notification.Message) (provider.Request, error) {
		text := strings.ReplaceAll(provider.Text(message), "@", "＠")
		return provider.PostJSON(config.URL, nil, struct {
			Text string `json:"text"`
		}{text}), nil
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
