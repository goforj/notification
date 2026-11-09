// Package provider shares validated HTTP execution while drivers own wire contracts.
package provider

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/goforj/notification"
	"github.com/goforj/notification/internal/httptransport"
)

// Client owns one bounded transport and a static provider identity.
type Client struct {
	client    *http.Client
	transport *http.Transport
	name      string
}

// New constructs an owned provider transport without network I/O.
func New(name string, config notification.HTTPConfig) (*Client, error) {
	client, transport, err := httptransport.New(config)
	if err != nil {
		return nil, err
	}
	return &Client{client, transport, name}, nil
}

// Close releases idle connections without canceling caller-owned contexts.
func (c *Client) Close() error {
	c.transport.CloseIdleConnections()
	return nil
}

// Request sends once and applies provider-owned acknowledgement validation.
// A nil validator means HTTP status alone is the documented acceptance contract.
func (c *Client) Request(ctx context.Context, method, endpoint string, headers http.Header, body []byte, acknowledge func([]byte) error) error {
	response, outcome, err := httptransport.Request(c.client, ctx, method, endpoint, headers, body)
	if err != nil {
		return notification.NewDeliveryError(c.name, outcome, err)
	}
	if acknowledge != nil {
		if !response.Complete {
			return notification.NewDeliveryError(c.name, notification.Outcome{Status: notification.Unknown}, notification.ErrOutcomeUnknown)
		}
		if err := acknowledge(response.Body); err != nil {
			status := notification.Unknown
			if errors.Is(err, notification.ErrPermanent) || errors.Is(err, notification.ErrUnauthorized) || errors.Is(err, notification.ErrRetryable) || errors.Is(err, notification.ErrThrottled) {
				status = notification.Failed
			}
			var detail *notification.DeliveryError
			advice := notification.Outcome{Status: status}
			if errors.As(err, &detail) {
				advice.RetryAfter = detail.RetryAfter()
			}
			return notification.NewDeliveryError(c.name, advice, err)
		}
	}
	return nil
}

// JSON encodes library-owned scalar wire structs. Unsupported internal payload
// types panic instead of quietly sending an empty body after a programming error.
func JSON(value any) []byte {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return data
}

// BaseURL validates an API base and permits explicit reverse-proxy path prefixes.
func BaseURL(raw, fallback string) (string, error) {
	if raw == "" {
		raw = fallback
	}
	u, err := httptransport.Destination(raw)
	if err != nil || u.RawQuery != "" {
		return "", notification.ErrInvalidRequest
	}
	return strings.TrimRight(u.String(), "/"), nil
}

// Endpoint validates an exact configured webhook URL, including provider query tokens.
func Endpoint(raw string) error {
	if _, err := httptransport.Destination(raw); err != nil {
		return notification.ErrInvalidRequest
	}
	return nil
}

// Token rejects unbounded credentials and controls before header or path construction.
func Token(value string) bool {
	if value == "" || len(value) > 4096 || !utf8.ValidString(value) {
		return false
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

// Text preserves a title in transports that expose one plain-text message field.
func Text(message notification.Message) string {
	if message.Title == "" {
		return message.Body
	}
	return message.Title + "\n" + message.Body
}

// Query appends encoded provider parameters to a previously validated endpoint.
func Query(endpoint string, values map[string]string) string {
	u, _ := url.Parse(endpoint)
	query := u.Query()
	for key, value := range values {
		query.Set(key, value)
	}
	u.RawQuery = query.Encode()
	return u.String()
}
