// Package apprise submits plain notifications to a configured stateful Apprise API relay.
package apprise

import (
	"context"
	"encoding/base64"
	"net/http"

	"github.com/goforj/notification"
	"github.com/goforj/notification/internal/provider"
)

// Config fixes a trusted apprise destination and caller-owned connection policy.
type Config struct {
	URL string
	// Key selects an existing server-owned configuration; downstream credentials never enter Message.
	Key      string
	Tag      string
	Username string
	Password string
	HTTP     notification.HTTPConfig
}

// Driver sends synchronously to the configured apprise destination.
type Driver struct{ sender *provider.Sender }

// New validates configuration without contacting the provider.
func New(config Config) (*Driver, error) {
	base, err := provider.BaseURL(config.URL, "")
	if err != nil || !provider.Token(config.Key) || (config.Tag != "" && !provider.Token(config.Tag)) || (config.Username != "" && !provider.Token(config.Username)) || (config.Password != "" && !provider.Token(config.Password)) || (config.Username != "" && config.Password == "") {
		return nil, notification.ErrInvalidRequest
	}
	headers := http.Header{"X-Apprise-Config-Id": {config.Key}}
	if config.Password != "" {
		headers.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(config.Username+":"+config.Password)))
	}
	sender, err := provider.NewSender("apprise", config.HTTP, func(ctx context.Context, message notification.Message) (provider.Request, error) {
		return provider.PostJSON(base+"/notify", headers, struct {
			Title  string `json:"title"`
			Body   string `json:"body"`
			Format string `json:"format"`
			Tag    string `json:"tag,omitempty"`
		}{message.Title, message.Body, "text", config.Tag}), nil
	}, nil, http.StatusOK)
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
