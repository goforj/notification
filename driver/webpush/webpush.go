// Package webpush sends encrypted browser push messages with VAPID authentication.
package webpush

import (
	"bytes"
	"context"
	"crypto/ecdh"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/goforj/notification"
	"github.com/goforj/notification/internal/httptransport"
	"github.com/goforj/notification/internal/provider"
)

// Subscription is the browser PushSubscription representation, interoperable with the underlying Web Push library.
type Subscription = webpush.Subscription

// Keys contains the browser's authentication secret and P-256 ECDH public key.
type Keys = webpush.Keys

// Urgency is the standard Web Push delivery priority.
type Urgency = webpush.Urgency

const (
	// UrgencyVeryLow allows delivery when the device is idle and connected to power.
	UrgencyVeryLow = webpush.UrgencyVeryLow
	// UrgencyLow allows delivery when the device is idle.
	UrgencyLow = webpush.UrgencyLow
	// UrgencyNormal requests ordinary delivery and is the default.
	UrgencyNormal = webpush.UrgencyNormal
	// UrgencyHigh requests prompt delivery when the device is available.
	UrgencyHigh = webpush.UrgencyHigh
)

// Config fixes a browser subscription, VAPID identity and caller-owned connection policy.
// Subscription data is supplied by the browser. Subscriber is a contact email or HTTPS URL.
type Config struct {
	Subscription    Subscription
	Subscriber      string
	VAPIDPublicKey  string
	VAPIDPrivateKey string
	TTL             int
	Topic           string
	Urgency         Urgency
	HTTP            notification.HTTPConfig
}

// Driver owns a bounded client and immutable copies of subscription and signing material.
type Driver struct {
	client    *http.Client
	transport *http.Transport
	config    Config
}

// decodeKey accepts browser URL-safe keys with optional base64 padding.
func decodeKey(value string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(strings.TrimRight(value, "="))
}

// New validates keys and subscription ownership without connecting or generating replacement credentials.
func New(config Config) (*Driver, error) {
	endpoint, err := httptransport.Destination(config.Subscription.Endpoint)
	if err != nil || endpoint.Scheme != "https" || len(config.Subscriber) > 256 {
		return nil, notification.ErrInvalidRequest
	}
	for _, value := range []string{config.Subscription.Keys.Auth, config.Subscription.Keys.P256dh, config.VAPIDPublicKey, config.VAPIDPrivateKey} {
		if len(value) > 128 {
			return nil, notification.ErrInvalidRequest
		}
	}
	auth, authErr := decodeKey(config.Subscription.Keys.Auth)
	subscription, subscriptionErr := decodeKey(config.Subscription.Keys.P256dh)
	public, publicErr := decodeKey(config.VAPIDPublicKey)
	private, privateErr := decodeKey(config.VAPIDPrivateKey)
	if authErr != nil || subscriptionErr != nil || publicErr != nil || privateErr != nil || len(auth) != 16 {
		return nil, notification.ErrInvalidRequest
	}
	if _, err := ecdh.P256().NewPublicKey(subscription); err != nil {
		return nil, notification.ErrInvalidRequest
	}
	signing, err := ecdh.P256().NewPrivateKey(private)
	if err != nil || !bytes.Equal(signing.PublicKey().Bytes(), public) {
		return nil, notification.ErrInvalidRequest
	}
	if strings.HasPrefix(config.Subscriber, "https://") {
		contact, err := url.Parse(config.Subscriber)
		if err != nil || contact.Host == "" || contact.User != nil {
			return nil, notification.ErrInvalidRequest
		}
	} else {
		config.Subscriber = strings.TrimPrefix(config.Subscriber, "mailto:")
		address, err := mail.ParseAddress(config.Subscriber)
		if err != nil || address.Address != config.Subscriber || address.Name != "" {
			return nil, notification.ErrInvalidRequest
		}
	}
	if config.TTL < 0 || config.TTL > 2419200 {
		return nil, notification.ErrInvalidRequest
	}
	if config.Topic != "" {
		valid, _ := regexp.MatchString(`^[A-Za-z0-9_-]{1,32}$`, config.Topic)
		if !valid {
			return nil, notification.ErrInvalidRequest
		}
	}
	if config.Urgency == "" {
		config.Urgency = webpush.UrgencyNormal
	}
	switch config.Urgency {
	case webpush.UrgencyVeryLow, webpush.UrgencyLow, webpush.UrgencyNormal, webpush.UrgencyHigh:
	default:
		return nil, notification.ErrInvalidRequest
	}
	config.Subscription.Keys.Auth = base64.RawURLEncoding.EncodeToString(auth)
	config.Subscription.Keys.P256dh = base64.RawURLEncoding.EncodeToString(subscription)
	config.VAPIDPublicKey = base64.RawURLEncoding.EncodeToString(public)
	config.VAPIDPrivateKey = base64.RawURLEncoding.EncodeToString(private)
	client, transport, err := httptransport.New(config.HTTP)
	if err != nil {
		return nil, err
	}
	return &Driver{client: client, transport: transport, config: config}, nil
}

// Send encrypts one title/body JSON payload and requires native push-service creation acceptance.
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
	payload := provider.JSON(notification.TextPayload{Title: message.Title, Body: message.Body})
	// This bound leaves space for RFC 8291 framing inside the SDK's single 4096-byte record.
	if len(payload) > 3900 {
		return notification.NewDeliveryError("webpush", notification.Outcome{Status: notification.Failed}, notification.ErrRender)
	}
	response, err := webpush.SendNotificationWithContext(ctx, payload, &d.config.Subscription, &webpush.Options{
		HTTPClient: d.client, Subscriber: d.config.Subscriber, VAPIDPublicKey: d.config.VAPIDPublicKey, VAPIDPrivateKey: d.config.VAPIDPrivateKey,
		TTL: d.config.TTL, Topic: d.config.Topic, Urgency: d.config.Urgency,
	})
	if err != nil {
		if ctx.Err() != nil {
			return notification.NewDeliveryError("webpush", notification.Outcome{Status: notification.Unknown}, ctx.Err())
		}
		if errors.Is(err, notification.ErrPermanent) {
			return notification.NewDeliveryError("webpush", notification.Outcome{Status: notification.Failed}, notification.ErrPermanent)
		}
		return notification.NewDeliveryError("webpush", notification.Outcome{Status: notification.Unknown}, notification.ErrOutcomeUnknown)
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 65537))
	switch response.StatusCode {
	case http.StatusCreated:
		return nil
	case http.StatusTooManyRequests:
		return notification.NewDeliveryError("webpush", notification.Outcome{Status: notification.Failed, RetryAfter: httptransport.RetryAfter(response.Header.Get("Retry-After"), time.Now())}, notification.ErrThrottled)
	case http.StatusUnauthorized, http.StatusForbidden:
		return notification.NewDeliveryError("webpush", notification.Outcome{Status: notification.Failed}, notification.ErrUnauthorized)
	}
	if response.StatusCode >= 500 {
		return notification.NewDeliveryError("webpush", notification.Outcome{Status: notification.Failed}, notification.ErrRetryable)
	}
	if response.StatusCode >= 200 && response.StatusCode < 300 {
		return notification.NewDeliveryError("webpush", notification.Outcome{Status: notification.Unknown}, notification.ErrOutcomeUnknown)
	}
	return notification.NewDeliveryError("webpush", notification.Outcome{Status: notification.Failed}, notification.ErrPermanent)
}

// Close releases only owned idle connections; injected transports remain caller-owned.
func (d *Driver) Close() error {
	if d.transport != nil {
		d.transport.CloseIdleConnections()
	}
	return nil
}
