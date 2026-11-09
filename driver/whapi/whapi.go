// Package whapi sends plain text through Whapi.Cloud's messages API.
package whapi

import (
	"context"
	"encoding/json"
	"github.com/goforj/notification"
	"github.com/goforj/notification/internal/provider"
	"net/http"
	"regexp"
)

// Config supplies a channel token and recipient phone number or chat ID.
type Config struct {
	Token   string
	To      string
	BaseURL string
	HTTP    notification.HTTPConfig
}

// Driver owns one immutable Whapi channel destination.
type Driver struct{ hook *provider.Hook }

var recipientPattern = regexp.MustCompile(`^[0-9-]{9,31}(@[A-Za-z0-9_.]+)?$`)

// New validates channel configuration without querying or modifying it.
func New(config Config) (*Driver, error) {
	if !provider.Token(config.Token) || !recipientPattern.MatchString(config.To) {
		return nil, notification.ErrInvalidRequest
	}
	base, err := provider.BaseURL(config.BaseURL, "https://gate.whapi.cloud")
	if err != nil {
		return nil, err
	}
	hook, err := provider.NewHook("whapi", base+"/messages/text", config.HTTP, http.Header{"Content-Type": {"application/json"}, "Authorization": {"Bearer " + config.Token}}, func(message notification.Message) ([]byte, error) {
		return provider.JSON(struct {
			To      string `json:"to"`
			Body    string `json:"body"`
			Preview bool   `json:"no_link_preview"`
		}{config.To, provider.Text(message), true}), nil
	}, func(data []byte) error {
		var ack struct {
			Sent *bool `json:"sent"`
		}
		if json.Unmarshal(data, &ack) != nil || ack.Sent == nil {
			return notification.ErrOutcomeUnknown
		}
		if !*ack.Sent {
			return notification.ErrPermanent
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &Driver{hook}, nil
}

// Close releases only owned idle connections; injected transports remain caller-owned.
func (d *Driver) Close() error { return d.hook.Close() }

// Send submits one text request with link previews disabled.
func (d *Driver) Send(ctx context.Context, message notification.Message) error {
	return d.hook.Send(ctx, message)
}
