// Package webhook delivers bounded generic JSON or Slack-compatible plain text
// to trusted planner-resolved endpoints. It disables proxies and redirects and
// validates every resolved IP at connection time to resist DNS rebinding.
package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/goforj/notification"
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
type Resolver interface {
	LookupNetIP(context.Context, string, string) ([]netip.Addr, error)
}

// Config controls a driver without credentials in endpoint URLs or logs.
type Config struct {
	// URL is the trusted destination used by Send; Deliver uses its planned endpoint.
	URL          string
	Format       Format
	Timeout      time.Duration
	BearerToken  string
	AllowPrivate bool
	Resolver     Resolver
}

// Driver owns a bounded HTTP transport, never injected through application requests.
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
	if (config.Format != Generic && config.Format != Slack) || config.Timeout < time.Millisecond || config.Timeout > 30*time.Second || len(config.BearerToken) > 4096 || strings.ContainsAny(config.BearerToken, "\r\n") {
		return nil, notification.ErrInvalidRequest
	}
	if config.URL != "" {
		if _, err := destination(config.URL); err != nil {
			return nil, notification.ErrInvalidRequest
		}
	}
	resolver := config.Resolver
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.MaxConnsPerHost = 16
	transport.MaxResponseHeaderBytes = 32768
	transport.ResponseHeaderTimeout = config.Timeout
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, notification.ErrPermanent
		}
		ips, err := resolver.LookupNetIP(ctx, "ip", host)
		if err != nil || len(ips) == 0 {
			return nil, notification.ErrRetryable
		}
		for _, ip := range ips {
			if !config.AllowPrivate && !public(ip) {
				return nil, notification.ErrPermanent
			}
		}
		dialer := net.Dialer{Timeout: config.Timeout}
		var conn net.Conn
		for _, ip := range ips {
			conn, err = dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
			if err == nil {
				return conn, nil
			}
		}
		return nil, notification.ErrRetryable
	}
	return &Driver{
		client: &http.Client{
			Transport:     transport,
			Timeout:       config.Timeout,
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
		transport: transport,
		format:    config.Format,
		token:     config.BearerToken,
		endpoint: notification.Endpoint{
			Identity: "configured",
			Address:  notification.Address(config.URL),
		},
	}, nil
}

// Close releases idle connections; callers own shutdown and in-flight cancellation.
func (d *Driver) Close() error { d.transport.CloseIdleConnections(); return nil }

// Deliver performs one POST. Successful HTTP status proves acceptance only.
// No generic idempotency claim is made because arbitrary webhook servers may ignore keys.
func (d *Driver) Deliver(ctx context.Context, delivery notification.Delivery[notification.TextPayload]) (notification.Outcome, error) {
	if err := ctx.Err(); err != nil {
		return failed(err)
	}
	if err := delivery.Payload.Validate(); err != nil {
		return failed(notification.ErrRender)
	}
	u, err := destination(delivery.Endpoint.Address.Reveal())
	if err != nil {
		return failed(err)
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
	req := (&http.Request{Method: http.MethodPost, URL: u, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(data)), ContentLength: int64(len(data))}).WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	if d.token != "" {
		req.Header.Set("Authorization", "Bearer "+d.token)
	}
	response, err := d.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return notification.Outcome{Status: notification.Unknown}, ctx.Err()
		}
		if errors.Is(err, notification.ErrPermanent) {
			return failed(notification.ErrPermanent)
		}
		return notification.Outcome{Status: notification.Unknown}, notification.ErrOutcomeUnknown
	}
	defer response.Body.Close()
	// Provider response bodies are never interpreted as safe error prose.
	body, readErr := io.ReadAll(io.LimitReader(response.Body, 65537))
	switch {
	case response.StatusCode >= 200 && response.StatusCode < 300:
		if d.format == Slack && (readErr != nil || len(body) > 65536 || strings.TrimSpace(string(body)) != "ok") {
			return notification.Outcome{Status: notification.Unknown}, notification.ErrOutcomeUnknown
		}
		return notification.Outcome{Status: notification.Accepted}, nil
	case response.StatusCode == http.StatusTooManyRequests:
		return notification.Outcome{Status: notification.Failed, RetryAfter: retryAfter(response.Header.Get("Retry-After"), time.Now())}, notification.ErrThrottled
	case response.StatusCode == 401 || response.StatusCode == 403:
		return failed(notification.ErrUnauthorized)
	case response.StatusCode >= 500:
		return failed(notification.ErrRetryable)
	default:
		return failed(notification.ErrPermanent)
	}
}

// destination validates trusted configuration before constructing an HTTP request.
func destination(address string) (*url.URL, error) {
	if len(address) > 4096 {
		return nil, notification.ErrPermanent
	}
	u, err := url.Parse(address)
	if err != nil || u.Host == "" || u.User != nil || u.Fragment != "" || (u.Scheme != "https" && u.Scheme != "http") || len(u.String()) > 4096 {
		return nil, notification.ErrPermanent
	}
	if port := u.Port(); port != "" {
		n, err := strconv.Atoi(port)
		if err != nil || n < 1 || n > 65535 {
			return nil, notification.ErrPermanent
		}
	}
	return u, nil
}

// Send delivers plain text to Config.URL without accepting destinations in messages.
func (d *Driver) Send(ctx context.Context, message notification.Message) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := message.Validate(); err != nil {
		return err
	}
	if d.endpoint.Address.Reveal() == "" {
		return notification.ErrInvalidRequest
	}
	_, err := d.Deliver(ctx, notification.Delivery[notification.TextPayload]{
		Payload:  notification.TextPayload{Title: message.Title, Body: message.Body},
		Endpoint: d.endpoint,
	})
	return err
}

// failed constructs proved rejection without exposing transport or provider prose.
func failed(err error) (notification.Outcome, error) {
	return notification.Outcome{Status: notification.Failed}, err
}

// retryAfter bounds server-supplied scheduling advice without sleeping.
func retryAfter(value string, now time.Time) time.Duration {
	seconds, err := strconv.ParseInt(value, 10, 32)
	var wait time.Duration
	if err == nil {
		wait = time.Duration(seconds) * time.Second
	} else if date, err := http.ParseTime(value); err == nil {
		wait = date.Sub(now)
	}
	if wait < 0 {
		return 0
	}
	if wait > 24*time.Hour {
		return 24 * time.Hour
	}
	return wait
}

var reserved = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"), netip.MustParsePrefix("100.64.0.0/10"), netip.MustParsePrefix("192.0.0.0/24"), netip.MustParsePrefix("192.0.2.0/24"), netip.MustParsePrefix("192.88.99.0/24"), netip.MustParsePrefix("198.18.0.0/15"), netip.MustParsePrefix("198.51.100.0/24"), netip.MustParsePrefix("203.0.113.0/24"), netip.MustParsePrefix("240.0.0.0/4"), netip.MustParsePrefix("2001::/23"), netip.MustParsePrefix("2001:db8::/32"), netip.MustParsePrefix("2002::/16"),
}

// public rejects special ranges including mapped private IPv4 and IPv6 transition ranges.
func public(ip netip.Addr) bool {
	ip = ip.Unmap()
	if !ip.IsValid() || !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
		return false
	}
	if ip.Is6() && !netip.MustParsePrefix("2000::/3").Contains(ip) {
		return false
	}
	for _, prefix := range reserved {
		if prefix.Contains(ip) {
			return false
		}
	}
	return true
}
