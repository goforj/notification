// Package gotify sends native messages through Gotify's application API.
package gotify

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/goforj/notification"
	"github.com/goforj/notification/internal/provider"
)

// Config supplies a trusted server and application token. Priority zero preserves
// the server's application default; a positive value sends an explicit integer.
type Config struct {
	URL      string
	Token    string
	Priority int
	HTTP     notification.HTTPConfig
}

// Driver owns an immutable Gotify application destination and bounded transport.
type Driver struct {
	client   *provider.Client
	url      string
	token    string
	priority int
}

// New validates configuration without provisioning a Gotify application.
func New(config Config) (*Driver, error) {
	endpoint, err := provider.BaseURL(config.URL, "")
	if err != nil || !provider.Token(config.Token) || config.Priority < 0 || int64(config.Priority) > 2147483647 {
		return nil, notification.ErrInvalidRequest
	}
	client, err := provider.New("gotify", config.HTTP)
	if err != nil {
		return nil, err
	}
	return &Driver{client, endpoint + "/message", config.Token, config.Priority}, nil
}

// Close releases only owned idle connections; injected transports remain caller-owned.
func (d *Driver) Close() error { return d.client.Close() }

// Send confirms a native message ID and the echoed content before returning success.
func (d *Driver) Send(ctx context.Context, message notification.Message) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := message.Validate(); err != nil {
		return err
	}
	body := provider.JSON(struct {
		Title    string `json:"title,omitempty"`
		Message  string `json:"message"`
		Priority int    `json:"priority,omitempty"`
	}{message.Title, message.Body, d.priority})
	headers := http.Header{
		"Content-Type": []string{"application/json"},
		"X-Gotify-Key": []string{d.token},
	}
	return d.client.Request(ctx, http.MethodPost, d.url, headers, body, func(data []byte) error {
		var ack struct {
			ID      *int64 `json:"id"`
			Title   string `json:"title"`
			Message string `json:"message"`
		}
		if json.Unmarshal(data, &ack) != nil || ack.ID == nil || *ack.ID <= 0 || ack.Title != message.Title || ack.Message != message.Body {
			return notification.ErrOutcomeUnknown
		}
		return nil
	})
}
