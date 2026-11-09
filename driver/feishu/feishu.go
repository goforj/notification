// Package feishu sends plain text to Feishu or Lark v2 custom bot webhooks.
package feishu

import (
	"context"
	"github.com/goforj/notification"
	"github.com/goforj/notification/internal/provider"
	"net/http"
	"strconv"
	"time"
)

// Config supplies a trusted webhook and its optional signature-verification secret.
type Config struct {
	URL    string
	Secret string
	HTTP   notification.HTTPConfig
}

// Driver owns one custom bot webhook transport.
type Driver struct{ hook *provider.Hook }

// New validates configuration without contacting the bot.
func New(config Config) (*Driver, error) {
	if config.Secret != "" && !provider.Token(config.Secret) {
		return nil, notification.ErrInvalidRequest
	}
	hook, err := provider.NewHook("feishu", config.URL, config.HTTP, http.Header{"Content-Type": {"application/json"}}, func(message notification.Message) ([]byte, error) {
		var timestamp, sign string
		if config.Secret != "" {
			timestamp = strconv.FormatInt(time.Now().Unix(), 10)
			sign = provider.Signature(timestamp+"\n"+config.Secret, "")
		}
		body := provider.JSON(struct {
			Type    string `json:"msg_type"`
			Content struct {
				Text string `json:"text"`
			} `json:"content"`
			Timestamp string `json:"timestamp,omitempty"`
			Sign      string `json:"sign,omitempty"`
		}{"text", struct {
			Text string `json:"text"`
		}{provider.Text(message)}, timestamp, sign})
		if len(body) > 20*1024 {
			return nil, notification.ErrRender
		}
		return body, nil
	}, provider.RobotAck("code"))
	if err != nil {
		return nil, err
	}
	return &Driver{hook}, nil
}

// Close releases owned idle HTTP connections.
func (d *Driver) Close() error { return d.hook.Close() }

// Send signs each request with its current Unix timestamp when configured.
func (d *Driver) Send(ctx context.Context, message notification.Message) error {
	return d.hook.Send(ctx, message)
}
