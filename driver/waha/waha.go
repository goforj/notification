// Package waha sends text through a configured WAHA session.
package waha

import (
	"context"
	"github.com/goforj/notification"
	"github.com/goforj/notification/internal/provider"
	"net/http"
)

// Config supplies an existing authenticated session and chat destination.
type Config struct {
	URL     string
	APIKey  string
	Session string
	ChatID  string
	HTTP    notification.HTTPConfig
}

// Driver owns a WAHA transport, not the WhatsApp session lifecycle.
type Driver struct{ hook *provider.Hook }

// New validates configuration without starting, pairing or reconnecting a session.
func New(config Config) (*Driver, error) {
	if !provider.Token(config.APIKey) || !provider.Token(config.Session) || len(config.Session) > 128 || !provider.Token(config.ChatID) || len(config.ChatID) > 256 {
		return nil, notification.ErrInvalidRequest
	}
	base, err := provider.BaseURL(config.URL, "")
	if err != nil {
		return nil, err
	}
	hook, err := provider.NewHook("waha", base+"/api/sendText", config.HTTP, http.Header{"Content-Type": {"application/json"}, "X-Api-Key": {config.APIKey}}, func(message notification.Message) ([]byte, error) {
		return provider.JSON(struct {
			Session string `json:"session"`
			ChatID  string `json:"chatId"`
			Text    string `json:"text"`
			Preview bool   `json:"linkPreview"`
		}{config.Session, config.ChatID, provider.Text(message), false}), nil
	}, nil)
	if err != nil {
		return nil, err
	}
	return &Driver{hook}, nil
}

// Close releases idle connections without logging out the configured session.
func (d *Driver) Close() error { return d.hook.Close() }

// Send treats HTTP success as API acceptance across WAHA's different engine schemas.
func (d *Driver) Send(ctx context.Context, message notification.Message) error {
	return d.hook.Send(ctx, message)
}
