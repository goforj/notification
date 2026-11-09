package driver_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/goforj/notification"
	"github.com/goforj/notification/driver/jsm"
	"github.com/goforj/notification/driver/opsgenie"
	"github.com/goforj/notification/driver/pagerduty"
	"github.com/goforj/notification/driver/twilio"
)

// borrowedTransport records calls and closure independently of the configured driver.
type borrowedTransport struct {
	base          http.RoundTripper
	calls, closes atomic.Int64
}

// RoundTrip delegates through a concurrency-safe caller-supplied transport.
func (t *borrowedTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	t.calls.Add(1)
	return t.base.RoundTrip(r)
}

// CloseIdleConnections records accidental library ownership of a borrowed resource.
func (t *borrowedTransport) CloseIdleConnections() {
	t.closes.Add(1)
	if closer, ok := t.base.(interface{ CloseIdleConnections() }); ok {
		closer.CloseIdleConnections()
	}
}

// responseTransport models a provider wire acknowledgement without a network dependency.
type responseTransport string

// RoundTrip exposes actual configured request URLs and bounded native response handling.
func (t responseTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusAccepted, Body: io.NopCloser(strings.NewReader(string(t))), Header: make(http.Header), Request: r}, nil
}

// TestLifecycleAndSMSTransport verifies injection through constructors outside the shared chat catalog.
func TestLifecycleAndSMSTransport(t *testing.T) {
	cases := []struct {
		name, ack string
		open      func(notification.HTTPConfig) (nativeSender, error)
	}{
		{"pagerduty", `{"status":"success","dedup_key":"transport-key"}`, func(h notification.HTTPConfig) (nativeSender, error) {
			return pagerduty.New(pagerduty.Config{RoutingKey: strings.Repeat("a", 32), Key: "transport-key", HTTP: h})
		}},
		{"jsm", `{"requestId":"request-id"}`, func(h notification.HTTPConfig) (nativeSender, error) {
			return jsm.New(jsm.Config{CloudID: "cloud", Email: "fixture@example.com", Token: "public-token", Key: "transport-key", HTTP: h})
		}},
		{"opsgenie", `{"requestId":"request-id"}`, func(h notification.HTTPConfig) (nativeSender, error) {
			return opsgenie.New(opsgenie.Config{Token: "public-token", Key: "transport-key", HTTP: h})
		}},
		{"twilio", `{"sid":"SM` + strings.Repeat("c", 32) + `","status":"queued"}`, func(h notification.HTTPConfig) (nativeSender, error) {
			return twilio.New(twilio.Config{AccountSID: "AC" + strings.Repeat("a", 32), AuthToken: "public-token", From: "+12125550100", To: "+12125550101", HTTP: h})
		}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			transport := &borrowedTransport{base: responseTransport(test.ack)}
			driver, err := test.open(notification.HTTPConfig{Transport: transport})
			if err != nil {
				t.Fatal(err)
			}
			if transport.calls.Load() != 0 {
				t.Fatal("constructor performed I/O")
			}
			message := notification.Message{Title: "Ticket", Body: "T-42 changed"}
			if err := driver.Send(context.Background(), message); err != nil {
				t.Fatal(err)
			}
			if err := driver.Close(); err != nil {
				t.Fatal(err)
			}
			if transport.closes.Load() != 0 {
				t.Fatal("driver closed borrowed transport")
			}
			if err := driver.Send(context.Background(), message); err != nil || transport.calls.Load() != 2 {
				t.Fatal("transport unavailable after driver Close", err)
			}
			transport.CloseIdleConnections()
			if transport.closes.Load() != 1 {
				t.Fatal("caller lost transport ownership")
			}
		})
	}
}
