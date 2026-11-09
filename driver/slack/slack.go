// Package slack sends plain text through Slack incoming webhooks.
package slack

import (
	"context"
	"net/http"
	"strings"

	"github.com/goforj/notification"
	"github.com/goforj/notification/internal/provider"
)

// Config contains a trusted Slack incoming-webhook URL and transport policy.
type Config struct {
	URL  string
	HTTP notification.HTTPConfig
}

// Driver owns a Slack webhook sender without requiring the advanced dispatcher.
type Driver struct{ hook *provider.Hook }

// New validates configuration without contacting Slack.
func New(config Config) (*Driver, error) {
	hook, err := provider.NewHook("slack", config.URL, config.HTTP, http.Header{"Content-Type": []string{"application/json"}}, func(message notification.Message) ([]byte, error) {
		return provider.JSON(struct {
			Text   string `json:"text"`
			Mrkdwn bool   `json:"mrkdwn"`
		}{provider.Text(message), false}), nil
	}, func(data []byte) error {
		if strings.TrimSpace(string(data)) != "ok" {
			return notification.ErrOutcomeUnknown
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &Driver{hook}, nil
}

// Close releases idle connections.
func (d *Driver) Close() error { return d.hook.Close() }

// Send requires Slack's documented ok acknowledgement.
func (d *Driver) Send(ctx context.Context, message notification.Message) error {
	return d.hook.Send(ctx, message)
}
