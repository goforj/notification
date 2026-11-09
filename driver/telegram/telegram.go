// Package telegram sends plain text through the Telegram Bot API.
package telegram

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
	"time"
	"unicode/utf8"

	"github.com/goforj/notification"
	"github.com/goforj/notification/internal/provider"
)

// Config fixes a bot, chat and optional topic. BaseURL supports trusted Bot API servers.
type Config struct {
	Token          string
	ChatID         string
	ThreadID       int64
	Silent         bool
	ProtectContent bool
	BaseURL        string
	HTTP           notification.HTTPConfig
}

// Driver owns one immutable Telegram chat destination.
type Driver struct{ hook *provider.Hook }

var tokenPattern = regexp.MustCompile(`^[0-9]+:[A-Za-z0-9_-]+$`)

// New validates configuration without invoking getMe or contacting the bot.
func New(config Config) (*Driver, error) {
	base, err := provider.BaseURL(config.BaseURL, "https://api.telegram.org")
	if err != nil || !provider.Token(config.Token) || !tokenPattern.MatchString(config.Token) || !provider.Token(config.ChatID) || len(config.ChatID) > 128 || config.ThreadID < 0 {
		return nil, notification.ErrInvalidRequest
	}
	hook, err := provider.NewHook("telegram", base+"/bot"+url.PathEscape(config.Token)+"/sendMessage", config.HTTP, http.Header{"Content-Type": []string{"application/json"}}, func(message notification.Message) ([]byte, error) {
		text := provider.Text(message)
		if utf8.RuneCountInString(text) > 4096 {
			return nil, notification.ErrRender
		}
		return provider.JSON(struct {
			ChatID      string `json:"chat_id"`
			Text        string `json:"text"`
			ThreadID    int64  `json:"message_thread_id,omitempty"`
			Silent      bool   `json:"disable_notification"`
			Protect     bool   `json:"protect_content"`
			LinkPreview struct {
				Disabled bool `json:"is_disabled"`
			} `json:"link_preview_options"`
		}{config.ChatID, text, config.ThreadID, config.Silent, config.ProtectContent, struct {
			Disabled bool `json:"is_disabled"`
		}{true}}), nil
	}, func(data []byte) error {
		var ack struct {
			OK     *bool `json:"ok"`
			Result *struct {
				ID *int64 `json:"message_id"`
			} `json:"result"`
			ErrorCode  int `json:"error_code"`
			Parameters struct {
				RetryAfter int64 `json:"retry_after"`
			} `json:"parameters"`
		}
		if json.Unmarshal(data, &ack) != nil || ack.OK == nil {
			return notification.ErrOutcomeUnknown
		}
		if !*ack.OK {
			switch ack.ErrorCode {
			case 401, 403:
				return notification.ErrUnauthorized
			case 429:
				seconds := max(int64(0), min(ack.Parameters.RetryAfter, int64(86400)))
				return notification.NewDeliveryError("telegram", notification.Outcome{Status: notification.Failed, RetryAfter: time.Duration(seconds) * time.Second}, notification.ErrThrottled)
			default:
				return notification.ErrPermanent
			}
		}
		if ack.Result == nil || ack.Result.ID == nil || *ack.Result.ID <= 0 {
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

// Send uses plain text, disables link previews and rejects oversized messages.
func (d *Driver) Send(ctx context.Context, message notification.Message) error {
	return d.hook.Send(ctx, message)
}
