// Package telnyx submits a single native Telnyx SMS with recipient correlation.
package telnyx

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/goforj/notification"
	"github.com/goforj/notification/internal/provider"
)

// Config fixes a trusted telnyx destination and caller-owned connection policy.
type Config struct {
	Token              string
	From               string
	To                 string
	MessagingProfileID string
	BaseURL            string
	HTTP               notification.HTTPConfig
}

// Driver sends synchronously to the configured telnyx destination.
type Driver struct{ sender *provider.Sender }

// New validates configuration without contacting the provider.
func New(config Config) (*Driver, error) {
	base, err := provider.BaseURL(config.BaseURL, "https://api.telnyx.com")
	if err != nil || !provider.Token(config.Token) || !provider.Phone(config.From) || !provider.Phone(config.To) || (config.MessagingProfileID != "" && !provider.Token(config.MessagingProfileID)) {
		return nil, notification.ErrInvalidRequest
	}
	sender, err := provider.NewSender("telnyx", config.HTTP, func(ctx context.Context, message notification.Message) (provider.Request, error) {
		text, err := provider.SMSText(message)
		if err != nil {
			return provider.Request{}, err
		}
		return provider.PostJSON(base+"/v2/messages", http.Header{"Authorization": {"Bearer " + config.Token}}, struct {
			From    string `json:"from"`
			To      string `json:"to"`
			Text    string `json:"text"`
			Profile string `json:"messaging_profile_id,omitempty"`
		}{config.From, config.To, text, config.MessagingProfileID}), nil
	}, func(data []byte) error {
		var ack struct {
			Data struct {
				ID string `json:"id"`
				To []struct {
					Phone  string `json:"phone_number"`
					Status string `json:"status"`
				} `json:"to"`
			} `json:"data"`
		}
		if json.Unmarshal(data, &ack) != nil || ack.Data.ID == "" || len(ack.Data.To) != 1 || ack.Data.To[0].Phone != config.To {
			return notification.ErrOutcomeUnknown
		}
		switch ack.Data.To[0].Status {
		case "queued", "sending", "sent", "delivered":
			return nil
		case "failed", "delivery_failed":
			return notification.ErrPermanent
		default:
			return notification.ErrOutcomeUnknown
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
