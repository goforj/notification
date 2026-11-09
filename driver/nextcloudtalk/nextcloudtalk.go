// Package nextcloudtalk sends messages to an existing Nextcloud Talk conversation.
package nextcloudtalk

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/goforj/notification"
	"github.com/goforj/notification/internal/provider"
)

// Config fixes a trusted nextcloudtalk destination and caller-owned connection policy.
type Config struct {
	URL       string
	Username  string
	Password  string
	RoomToken string
	Silent    bool
	HTTP      notification.HTTPConfig
}

// Driver sends synchronously to the configured nextcloudtalk destination.
type Driver struct{ sender *provider.Sender }

// New validates configuration without contacting the provider.
func New(config Config) (*Driver, error) {
	base, err := provider.BaseURL(config.URL, "")
	if err != nil || !provider.Token(config.Username) || !provider.Token(config.Password) || !provider.Token(config.RoomToken) || len(config.RoomToken) > 256 {
		return nil, notification.ErrInvalidRequest
	}
	auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(config.Username+":"+config.Password))
	sender, err := provider.NewSender("nextcloudtalk", config.HTTP, func(ctx context.Context, message notification.Message) (provider.Request, error) {
		return provider.PostJSON(base+"/ocs/v2.php/apps/spreed/api/v1/chat/"+url.PathEscape(config.RoomToken)+"?format=json", http.Header{"Authorization": {auth}, "Ocs-Apirequest": {"true"}, "Accept": {"application/json"}}, struct {
			Message string `json:"message"`
			Silent  bool   `json:"silent"`
		}{provider.Text(message), config.Silent}), nil
	}, func(data []byte) error {
		var ack struct {
			OCS struct {
				Meta struct {
					StatusCode *int `json:"statuscode"`
				} `json:"meta"`
				Data struct {
					ID *int64 `json:"id"`
				} `json:"data"`
			} `json:"ocs"`
		}
		if json.Unmarshal(data, &ack) != nil || ack.OCS.Meta.StatusCode == nil {
			return notification.ErrOutcomeUnknown
		}
		if *ack.OCS.Meta.StatusCode != 100 && *ack.OCS.Meta.StatusCode != 200 {
			return notification.ErrPermanent
		}
		if ack.OCS.Data.ID == nil || *ack.OCS.Data.ID <= 0 {
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
