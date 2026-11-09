// Package httptransport shares bounded, destination-safe HTTP mechanics across drivers.
package httptransport

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"time"

	"github.com/goforj/notification"
)

// New creates a bounded client without I/O. Its second result is the owned
// default transport, or nil when borrowing the explicitly configured transport.
func New(config notification.HTTPConfig) (*http.Client, *http.Transport, error) {
	if config.Timeout == 0 {
		config.Timeout = 10 * time.Second
	}
	if config.Timeout < time.Millisecond || config.Timeout > 30*time.Second {
		return nil, nil, notification.ErrInvalidRequest
	}
	if config.Transport != nil {
		return newClient(config.Transport, config.Timeout), nil, nil
	}
	resolver := config.Resolver
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	var transport *http.Transport
	if defaults, ok := http.DefaultTransport.(*http.Transport); ok {
		transport = defaults.Clone()
	} else {
		// Application instrumentation may replace the global RoundTripper. Keep an
		// owned transport so its wrapper cannot bypass destination checks or redirects.
		transport = &http.Transport{
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: time.Second,
		}
	}
	// HTTPS must use the validated DialContext rather than an inherited TLS dial bypass.
	transport.DialTLS = nil
	transport.DialTLSContext = nil
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
			if !config.AllowPrivate && !Public(ip) {
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
	return newClient(transport, config.Timeout), transport, nil
}

// newClient keeps timeout and redirect behavior independent of transport ownership.
func newClient(transport http.RoundTripper, timeout time.Duration) *http.Client {
	return &http.Client{
		Transport:     transport,
		Timeout:       timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

// Response retains bounded wire evidence for provider-owned acknowledgement parsing.
type Response struct {
	StatusCode int
	Header     http.Header
	Body       []byte
	Complete   bool
}

// Request executes one request without redirects, retries, raw error causes or unbounded bodies.
func Request(client *http.Client, ctx context.Context, method, address string, headers http.Header, body []byte) (Response, notification.Outcome, error) {
	if err := ctx.Err(); err != nil {
		return Response{}, notification.Outcome{Status: notification.Failed}, err
	}
	if len(body) > 128<<10 {
		return Response{}, notification.Outcome{Status: notification.Failed}, notification.ErrInvalidMessage
	}
	u, err := Destination(address)
	if err != nil {
		return Response{}, notification.Outcome{Status: notification.Failed}, err
	}
	request := (&http.Request{
		Method:        method,
		URL:           u,
		Header:        headers.Clone(),
		Body:          io.NopCloser(bytes.NewReader(body)),
		ContentLength: int64(len(body)),
	}).WithContext(ctx)
	response, err := client.Do(request)
	if err != nil {
		if ctx.Err() != nil {
			return Response{}, notification.Outcome{Status: notification.Unknown}, ctx.Err()
		}
		if errors.Is(err, notification.ErrPermanent) {
			return Response{}, notification.Outcome{Status: notification.Failed}, notification.ErrPermanent
		}
		return Response{}, notification.Outcome{Status: notification.Unknown}, notification.ErrOutcomeUnknown
	}
	defer response.Body.Close()
	data, readErr := io.ReadAll(io.LimitReader(response.Body, 65537))
	bounded := Response{StatusCode: response.StatusCode, Header: response.Header.Clone(), Body: data, Complete: readErr == nil && len(data) <= 65536}
	switch {
	case response.StatusCode >= 200 && response.StatusCode < 300:
		return bounded, notification.Outcome{Status: notification.Accepted}, nil
	case response.StatusCode == 429:
		return bounded, notification.Outcome{Status: notification.Failed, RetryAfter: RetryAfter(response.Header.Get("Retry-After"), time.Now())}, notification.ErrThrottled
	case response.StatusCode == 401 || response.StatusCode == 403:
		return bounded, notification.Outcome{Status: notification.Failed}, notification.ErrUnauthorized
	case response.StatusCode >= 500:
		return bounded, notification.Outcome{Status: notification.Failed}, notification.ErrRetryable
	default:
		return bounded, notification.Outcome{Status: notification.Failed}, notification.ErrPermanent
	}
}

// Destination validates trusted configuration before constructing an HTTP request.
func Destination(address string) (*url.URL, error) {
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

// RetryAfter bounds server-supplied scheduling advice without sleeping.
func RetryAfter(value string, now time.Time) time.Duration {
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
	netip.MustParsePrefix("0.0.0.0/8"), netip.MustParsePrefix("100.64.0.0/10"), netip.MustParsePrefix("192.0.0.0/24"), netip.MustParsePrefix("192.0.2.0/24"), netip.MustParsePrefix("192.88.99.0/24"), netip.MustParsePrefix("198.18.0.0/15"), netip.MustParsePrefix("198.51.100.0/24"), netip.MustParsePrefix("203.0.113.0/24"), netip.MustParsePrefix("240.0.0.0/4"), netip.MustParsePrefix("2001::/23"), netip.MustParsePrefix("2001:db8::/32"), netip.MustParsePrefix("2002::/16"), netip.MustParsePrefix("3fff::/20"),
}

// Public rejects special ranges including mapped private IPv4 and IPv6 transition ranges.
func Public(ip netip.Addr) bool {
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
