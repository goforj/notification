// Package discord executes incoming webhooks with explicit acknowledgement and no mentions.
package discord

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"unicode/utf8"

	"github.com/goforj/notification"
	"github.com/goforj/notification/internal/provider"
)

// Config contains a trusted webhook and an optional existing thread snowflake.
type Config struct {
	URL      string
	ThreadID string
	HTTP     notification.HTTPConfig
}

// Driver owns one Discord incoming webhook.
type Driver struct{ hook *provider.Hook }

// New validates configuration and enables wait=true to obtain a message acknowledgement.
func New(config Config) (*Driver, error) {
	if err := provider.Endpoint(config.URL); err != nil {
		return nil, err
	}
	params := map[string]string{"wait": "true"}
	if config.ThreadID != "" {
		id, err := strconv.ParseUint(config.ThreadID, 10, 64)
		if err != nil || id == 0 {
			return nil, notification.ErrInvalidRequest
		}
		params["thread_id"] = config.ThreadID
	}
	endpoint := provider.Query(config.URL, params)
	hook, err := provider.NewHook("discord", endpoint, config.HTTP, http.Header{"Content-Type": []string{"application/json"}}, func(message notification.Message) ([]byte, error) {
		content := provider.Text(message)
		if utf8.RuneCountInString(content) > 2000 {
			return nil, notification.ErrRender
		}
		return provider.JSON(struct {
			Content         string `json:"content"`
			AllowedMentions struct {
				Parse []string `json:"parse"`
			} `json:"allowed_mentions"`
		}{content, struct {
			Parse []string `json:"parse"`
		}{[]string{}}}), nil
	}, func(data []byte) error {
		var ack struct {
			ID        string `json:"id"`
			ChannelID string `json:"channel_id"`
		}
		if json.Unmarshal(data, &ack) != nil || ack.ID == "" || ack.ChannelID == "" {
			return notification.ErrOutcomeUnknown
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &Driver{hook}, nil
}

// Close releases idle connections.
func (d *Driver) Close() error { return d.hook.Close() }

// Send rejects oversized text instead of truncating it or enabling mentions.
func (d *Driver) Send(ctx context.Context, message notification.Message) error {
	return d.hook.Send(ctx, message)
}
