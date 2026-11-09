// Package vonage submits Unicode SMS through the native Vonage SMS API.
package vonage

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/goforj/notification"
	"github.com/goforj/notification/internal/provider"
)

// Config fixes a trusted vonage destination and caller-owned connection policy.
type Config struct {
	APIKey    string
	APISecret string
	From      string
	To        string
	BaseURL   string
	HTTP      notification.HTTPConfig
}

// Driver sends synchronously to the configured vonage destination.
type Driver struct{ sender *provider.Sender }

// New validates configuration without contacting the provider.
func New(config Config) (*Driver, error) {
	base, err := provider.BaseURL(config.BaseURL, "https://rest.nexmo.com")
	if err != nil || !provider.Token(config.APIKey) || !provider.Token(config.APISecret) || !provider.Token(config.From) || len(config.From) > 16 || !provider.Phone(config.To) {
		return nil, notification.ErrInvalidRequest
	}
	auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(config.APIKey+":"+config.APISecret))
	sender, err := provider.NewSender("vonage", config.HTTP, func(ctx context.Context, message notification.Message) (provider.Request, error) {
		text, err := provider.SMSText(message)
		if err != nil {
			return provider.Request{}, err
		}
		body := url.Values{"from": {config.From}, "to": {strings.TrimPrefix(config.To, "+")}, "text": {text}, "type": {"unicode"}}
		return provider.Request{Method: http.MethodPost, URL: base + "/sms/json", Header: http.Header{"Authorization": {auth}, "Content-Type": {"application/x-www-form-urlencoded"}}, Body: []byte(body.Encode())}, nil
	}, func(data []byte) error {
		var ack struct {
			Messages []struct {
				ID     string  `json:"message-id"`
				Status *string `json:"status"`
			} `json:"messages"`
		}
		if json.Unmarshal(data, &ack) != nil || len(ack.Messages) != 1 || ack.Messages[0].Status == nil {
			return notification.ErrOutcomeUnknown
		}
		switch *ack.Messages[0].Status {
		case "0":
			if ack.Messages[0].ID == "" {
				return notification.ErrOutcomeUnknown
			}
			return nil
		case "1":
			return notification.ErrThrottled
		case "4":
			return notification.ErrUnauthorized
		case "5":
			return notification.ErrRetryable
		default:
			return notification.ErrPermanent
		}
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
