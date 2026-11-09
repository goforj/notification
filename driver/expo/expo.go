// Package expo submits native Expo push tickets without claiming device delivery.
package expo

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"

	"github.com/goforj/notification"
	"github.com/goforj/notification/internal/provider"
)

// Config fixes a trusted expo destination and caller-owned connection policy.
type Config struct {
	PushToken   string
	AccessToken string
	BaseURL     string
	HTTP        notification.HTTPConfig
}

// Driver sends synchronously to the configured expo destination.
type Driver struct{ sender *provider.Sender }

// New validates configuration without contacting the provider.
func New(config Config) (*Driver, error) {
	base, err := provider.BaseURL(config.BaseURL, "https://exp.host")
	valid, _ := regexp.MatchString(`^(Expo|Exponent)PushToken\[[A-Za-z0-9_-]{1,256}\]$`, config.PushToken)
	if err != nil || !valid || (config.AccessToken != "" && !provider.Token(config.AccessToken)) {
		return nil, notification.ErrInvalidRequest
	}
	headers := http.Header{}
	if config.AccessToken != "" {
		headers.Set("Authorization", "Bearer "+config.AccessToken)
	}
	sender, err := provider.NewSender("expo", config.HTTP, func(ctx context.Context, message notification.Message) (provider.Request, error) {
		request := provider.PostJSON(base+"/--/api/v2/push/send", headers, struct {
			To    string `json:"to"`
			Title string `json:"title,omitempty"`
			Body  string `json:"body"`
		}{config.PushToken, message.Title, message.Body})
		if len(request.Body) > 4096 {
			return provider.Request{}, notification.ErrRender
		}
		return request, nil
	}, func(data []byte) error {
		var ack struct {
			Data struct {
				Status  string `json:"status"`
				ID      string `json:"id"`
				Details struct {
					Error string `json:"error"`
				} `json:"details"`
			} `json:"data"`
		}
		if json.Unmarshal(data, &ack) != nil {
			return notification.ErrOutcomeUnknown
		}
		if ack.Data.Status == "error" {
			switch ack.Data.Details.Error {
			case "InvalidCredentials":
				return notification.ErrUnauthorized
			case "MessageRateExceeded":
				return notification.ErrThrottled
			case "MessageTooBig":
				return notification.ErrRender
			default:
				return notification.ErrPermanent
			}
		}
		if ack.Data.Status != "ok" || ack.Data.ID == "" {
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
