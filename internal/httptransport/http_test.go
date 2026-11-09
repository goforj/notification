package httptransport

import (
	"context"
	"errors"
	"net/http"
	"net/netip"
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
