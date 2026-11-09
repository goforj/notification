// Package mattermost sends plain text to Mattermost incoming webhooks.
package mattermost

import (
	"context"
	"github.com/goforj/notification"
	"github.com/goforj/notification/internal/provider"
	"strings"
)

// Config fixes a trusted mattermost destination and caller-owned connection policy.
type Config struct {
	URL  string
	HTTP notification.HTTPConfig
}

// Driver sends synchronously to the configured mattermost destination.
type Driver struct{ sender *provider.Sender }

// New validates configuration without contacting the provider.
func New(config Config) (*Driver, error) {
	if provider.Endpoint(config.URL) != nil {
		return nil, notification.ErrInvalidRequest
	}
	sender, err := provider.NewSender("mattermost", config.HTTP, func(ctx context.Context, message notification.Message) (provider.Request, error) {
		text := strings.NewReplacer("@", "＠").Replace(provider.Text(message))
		return provider.PostJSON(config.URL, nil, struct {
			Text string `json:"text"`
		}{text}), nil
	}, func(data []byte) error {
		if strings.TrimSpace(string(data)) != "ok" {
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
