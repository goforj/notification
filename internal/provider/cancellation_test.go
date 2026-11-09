package provider

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/goforj/notification"
)

// canceledBody models caller cancellation after successful headers but before a complete native acknowledgement.
type canceledBody struct{ cancel context.CancelFunc }

// Read cancels the caller while an acknowledgement body is being consumed.
func (body canceledBody) Read([]byte) (int, error) { body.cancel(); return 0, context.Canceled }

// Close satisfies the transport-owned response lifetime without opening a resource.
func (canceledBody) Close() error { return nil }

// cancellationTransport isolates response-body cancellation from actual network activity.
type cancellationTransport struct{ body canceledBody }

// RoundTrip supplies positive headers without falsely supplying complete native acceptance evidence.
func (transport cancellationTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: transport.body, Request: request}, nil
}

// TestAcknowledgementReadCancellation preserves caller cancellation alongside uncertain provider acceptance.
func TestAcknowledgementReadCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client, err := New("fixture", notification.HTTPConfig{Transport: cancellationTransport{body: canceledBody{cancel: cancel}}})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	acknowledged := false
	err = client.Request(ctx, http.MethodPost, "https://notification.invalid", nil, nil, func([]byte) error { acknowledged = true; return nil })
	var detail *notification.DeliveryError
	if !errors.Is(err, context.Canceled) || !errors.Is(err, notification.ErrOutcomeUnknown) || !errors.As(err, &detail) || detail.Status() != notification.Unknown || acknowledged {
		t.Fatal("acknowledgement cancellation was lost or accepted", err)
	}
}
