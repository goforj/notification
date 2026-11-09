// Package signal sends through an application-owned paired signal-cli REST relay.
package signal

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/goforj/notification"
	"github.com/goforj/notification/internal/provider"
)

// Config fixes a trusted signal destination and caller-owned connection policy.
type Config struct {
	URL string
	// Number is the relay's already registered E.164 sender.
	Number string
	// Recipient is an E.164 number or a group. identifier understood by the relay.
	Recipient string
	// Token optionally authenticates a trusted reverse proxy in front of the relay.
	Token string
	HTTP  notification.HTTPConfig
}

// Driver sends synchronously to the configured signal destination.
type Driver struct{ sender *provider.Sender }

// New validates configuration without contacting the provider.
func New(config Config) (*Driver, error) {
	base, err := provider.BaseURL(config.URL, "")
	if err != nil || !provider.Phone(config.Number) || !provider.Token(config.Recipient) || (!provider.Phone(config.Recipient) && !strings.HasPrefix(config.Recipient, "group.")) || (config.Token != "" && !provider.Token(config.Token)) {
		return nil, notification.ErrInvalidRequest
	}
	headers := http.Header{}
	if config.Token != "" {
		headers.Set("Authorization", "Bearer "+config.Token)
	}
	sender, err := provider.NewSender("signal", config.HTTP, func(ctx context.Context, message notification.Message) (provider.Request, error) {
		return provider.PostJSON(base+"/v2/send", headers, struct {
			Number     string   `json:"number"`
			Recipients []string `json:"recipients"`
			Message    string   `json:"message"`
			Mode       string   `json:"text_mode"`
		}{config.Number, []string{config.Recipient}, provider.Text(message), "normal"}), nil
	}, func(data []byte) error {
		var ack struct {
			Timestamp json.Number `json:"timestamp"`
		}
		if json.Unmarshal(data, &ack) != nil {
			return notification.ErrOutcomeUnknown
		}
		timestamp, err := strconv.ParseInt(ack.Timestamp.String(), 10, 64)
		if err != nil || timestamp <= 0 {
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
