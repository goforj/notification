// Package ifttt triggers a configured IFTTT Webhooks event with title and body values.
package ifttt

import (
	"context"
	"net/url"
	"strings"

	"github.com/goforj/notification"
	"github.com/goforj/notification/internal/provider"
)

// Config fixes a trusted ifttt destination and caller-owned connection policy.
type Config struct {
	Key     string
	Event   string
	BaseURL string
	HTTP    notification.HTTPConfig
}

// Driver sends synchronously to the configured ifttt destination.
type Driver struct{ sender *provider.Sender }

// New validates configuration without contacting the provider.
func New(config Config) (*Driver, error) {
	base, err := provider.BaseURL(config.BaseURL, "https://maker.ifttt.com")
	if err != nil || !provider.Token(config.Key) || !provider.Token(config.Event) || len(config.Event) > 128 {
		return nil, notification.ErrInvalidRequest
	}
	sender, err := provider.NewSender("ifttt", config.HTTP, func(ctx context.Context, message notification.Message) (provider.Request, error) {
		return provider.PostJSON(base+"/trigger/"+url.PathEscape(config.Event)+"/with/key/"+url.PathEscape(config.Key), nil, struct {
			Title string `json:"value1"`
			Body  string `json:"value2"`
		}{message.Title, message.Body}), nil
	}, func(data []byte) error {
		if !strings.HasPrefix(string(data), "Congratulations!") {
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
