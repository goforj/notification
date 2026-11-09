// Package zulip posts plain notification content to a configured Zulip channel and topic.
package zulip

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/mail"
	"net/url"

	"github.com/goforj/notification"
	"github.com/goforj/notification/internal/provider"
)

// Config fixes a trusted zulip destination and caller-owned connection policy.
type Config struct {
	URL     string
	Email   string
	APIKey  string
	Channel string
	Topic   string
	HTTP    notification.HTTPConfig
}

// Driver sends synchronously to the configured zulip destination.
type Driver struct{ sender *provider.Sender }

// New validates configuration without contacting the provider.
func New(config Config) (*Driver, error) {
	base, err := provider.BaseURL(config.URL, "")
	email, emailErr := mail.ParseAddress(config.Email)
	if err != nil || emailErr != nil || email.Address != config.Email || !provider.Token(config.APIKey) || !provider.Token(config.Channel) || !provider.Token(config.Topic) {
		return nil, notification.ErrInvalidRequest
	}
	auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(config.Email+":"+config.APIKey))
	sender, err := provider.NewSender("zulip", config.HTTP, func(ctx context.Context, message notification.Message) (provider.Request, error) {
		body := url.Values{"type": {"stream"}, "to": {config.Channel}, "topic": {config.Topic}, "content": {provider.Text(message)}}
		return provider.Request{Method: http.MethodPost, URL: base + "/api/v1/messages", Header: http.Header{"Authorization": {auth}, "Content-Type": {"application/x-www-form-urlencoded"}}, Body: []byte(body.Encode())}, nil
	}, func(data []byte) error {
		var ack struct {
			Result string `json:"result"`
			ID     *int64 `json:"id"`
		}
		if json.Unmarshal(data, &ack) != nil {
			return notification.ErrOutcomeUnknown
		}
		if ack.Result == "error" {
			return notification.ErrPermanent
		}
		if ack.Result != "success" || ack.ID == nil || *ack.ID <= 0 {
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
