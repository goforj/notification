// Package pushover submits non-emergency Pushover messages with explicit priority.
package pushover

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"unicode/utf8"

	"github.com/goforj/notification"
	"github.com/goforj/notification/internal/provider"
)

// Config fixes a trusted pushover destination and caller-owned connection policy.
type Config struct {
	Token  string
	User   string
	Device string
	// Priority ranges from -2 to 1. Emergency priority requires receipt management and is not accepted here.
	Priority int
	BaseURL  string
	HTTP     notification.HTTPConfig
}

// Driver sends synchronously to the configured pushover destination.
type Driver struct{ sender *provider.Sender }

// New validates configuration without contacting the provider.
func New(config Config) (*Driver, error) {
	base, err := provider.BaseURL(config.BaseURL, "https://api.pushover.net")
	if err != nil || !provider.Token(config.Token) || !provider.Token(config.User) || (config.Device != "" && !provider.Token(config.Device)) || config.Priority < -2 || config.Priority > 1 {
		return nil, notification.ErrInvalidRequest
	}
	sender, err := provider.NewSender("pushover", config.HTTP, func(ctx context.Context, message notification.Message) (provider.Request, error) {
		if utf8.RuneCountInString(message.Title) > 250 || utf8.RuneCountInString(message.Body) > 1024 {
			return provider.Request{}, notification.ErrRender
		}
		body := url.Values{"token": {config.Token}, "user": {config.User}, "title": {message.Title}, "message": {message.Body}, "priority": {strconv.Itoa(config.Priority)}}
		if config.Device != "" {
			body.Set("device", config.Device)
		}
		return provider.Request{Method: http.MethodPost, URL: base + "/1/messages.json", Header: http.Header{"Content-Type": {"application/x-www-form-urlencoded"}}, Body: []byte(body.Encode())}, nil
	}, func(data []byte) error {
		var ack struct {
			Status  *int   `json:"status"`
			Request string `json:"request"`
		}
		if json.Unmarshal(data, &ack) != nil || ack.Status == nil {
			return notification.ErrOutcomeUnknown
		}
		if *ack.Status != 1 {
			return notification.ErrPermanent
		}
		if ack.Request == "" {
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
