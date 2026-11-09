// Package line sends plain text through the LINE Messaging API, replacing LINE Notify.
package line

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"unicode/utf16"

	"github.com/goforj/notification"
	"github.com/goforj/notification/internal/provider"
)

// Config supplies a channel access token and a user, group or room recipient.
type Config struct {
	Token   string
	To      string
	BaseURL string
	Silent  bool
	HTTP    notification.HTTPConfig
}

// Driver owns one configured Messaging API recipient.
type Driver struct{ hook *provider.Hook }

var recipientPattern = regexp.MustCompile(`^[UCR][0-9a-fA-F]{32}$`)

// New validates credentials without contacting a recipient or modifying the channel.
func New(config Config) (*Driver, error) {
	if !provider.Token(config.Token) || !recipientPattern.MatchString(config.To) {
		return nil, notification.ErrInvalidRequest
	}
	base, err := provider.BaseURL(config.BaseURL, "https://api.line.me")
	if err != nil {
		return nil, err
	}
	hook, err := provider.NewHook("line", base+"/v2/bot/message/push", config.HTTP, http.Header{"Content-Type": {"application/json"}, "Authorization": {"Bearer " + config.Token}}, func(message notification.Message) ([]byte, error) {
		text := provider.Text(message)
		if len(utf16.Encode([]rune(text))) > 5000 {
			return nil, notification.ErrRender
		}
		return provider.JSON(struct {
			To       string `json:"to"`
			Messages []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"messages"`
			Silent bool `json:"notificationDisabled"`
		}{config.To, []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}{{"text", text}}, config.Silent}), nil
	}, func(data []byte) error {
		var ack struct {
			Sent []struct {
				ID string `json:"id"`
			} `json:"sentMessages"`
		}
		if json.Unmarshal(data, &ack) != nil || len(ack.Sent) != 1 || ack.Sent[0].ID == "" {
			return notification.ErrOutcomeUnknown
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &Driver{hook}, nil
}

// Close releases owned idle connections.
func (d *Driver) Close() error { return d.hook.Close() }

// Send submits one plain-text message without hidden splitting or retries.
func (d *Driver) Send(ctx context.Context, message notification.Message) error {
	return d.hook.Send(ctx, message)
}
