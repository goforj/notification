package webhook

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/goforj/notification"
	"github.com/goforj/notification/notificationtest"
)

// newDriver constructs a local fixture with explicit private-network permission.
func newDriver(t *testing.T, format Format) *Driver {
	t.Helper()
	d, err := New(Config{Format: format, AllowPrivate: true, BearerToken: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	return d
}

// payload creates a safe transport delivery for one trusted endpoint.
func payload(address string) notification.Delivery[notification.TextPayload] {
	return notification.Delivery[notification.TextPayload]{Payload: notification.TextPayload{Title: "Ticket", Body: "T-42 changed"}, Endpoint: notification.Endpoint{Address: notification.Address(address)}}
}

// TestConformance exercises a real local HTTP provider through the portable inventory.
func TestConformance(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.Header.Get("Authorization") != "Bearer secret" || r.Header.Get("Content-Type") != "application/json" {
			t.Error("request contract")
		}
		var got notification.TextPayload
		if json.NewDecoder(r.Body).Decode(&got) != nil || got.Body != "T-42 changed" {
			t.Error("payload shape")
		}
		w.WriteHeader(202)
	}))
	defer server.Close()
	notificationtest.Driver(t, func() notificationtest.Fixture {
		return notificationtest.Fixture{Driver: newDriver(t, Generic), Delivery: payload(server.URL), Status: notification.Accepted}
	})
	d := newDriver(t, Generic)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := d.Deliver(context.Background(), payload(server.URL)); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
}

// TestResponses verifies classified failures, redirects and provider-body redaction.
func TestResponses(t *testing.T) {
	for _, tt := range []struct {
		status int
		want   error
	}{{429, notification.ErrThrottled}, {401, notification.ErrUnauthorized}, {403, notification.ErrUnauthorized}, {500, notification.ErrRetryable}, {400, notification.ErrPermanent}, {302, notification.ErrPermanent}} {
		t.Run(fmt.Sprint(tt.status), func(t *testing.T) {
			var redirected bool
			target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { redirected = true }))
			defer target.Close()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Retry-After", "999999")
				w.Header().Set("Location", target.URL)
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte("secret provider body"))
			}))
			defer server.Close()
			out, err := newDriver(t, Generic).Deliver(context.Background(), payload(server.URL))
			if !errors.Is(err, tt.want) || strings.Contains(err.Error(), "secret") || redirected || out.IdempotencyUsed {
				t.Fatal(out, err)
			}
			if tt.status == 429 && out.RetryAfter != 24*time.Hour {
				t.Fatal(out)
			}
		})
	}
}

// TestSlack verifies plain text wire format and actual Slack acknowledgement evidence.
func TestSlack(t *testing.T) {
	for _, response := range []string{"ok", "invalid_payload", strings.Repeat("x", 65537)} {
		for _, title := range []string{"Ticket", ""} {
			t.Run(fmt.Sprint(len(response))+title, func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					var data struct {
						Text   string `json:"text"`
						Mrkdwn bool   `json:"mrkdwn"`
					}
					if json.NewDecoder(r.Body).Decode(&data) != nil || data.Mrkdwn || !strings.Contains(data.Text, "T-42") {
						t.Error("Slack shape")
					}
					_, _ = w.Write([]byte(response))
				}))
				defer server.Close()
				p := payload(server.URL)
				p.Payload.Title = title
				out, err := newDriver(t, Slack).Deliver(context.Background(), p)
				if (err == nil) != (response == "ok") || ((out.Status == notification.Accepted) != (response == "ok")) {
					t.Fatal(out, err)
				}
			})
		}
	}
}

// TestValidation rejects malformed endpoints, payloads and configuration before I/O.
func TestValidation(t *testing.T) {
	for _, config := range []Config{{Format: "bad"}, {Timeout: -1}, {Timeout: 31 * time.Second}, {BearerToken: "x\ny"}, {BearerToken: strings.Repeat("x", 4097)}} {
		if _, err := New(config); !errors.Is(err, notification.ErrInvalidRequest) {
			t.Fatal(err)
		}
	}
	d, err := New(Config{})
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	for _, address := range []string{"https://%zz", "https:///missing", "https://u:p@example.com", "https://example.com/#fragment", "ftp://example.com", "http://example.com:70000", "http://example.com:0", "http://example.com:not-port", "https://example.com/" + strings.Repeat("x", 4096)} {
		if _, err := d.Deliver(context.Background(), payload(address)); !errors.Is(err, notification.ErrPermanent) {
			t.Fatal(address, err)
		}
	}
	p := payload("https://example.com")
	p.Payload.Body = ""
	if _, err := d.Deliver(context.Background(), p); !errors.Is(err, notification.ErrRender) {
		t.Fatal(err)
	}
}

