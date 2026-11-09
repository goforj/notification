// Package matrix sends unencrypted room messages through the Matrix client-server API.
package matrix

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/goforj/notification"
	"github.com/goforj/notification/internal/provider"
)

// Config fixes a trusted matrix destination and caller-owned connection policy.
type Config struct {
	URL   string
	Token string
	// RoomID is an existing joined room ID. Encrypted rooms are not supported.
	RoomID string
	HTTP   notification.HTTPConfig
}

// Driver sends synchronously to the configured matrix destination.
type Driver struct{ sender *provider.Sender }

// New validates configuration without contacting the provider.
func New(config Config) (*Driver, error) {
	base, err := provider.BaseURL(config.URL, "")
	if err != nil || !provider.Token(config.Token) || !provider.Token(config.RoomID) || !strings.HasPrefix(config.RoomID, "!") || len(config.RoomID) < 2 || len(config.RoomID) > 255 {
		return nil, notification.ErrInvalidRequest
	}
	sender, err := provider.NewSender("matrix", config.HTTP, func(ctx context.Context, message notification.Message) (provider.Request, error) {
		var random [16]byte
		// A fresh transaction per Send prevents unrelated calls from sharing Matrix deduplication identity.
		_, _ = rand.Read(random[:])
		request := provider.PostJSON(base+"/_matrix/client/v3/rooms/"+url.PathEscape(config.RoomID)+"/send/m.room.message/"+hex.EncodeToString(random[:]), http.Header{"Authorization": {"Bearer " + config.Token}}, struct {
			Type string `json:"msgtype"`
			Body string `json:"body"`
		}{"m.text", provider.Text(message)})
		request.Method = http.MethodPut
		return request, nil
	}, func(data []byte) error {
		var ack struct {
			ID string `json:"event_id"`
		}
		if json.Unmarshal(data, &ack) != nil || !strings.HasPrefix(ack.ID, "$") {
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
