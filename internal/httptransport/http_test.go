package httptransport

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"sync/atomic"
	"testing"

	"github.com/goforj/notification"
)

// TestRequestValidation proves preflight bounds fail before any network call.
func TestRequestValidation(t *testing.T) {
	client, transport, err := New(notification.HTTPConfig{})
	if err != nil {
		t.Fatal(err)
	}
	defer transport.CloseIdleConnections()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, out, err := Request(client, ctx, http.MethodPost, "https://example.com", nil, nil); !errors.Is(err, context.Canceled) || out.Status != notification.Failed {
		t.Fatal("canceled request attempted")
	}
	if _, out, err := Request(client, context.Background(), http.MethodPost, "https://example.com", nil, make([]byte, (128<<10)+1)); !errors.Is(err, notification.ErrInvalidMessage) || out.Status != notification.Failed {
		t.Fatal("oversized request attempted")
	}
	if Public(netip.MustParseAddr("3fff::1")) {
		t.Fatal("IPv6 documentation range permitted")
	}
}

// customDefaultTransport models application-wide HTTP instrumentation outside the library's ownership.
type customDefaultTransport struct{}

// RoundTrip must not replace the notification transport's destination policy.
func (customDefaultTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("application transport must not handle notification requests")
}

// TestCustomDefaultTransport permits standard HTTP instrumentation without sharing its transport policy.
func TestCustomDefaultTransport(t *testing.T) {
	original := http.DefaultTransport
	http.DefaultTransport = customDefaultTransport{}
	defer func() { http.DefaultTransport = original }()
	client, transport, err := New(notification.HTTPConfig{})
	if err != nil {
		t.Fatal(err)
	}
	defer transport.CloseIdleConnections()
	if client.Transport != transport || transport.Proxy != nil || transport.DialContext == nil || !transport.ForceAttemptHTTP2 {
		t.Fatal("owned destination policy changed")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusAccepted) }))
	defer server.Close()
	if _, out, err := Request(client, context.Background(), http.MethodPost, server.URL, nil, nil); !errors.Is(err, notification.ErrPermanent) || out.Status != notification.Failed {
		t.Fatal("private destination policy bypassed", err)
	}
	allowed, owned, err := New(notification.HTTPConfig{AllowPrivate: true})
	if err != nil {
		t.Fatal(err)
	}
	defer owned.CloseIdleConnections()
	if _, out, err := Request(allowed, context.Background(), http.MethodPost, server.URL, nil, nil); err != nil || out.Status != notification.Accepted {
		t.Fatal("owned transport did not deliver", err)
	}
}

// TestGlobalTLSDialHooksCannotBypassPolicy keeps HTTPS on the same validated dial path as HTTP.
func TestGlobalTLSDialHooksCannotBypassPolicy(t *testing.T) {
	original := http.DefaultTransport
	defer func() { http.DefaultTransport = original }()
	for _, hook := range []string{"context", "legacy"} {
		t.Run(hook, func(t *testing.T) {
			var calls atomic.Int64
			defaults := &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12}}
			if hook == "context" {
				defaults.DialTLSContext = func(context.Context, string, string) (net.Conn, error) {
					calls.Add(1)
					return nil, errors.New("global TLS dial bypass")
				}
			} else {
				defaults.DialTLS = func(string, string) (net.Conn, error) { calls.Add(1); return nil, errors.New("global TLS dial bypass") }
			}
			http.DefaultTransport = defaults
			client, transport, err := New(notification.HTTPConfig{})
			if err != nil {
				t.Fatal(err)
			}
			defer transport.CloseIdleConnections()
			if transport.TLSClientConfig == defaults.TLSClientConfig || transport.TLSClientConfig.MinVersion != tls.VersionTLS12 {
				t.Fatal("TLS settings were not cloned")
			}
			_, out, err := Request(client, context.Background(), http.MethodPost, "https://127.0.0.1:1", nil, nil)
			if !errors.Is(err, notification.ErrPermanent) || out.Status != notification.Failed || calls.Load() != 0 {
				t.Fatal("global TLS hook bypassed destination policy")
			}
		})
	}
}

// TestEmptyHTTPHost rejects malformed authorities before clients can attempt resolution or transport.
func TestEmptyHTTPHost(t *testing.T) {
	for _, address := range []string{"http://:80", "https://:443/path", "http://[]:8080"} {
		if _, err := Destination(address); !errors.Is(err, notification.ErrPermanent) {
			t.Fatal("empty hostname accepted", address, err)
		}
	}
	for _, address := range []string{"http://example.com:80", "https://[2001:db8::1]:443"} {
		if _, err := Destination(address); err != nil {
			t.Fatal("valid authority rejected", err)
		}
	}
}
