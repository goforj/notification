// Package plivo submits a single SMS through the native Plivo Messages API.
package plivo

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/goforj/notification"
	"github.com/goforj/notification/internal/provider"
)

// Config fixes a trusted plivo destination and caller-owned connection policy.
type Config struct {
	AuthID    string
	AuthToken string
	From      string
	To        string
	BaseURL   string
	HTTP      notification.HTTPConfig
}

// Driver sends synchronously to the configured plivo destination.
type Driver struct{ sender *provider.Sender }

// New validates configuration without contacting the provider.
func New(config Config) (*Driver, error) {
	base, err := provider.BaseURL(config.BaseURL, "https://api.plivo.com")
	if err != nil || !provider.Token(config.AuthID) || !provider.Token(config.AuthToken) || !provider.Phone(config.From) || !provider.Phone(config.To) {
		return nil, notification.ErrInvalidRequest
	}
	auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(config.AuthID+":"+config.AuthToken))
	sender, err := provider.NewSender("plivo", config.HTTP, func(ctx context.Context, message notification.Message) (provider.Request, error) {
		text, err := provider.SMSText(message)
		if err != nil {
			return provider.Request{}, err
		}
		return provider.PostJSON(base+"/v1/Account/"+url.PathEscape(config.AuthID)+"/Message/", http.Header{"Authorization": {auth}}, struct {
			From string `json:"src"`
			To   string `json:"dst"`
			Text string `json:"text"`
		}{config.From, config.To, text}), nil
	}, func(data []byte) error {
		var ack struct {
			APIID string   `json:"api_id"`
			IDs   []string `json:"message_uuid"`
		}
		if json.Unmarshal(data, &ack) != nil || ack.APIID == "" || len(ack.IDs) != 1 || ack.IDs[0] == "" {
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
