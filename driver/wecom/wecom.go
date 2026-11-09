// Package wecom submits plain text to a WeCom group robot webhook.
package wecom

import (
	"context"
	"github.com/goforj/notification"
	"github.com/goforj/notification/internal/provider"
	"net/http"
)

// Config supplies a trusted group robot webhook URL, including its key.
type Config struct {
	URL  string
	HTTP notification.HTTPConfig
}

// Driver owns one WeCom robot transport.
type Driver struct{ hook *provider.Hook }

// New constructs a robot without sending a test notification.
func New(config Config) (*Driver, error) {
	hook, err := provider.NewHook("wecom", config.URL, config.HTTP, http.Header{"Content-Type": {"application/json"}}, func(message notification.Message) ([]byte, error) {
		text := provider.Text(message)
		if len(text) > 2048 {
			return nil, notification.ErrRender
		}
		return provider.JSON(struct {
			Type string `json:"msgtype"`
			Text struct {
				Content string `json:"content"`
			} `json:"text"`
		}{"text", struct {
			Content string `json:"content"`
		}{text}}), nil
	}, provider.RobotAck("errcode"))
	if err != nil {
		return nil, err
	}
	return &Driver{hook}, nil
}

// Close releases owned idle HTTP connections.
func (d *Driver) Close() error { return d.hook.Close() }

// Send submits text without generating mentions or markdown.
func (d *Driver) Send(ctx context.Context, message notification.Message) error {
	return d.hook.Send(ctx, message)
}
