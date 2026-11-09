// Package homeassistant invokes a configured Home Assistant notify service.
package homeassistant

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"

	"github.com/goforj/notification"
	"github.com/goforj/notification/internal/provider"
)

// Config fixes a trusted homeassistant destination and caller-owned connection policy.
type Config struct {
	URL     string
	Token   string
	Service string
	HTTP    notification.HTTPConfig
}

// Driver sends synchronously to the configured homeassistant destination.
type Driver struct{ sender *provider.Sender }

// New validates configuration without contacting the provider.
func New(config Config) (*Driver, error) {
	base, err := provider.BaseURL(config.URL, "")
	if config.Service == "" {
		config.Service = "notify"
	}
	valid, _ := regexp.MatchString(`^[a-z0-9_]{1,128}$`, config.Service)
	if err != nil || !provider.Token(config.Token) || !valid {
		return nil, notification.ErrInvalidRequest
	}
	sender, err := provider.NewSender("homeassistant", config.HTTP, func(ctx context.Context, message notification.Message) (provider.Request, error) {
		return provider.PostJSON(base+"/api/services/notify/"+config.Service, http.Header{"Authorization": {"Bearer " + config.Token}}, struct {
			Title   string `json:"title"`
			Message string `json:"message"`
		}{message.Title, message.Body}), nil
	}, func(data []byte) error {
		var ack []json.RawMessage
		if json.Unmarshal(data, &ack) != nil || ack == nil {
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
