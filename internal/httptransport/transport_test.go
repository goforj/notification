package httptransport

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/goforj/notification"
)

// transportFunc adapts a deterministic wire boundary for client-policy tests.
type transportFunc func(*http.Request) (*http.Response, error)

// RoundTrip delegates without introducing retries or connection ownership.
func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// TestBorrowedTransport preserves explicit connection policy and request bounds.
func TestBorrowedTransport(t *testing.T) {
	calls := 0
	transport := transportFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.Host != "127.0.0.1:1" || r.Method != http.MethodPost || r.Header.Get("Authorization") != "Bearer public-fixture" {
			t.Fatal("request changed")
		}
		if _, ok := r.Context().Deadline(); !ok {
			t.Fatal("client timeout not applied")
		}
		body, err := io.ReadAll(r.Body)
		if err != nil || string(body) != "public-message" {
			t.Fatal("body changed")
		}
		return &http.Response{StatusCode: http.StatusAccepted, Body: io.NopCloser(strings.NewReader("public-response")), Header: make(http.Header), Request: r}, nil
	})
	client, owned, err := New(notification.HTTPConfig{Transport: transport})
	if err != nil || owned != nil || client.Timeout != 10*time.Second {
		t.Fatal("borrowed ownership or defaults changed")
	}
	response, outcome, err := Request(client, context.Background(), http.MethodPost, "https://127.0.0.1:1", http.Header{"Authorization": []string{"Bearer public-fixture"}}, []byte("public-message"))
	if err != nil || outcome.Status != notification.Accepted || !response.Complete || string(response.Body) != "public-response" || calls != 1 {
		t.Fatal("custom transport not used", err)
	}
	for _, timeout := range []time.Duration{-1, time.Minute} {
		if _, _, err := New(notification.HTTPConfig{Transport: transport, Timeout: timeout}); !errors.Is(err, notification.ErrInvalidRequest) {
			t.Fatal("invalid timeout accepted")
		}
	}
	if _, _, err := Request(client, context.Background(), http.MethodPost, "ftp://127.0.0.1", nil, nil); !errors.Is(err, notification.ErrPermanent) || calls != 1 {
		t.Fatal("URL validation bypassed")
	}
}

// TestConcreteTransportIsNotMutated freezes ownership for user TLS, proxies, and dial hooks.
func TestConcreteTransportIsNotMutated(t *testing.T) {
	transport := &http.Transport{MaxConnsPerHost: 37, Proxy: http.ProxyFromEnvironment, DialTLSContext: func(context.Context, string, string) (net.Conn, error) {
		return nil, errors.New("unused public fixture")
	}}
	defer transport.CloseIdleConnections()
	client, owned, err := New(notification.HTTPConfig{Transport: transport})
	if err != nil || owned != nil || client.Transport != transport || transport.MaxConnsPerHost != 37 || transport.Proxy == nil || transport.DialTLSContext == nil {
		t.Fatal("custom transport was cloned or mutated")
	}
}

// TestBorrowedTransportTimeout preserves a deadline while normalizing an ambiguous transport error.
func TestBorrowedTransportTimeout(t *testing.T) {
	sawDeadline := false
	transport := transportFunc(func(r *http.Request) (*http.Response, error) {
		_, sawDeadline = r.Context().Deadline()
		if !sawDeadline {
			return nil, errors.New("deadline missing")
		}
		<-r.Context().Done()
		return nil, r.Context().Err()
	})
	client, _, err := New(notification.HTTPConfig{Transport: transport, Timeout: time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	_, outcome, err := Request(client, context.Background(), http.MethodPost, "https://example.com", nil, nil)
	if !sawDeadline || outcome.Status != notification.Unknown || !errors.Is(err, notification.ErrOutcomeUnknown) {
		t.Fatal("timeout lost uncertainty", err)
	}
}

// TestBorrowedTransportRedirectAndResponseLimits preserves library policy around injected behavior.
func TestBorrowedTransportRedirectAndResponseLimits(t *testing.T) {
	calls := 0
	transport := transportFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: http.StatusFound, Header: http.Header{"Location": []string{"https://elsewhere.example"}}, Body: http.NoBody, Request: r}, nil
	})
	client, _, err := New(notification.HTTPConfig{Transport: transport})
	if err != nil {
		t.Fatal(err)
	}
	if _, outcome, err := Request(client, context.Background(), http.MethodPost, "https://example.com", nil, nil); !errors.Is(err, notification.ErrPermanent) || outcome.Status != notification.Failed || calls != 1 {
		t.Fatal("redirect followed", err)
	}
	transport = func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(strings.Repeat("x", 1<<20))), Request: r}, nil
	}
	client, _, err = New(notification.HTTPConfig{Transport: transport})
	if err != nil {
		t.Fatal(err)
	}
	response, _, err := Request(client, context.Background(), http.MethodPost, "https://example.com", nil, nil)
	if err != nil || response.Complete || len(response.Body) != 65537 {
		t.Fatal("response bound bypassed", err)
	}
}

// TestBorrowedTransportFailureRedaction preserves safe errors and cancellation after dispatch.
func TestBorrowedTransportFailureRedaction(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for _, cancelDuringCall := range []bool{false, true} {
		transport := transportFunc(func(*http.Request) (*http.Response, error) {
			if cancelDuringCall {
				cancel()
			}
			return nil, errors.New("private token and endpoint")
		})
		client, _, err := New(notification.HTTPConfig{Transport: transport})
		if err != nil {
			t.Fatal(err)
		}
		_, outcome, err := Request(client, ctx, http.MethodPost, "https://example.com", nil, nil)
		want := notification.ErrOutcomeUnknown
		if cancelDuringCall {
			want = context.Canceled
		}
		if outcome.Status != notification.Unknown || !errors.Is(err, want) || strings.Contains(err.Error(), "private") {
			t.Fatal("transport failure not sanitized")
		}
	}
}
