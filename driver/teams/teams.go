// Package teams posts Adaptive Cards to configured Microsoft Teams Workflows webhooks.
package teams

import (
	"context"
	"net/http"

	"github.com/goforj/notification"
	"github.com/goforj/notification/internal/provider"
)

// Config supplies a trusted Workflows URL, rather than a retired Office 365 connector.
type Config struct {
	URL  string
	HTTP notification.HTTPConfig
}

// Driver owns one Teams Workflows sender.
type Driver struct{ hook *provider.Hook }

// textBlock is the bounded Adaptive Card plain-message projection.
type textBlock struct {
	Type   string `json:"type"`
	Text   string `json:"text"`
	Wrap   bool   `json:"wrap"`
	Weight string `json:"weight,omitempty"`
}

// New validates configuration without contacting Microsoft.
func New(config Config) (*Driver, error) {
	hook, err := provider.NewHook("teams", config.URL, config.HTTP, http.Header{"Content-Type": []string{"application/json"}}, func(message notification.Message) ([]byte, error) {
		blocks := []textBlock{}
		if message.Title != "" {
			blocks = append(blocks, textBlock{"TextBlock", message.Title, true, "Bolder"})
		}
		blocks = append(blocks, textBlock{"TextBlock", message.Body, true, ""})
		card := struct {
			Type    string      `json:"type"`
			Version string      `json:"version"`
			Body    []textBlock `json:"body"`
		}{"AdaptiveCard", "1.2", blocks}
		return provider.JSON(struct {
			Type        string `json:"type"`
			Attachments []struct {
				ContentType string `json:"contentType"`
				Content     any    `json:"content"`
			} `json:"attachments"`
		}{"message", []struct {
			ContentType string `json:"contentType"`
			Content     any    `json:"content"`
		}{{"application/vnd.microsoft.card.adaptive", card}}}), nil
	}, nil)
	if err != nil {
		return nil, err
	}
	return &Driver{hook}, nil
}

// Close releases idle connections.
func (d *Driver) Close() error { return d.hook.Close() }

// Send reports HTTP acceptance by the configured workflow, not Teams recipient delivery.
func (d *Driver) Send(ctx context.Context, message notification.Message) error {
	return d.hook.Send(ctx, message)
}
