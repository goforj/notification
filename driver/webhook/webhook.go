// Package webhook delivers bounded generic JSON or Slack-compatible plain text
// to trusted planner-resolved endpoints. It disables proxies and redirects and
// validates every resolved IP at connection time to resist DNS rebinding when
// using its default transport. Explicit injected transports own network policy.
package webhook

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/goforj/notification"
	"github.com/goforj/notification/internal/httptransport"
)

// Format chooses an explicit outbound wire shape.
type Format string

const (
	// Generic sends the TextPayload JSON representation.
	Generic Format = "generic"
	// Slack sends a Slack incoming-webhook text field with mrkdwn disabled.
	Slack Format = "slack"
)

// Resolver supplies all IP addresses for connection-time destination validation.
type Resolver = notification.Resolver

// Config controls a driver without credentials in endpoint URLs or logs.
type Config struct {
	// URL is the trusted destination used by Send; Deliver uses its planned endpoint.
	URL          string
	Format       Format
	Timeout      time.Duration
	BearerToken  string
	AllowPrivate bool
	Resolver     Resolver
	// Transport is borrowed unchanged; nil selects the destination-checked default.
	// An injected transport owns DNS, TLS, proxy, dialing, and retry policy.
	Transport http.RoundTripper
}

// Driver owns request bounds and borrows an optional construction-time transport.
type Driver struct {
	client    *http.Client
	transport *http.Transport
	format    Format
	token     string
	endpoint  notification.Endpoint
}

// New constructs a reusable driver without network activity. AllowPrivate is an
// explicit deployment choice for local services, never a request-controlled flag.
func New(config Config) (*Driver, error) {
	if config.Format == "" {
		config.Format = Generic
	}
	if config.Timeout == 0 {
		config.Timeout = 10 * time.Second
	}
	if (config.Format != Generic && config.Format != Slack) || len(config.BearerToken) > 4096 || strings.ContainsAny(config.BearerToken, "\r\n") {
		return nil, notification.ErrInvalidRequest
	}
	if config.URL != "" {
		if _, err := httptransport.Destination(config.URL); err != nil {
			return nil, notification.ErrInvalidRequest
		}
	}
	client, transport, err := httptransport.New(notification.HTTPConfig{
		Timeout: config.Timeout, AllowPrivate: config.AllowPrivate, Resolver: config.Resolver, Transport: config.Transport,
	})
	if err != nil {
		return nil, err
	}

	return &Driver{
		client:    client,
		transport: transport,
		format:    config.Format,
		token:     config.BearerToken,
		endpoint: notification.Endpoint{
			Identity: "configured",
			Address:  notification.Address(config.URL),
		},
	}, nil
}

// Close releases only owned idle connections. Injected transports remain caller-owned.
func (d *Driver) Close() error {
	if d.transport != nil {
		d.transport.CloseIdleConnections()
	}
	return nil
}

// Deliver performs one POST. Successful HTTP status proves acceptance only.
// No generic idempotency claim is made because arbitrary webhook servers may ignore keys.
func (d *Driver) Deliver(ctx context.Context, delivery notification.Delivery[notification.TextPayload]) (notification.Outcome, error) {
	if err := ctx.Err(); err != nil {
		return failed(err)
	}
	if err := delivery.Payload.Validate(); err != nil {
		return failed(notification.ErrRender)
	}

	var data []byte
	if d.format == Slack {
		text := delivery.Payload.Body
		if delivery.Payload.Title != "" {
			text = delivery.Payload.Title + "\n" + text
		}
		data, _ = json.Marshal(struct {
			Text   string `json:"text"`
			Mrkdwn bool   `json:"mrkdwn"`
		}{text, false})
	} else {
		data, _ = json.Marshal(delivery.Payload)
	}
	headers := http.Header{"Content-Type": []string{"application/json"}}
	if d.token != "" {
		headers.Set("Authorization", "Bearer "+d.token)
	}
	response, outcome, err := httptransport.Request(d.client, ctx, http.MethodPost, delivery.Endpoint.Address.Reveal(), headers, data)
	if err != nil {
		return outcome, err
	}
	if d.format == Slack && (!response.Complete || strings.TrimSpace(string(response.Body)) != "ok") {
		return notification.Outcome{Status: notification.Unknown}, errors.Join(notification.ErrOutcomeUnknown, ctx.Err())
	}
	return outcome, nil
}

// Send delivers plain text to Config.URL without accepting destinations in messages.
func (d *Driver) Send(ctx context.Context, message notification.Message) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := message.Validate(); err != nil {
		return err
	}
	if d.endpoint.Address.Reveal() == "" {
		return notification.ErrInvalidRequest
	}
	outcome, err := d.Deliver(ctx, notification.Delivery[notification.TextPayload]{
		Payload:  notification.TextPayload{Title: message.Title, Body: message.Body},
		Endpoint: d.endpoint,
	})
	if err != nil {
		return notification.NewDeliveryError("webhook", outcome, err)
	}
	return nil
}

// failed constructs proved rejection without exposing transport or provider prose.
func failed(err error) (notification.Outcome, error) {
	return notification.Outcome{Status: notification.Failed}, err
}
