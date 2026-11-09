// Package fcm submits native Firebase Cloud Messaging HTTP v1 notifications.
package fcm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/goforj/notification"
	"github.com/goforj/notification/internal/provider"
)

// Config fixes a trusted fcm destination and caller-owned connection policy.
type Config struct {
	ProjectID   string
	DeviceToken string
	Topic       string
	BaseURL     string
	// TokenSource is a required, concurrency-safe borrowed function that obtains a current OAuth bearer token.
	// Applications can adapt their Google credential library without adding SDK dependencies to the root module.
	TokenSource func(context.Context) (string, error)
	HTTP        notification.HTTPConfig
}

// Driver sends synchronously to the configured fcm destination.
type Driver struct{ sender *provider.Sender }

// New validates configuration without contacting the provider.
func New(config Config) (*Driver, error) {
	base, err := provider.BaseURL(config.BaseURL, "https://fcm.googleapis.com")
	if err != nil || !provider.Token(config.ProjectID) || (config.DeviceToken == "") == (config.Topic == "") || (config.DeviceToken != "" && !provider.Token(config.DeviceToken)) || (config.Topic != "" && !provider.Token(config.Topic)) {
		return nil, notification.ErrInvalidRequest
	}
	sender, err := provider.NewSender("fcm", config.HTTP, func(ctx context.Context, message notification.Message) (provider.Request, error) {
		token, err := config.TokenSource(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return provider.Request{}, ctx.Err()
			}
			return provider.Request{}, notification.ErrRetryable
		}
		if !provider.Token(token) {
			return provider.Request{}, notification.ErrUnauthorized
		}
		request := provider.PostJSON(base+"/v1/projects/"+url.PathEscape(config.ProjectID)+"/messages:send", http.Header{"Authorization": {"Bearer " + token}}, struct {
			Message struct {
				Token        string `json:"token,omitempty"`
				Topic        string `json:"topic,omitempty"`
				Notification struct {
					Title string `json:"title,omitempty"`
					Body  string `json:"body"`
				} `json:"notification"`
			} `json:"message"`
		}{struct {
			Token        string `json:"token,omitempty"`
			Topic        string `json:"topic,omitempty"`
			Notification struct {
				Title string `json:"title,omitempty"`
				Body  string `json:"body"`
			} `json:"notification"`
		}{config.DeviceToken, config.Topic, struct {
			Title string `json:"title,omitempty"`
			Body  string `json:"body"`
		}{message.Title, message.Body}}})
		if len(request.Body) > 4096 {
			return provider.Request{}, notification.ErrRender
		}
		return request, nil
	}, func(data []byte) error {
		var ack struct {
			Name string `json:"name"`
		}
		prefix := "projects/" + config.ProjectID + "/messages/"
		if json.Unmarshal(data, &ack) != nil || !strings.HasPrefix(ack.Name, prefix) || len(ack.Name) <= len(prefix) {
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
