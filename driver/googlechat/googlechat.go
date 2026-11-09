// Package googlechat sends native Google Chat incoming-webhook messages.
package googlechat

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/goforj/notification"
	"github.com/goforj/notification/internal/provider"
)

// Config supplies a trusted webhook URL and HTTP policy.
type Config struct {
	URL  string
	HTTP notification.HTTPConfig
}

// Driver owns one Google Chat webhook.
type Driver struct{ hook *provider.Hook }

// New validates the endpoint without contacting Google Workspace.
func New(config Config) (*Driver, error) {
	hook, err := provider.NewHook("googlechat", config.URL, config.HTTP, http.Header{"Content-Type": []string{"application/json"}}, func(message notification.Message) ([]byte, error) {
		return provider.JSON(struct {
			Text string `json:"text"`
		}{provider.Text(message)}), nil
	}, func(data []byte) error {
		var ack struct {
			Name string `json:"name"`
		}
		if json.Unmarshal(data, &ack) != nil || !strings.HasPrefix(ack.Name, "spaces/") || !strings.Contains(ack.Name, "/messages/") {
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

// Send requires a native Chat message resource name in the acknowledgement.
func (d *Driver) Send(ctx context.Context, message notification.Message) error {
	return d.hook.Send(ctx, message)
}
