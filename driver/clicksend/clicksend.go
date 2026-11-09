// Package clicksend submits a single SMS and checks its native per-message result.
package clicksend

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"

	"github.com/goforj/notification"
	"github.com/goforj/notification/internal/provider"
)

// Config fixes a trusted clicksend destination and caller-owned connection policy.
type Config struct {
	Username string
	APIKey   string
	From     string
	To       string
	BaseURL  string
	HTTP     notification.HTTPConfig
}

// Driver sends synchronously to the configured clicksend destination.
type Driver struct{ sender *provider.Sender }

// New validates configuration without contacting the provider.
func New(config Config) (*Driver, error) {
	base, err := provider.BaseURL(config.BaseURL, "https://rest.clicksend.com")
	if err != nil || !provider.Token(config.Username) || !provider.Token(config.APIKey) || !provider.Phone(config.To) || (config.From != "" && !provider.Token(config.From)) {
		return nil, notification.ErrInvalidRequest
	}
	auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(config.Username+":"+config.APIKey))
	sender, err := provider.NewSender("clicksend", config.HTTP, func(ctx context.Context, message notification.Message) (provider.Request, error) {
		text, err := provider.SMSText(message)
		if err != nil {
			return provider.Request{}, err
		}
		return provider.PostJSON(base+"/v3/sms/send", http.Header{"Authorization": {auth}}, struct {
			Messages []struct {
				To     string `json:"to"`
				From   string `json:"from,omitempty"`
				Body   string `json:"body"`
				Source string `json:"source"`
			} `json:"messages"`
		}{[]struct {
			To     string `json:"to"`
			From   string `json:"from,omitempty"`
			Body   string `json:"body"`
			Source string `json:"source"`
		}{{config.To, config.From, text, "go"}}}), nil
	}, func(data []byte) error {
		var ack struct {
			Code string `json:"response_code"`
			Data struct {
				Messages []struct {
					ID     string `json:"message_id"`
					Status string `json:"status"`
				} `json:"messages"`
			} `json:"data"`
		}
		if json.Unmarshal(data, &ack) != nil || ack.Code == "" {
			return notification.ErrOutcomeUnknown
		}
		if ack.Code != "SUCCESS" {
			return notification.ErrPermanent
		}
		if len(ack.Data.Messages) != 1 {
			return notification.ErrOutcomeUnknown
		}
		message := ack.Data.Messages[0]
		if message.Status != "SUCCESS" {
			return notification.ErrPermanent
		}
		if message.ID == "" {
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
