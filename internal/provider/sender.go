package provider

import (
	"context"
	"net/http"

	"github.com/goforj/notification"
)

// Request contains one prepared native provider operation with no implicit retries.
type Request struct {
	Method string
	URL    string
	Header http.Header
	Body   []byte
}

// Sender centralizes validation and safe errors while each driver owns its wire protocol.
type Sender struct {
	client      *Client
	name        string
	prepare     func(context.Context, notification.Message) (Request, error)
	acknowledge func([]byte) error
	statuses    []int
}

// NewSender borrows a preparation function and transport after provider configuration validation.
func NewSender(name string, config notification.HTTPConfig, prepare func(context.Context, notification.Message) (Request, error), acknowledge func([]byte) error, statuses ...int) (*Sender, error) {
	client, err := New(name, config)
	if err != nil {
		return nil, err
	}
	return &Sender{client: client, name: name, prepare: prepare, acknowledge: acknowledge, statuses: append([]int(nil), statuses...)}, nil
}

// Send validates before constructing a request and preserves provider evidence without raw prose.
func (s *Sender) Send(ctx context.Context, message notification.Message) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := message.Validate(); err != nil {
		return err
	}
	request, err := s.prepare(ctx, message)
	if err != nil {
		return notification.NewDeliveryError(s.name, notification.Outcome{Status: notification.Failed}, err)
	}
	return s.Request(ctx, request)
}

// Close releases only the default owned transport, preserving injected transport ownership.
func (s *Sender) Close() error { return s.client.Close() }

// PostJSON encodes a native JSON request without exposing provider credentials in content.
func PostJSON(endpoint string, headers http.Header, value any) Request {
	headers = headers.Clone()
	if headers == nil {
		headers = make(http.Header)
	}
	headers.Set("Content-Type", "application/json")
	return Request{Method: http.MethodPost, URL: endpoint, Header: headers, Body: JSON(value)}
}

// Request executes an already validated native operation through the same bounded client and acknowledgement policy.
func (s *Sender) Request(ctx context.Context, request Request) error {
	return s.client.Request(ctx, request.Method, request.URL, request.Header, request.Body, s.acknowledge, s.statuses...)
}
