package provider

import (
	"context"
	"net/http"

	"github.com/goforj/notification"
)

// Hook pairs provider-owned plain-message rendering and acknowledgement rules.
type Hook struct {
	client      *Client
	name        string
	url         string
	headers     http.Header
	render      func(notification.Message) ([]byte, error)
	acknowledge func([]byte) error
}

// NewHook validates a configured destination and owns a copied header catalog.
func NewHook(name, endpoint string, config notification.HTTPConfig, headers http.Header, render func(notification.Message) ([]byte, error), acknowledge func([]byte) error) (*Hook, error) {
	if err := Endpoint(endpoint); err != nil {
		return nil, err
	}
	client, err := New(name, config)
	if err != nil {
		return nil, err
	}
	return &Hook{client, name, endpoint, headers.Clone(), render, acknowledge}, nil
}

// Close releases owned idle connections.
func (h *Hook) Close() error { return h.client.Close() }

// Send validates content before provider-specific bounds and one HTTP request.
func (h *Hook) Send(ctx context.Context, message notification.Message) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := message.Validate(); err != nil {
		return err
	}
	body, err := h.render(message)
	if err != nil {
		return notification.NewDeliveryError(h.name, notification.Outcome{Status: notification.Failed}, err)
	}
	return h.client.Request(ctx, http.MethodPost, h.url, h.headers, body, h.acknowledge)
}