// resolverFunc supplies deterministic DNS evidence without external network requests.
type resolverFunc func(context.Context, string, string) ([]netip.Addr, error)

// LookupNetIP invokes the configured deterministic lookup.
func (f resolverFunc) LookupNetIP(ctx context.Context, network, host string) ([]netip.Addr, error) {
	return f(ctx, network, host)
}

// TestDestinationPolicy rejects mixed answers and validates the actual dial path.
func TestDestinationPolicy(t *testing.T) {
	for _, address := range []string{"127.0.0.1", "10.0.0.1", "169.254.169.254", "192.0.2.1", "100.64.0.1", "0.0.0.0", "224.0.0.1", "240.0.0.1", "::1", "::ffff:127.0.0.1", "fc00::1", "fe80::1", "64:ff9b::1", "2001:db8::1", "2002::1"} {
		if public(netip.MustParseAddr(address)) {
			t.Fatalf("special address accepted: %s", address)
		}
	}
	for _, address := range []string{"8.8.8.8", "2606:4700:4700::1111"} {
		if !public(netip.MustParseAddr(address)) {
			t.Fatal(address)
		}
	}
	if public(netip.Addr{}) {
		t.Fatal("invalid IP accepted")
	}
	d, err := New(Config{Resolver: resolverFunc(func(context.Context, string, string) ([]netip.Addr, error) {
		return []netip.Addr{netip.MustParseAddr("8.8.8.8"), netip.MustParseAddr("127.0.0.1")}, nil
	})})
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	if _, err := d.Deliver(context.Background(), payload("http://public.example")); !errors.Is(err, notification.ErrPermanent) {
		t.Fatal(err)
	}
	for _, resolver := range []Resolver{resolverFunc(func(context.Context, string, string) ([]netip.Addr, error) { return nil, errors.New("secret DNS") }), resolverFunc(func(context.Context, string, string) ([]netip.Addr, error) { return nil, nil })} {
		d, _ := New(Config{Resolver: resolver})
		_, err := d.transport.DialContext(context.Background(), "tcp", "host:80")
		if !errors.Is(err, notification.ErrRetryable) {
			t.Fatal(err)
		}
		_ = d.Close()
	}
	if _, err := d.transport.DialContext(context.Background(), "tcp", "malformed"); !errors.Is(err, notification.ErrPermanent) {
		t.Fatal(err)
	}
	private := newDriver(t, Generic)
	if _, err := private.transport.DialContext(context.Background(), "tcp", "127.0.0.1:0"); !errors.Is(err, notification.ErrRetryable) {
		t.Fatal(err)
	}
}

// TestTransportUncertainty preserves cancellation after dispatch without exposing raw URLs.
func TestTransportUncertainty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hijacker := w.(http.Hijacker)
		conn, _, _ := hijacker.Hijack()
		_ = conn.Close()
	}))
	defer server.Close()
	out, err := newDriver(t, Generic).Deliver(context.Background(), payload(server.URL))
	if out.Status != notification.Unknown || !errors.Is(err, notification.ErrOutcomeUnknown) {
		t.Fatal(out, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	d, _ := New(Config{AllowPrivate: true, Resolver: resolverFunc(func(context.Context, string, string) ([]netip.Addr, error) { cancel(); return nil, context.Canceled })})
	defer d.Close()
	out, err = d.Deliver(ctx, payload("http://example.test"))
	if out.Status != notification.Unknown || !errors.Is(err, context.Canceled) {
		t.Fatal(out, err)
	}
}

// TestRetryAfter bounds numeric and HTTP-date scheduling advice.
func TestRetryAfter(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	for _, tt := range []struct {
		value string
		want  time.Duration
	}{{"5", 5 * time.Second}, {"-1", 0}, {"bad", 0}, {"999999", 24 * time.Hour}, {now.Add(2 * time.Minute).Format(http.TimeFormat), 2 * time.Minute}, {now.Add(-time.Hour).Format(http.TimeFormat), 0}} {
		if got := retryAfter(tt.value, now); got != tt.want {
			t.Fatal(tt.value, got)
		}
	}
}

// TestDNSPinning connects to the resolved IP rather than resolving the hostname twice.
func TestDNSPinning(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }))
	defer server.Close()
	_, port, _ := net.SplitHostPort(strings.TrimPrefix(server.URL, "http://"))
	d, _ := New(Config{AllowPrivate: true, Resolver: resolverFunc(func(context.Context, string, string) ([]netip.Addr, error) {
		return []netip.Addr{netip.MustParseAddr("127.0.0.1")}, nil
	})})
	defer d.Close()
	out, err := d.Deliver(context.Background(), payload("http://unresolvable.invalid:"+port))
	if err != nil || out.Status != notification.Accepted {
		t.Fatal(out, err)
	}
}
