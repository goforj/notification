package webpush

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/goforj/notification"
	"github.com/goforj/notification/notificationtest"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/hkdf"
)

// receiver creates independent browser subscription and application VAPID keys.
func receiver(t *testing.T) (Config, *ecdh.PrivateKey, []byte) {
	t.Helper()
	key, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	auth := make([]byte, 16)
	_, _ = rand.Read(auth)
	private, public, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		t.Fatal(err)
	}
	return Config{Subscription: webpush.Subscription{Endpoint: "https://example.com/push", Keys: webpush.Keys{Auth: base64.RawURLEncoding.EncodeToString(auth), P256dh: base64.RawURLEncoding.EncodeToString(key.PublicKey().Bytes())}}, Subscriber: "mailto:helpdesk@example.com", VAPIDPublicKey: public, VAPIDPrivateKey: private, Topic: "tickets", TTL: 60}, key, auth
}

// decryptPush independently verifies RFC 8291 encryption as a browser receiver would.
func decryptPush(t *testing.T, body []byte, key *ecdh.PrivateKey, auth []byte) []byte {
	t.Helper()
	if len(body) < 86 || body[20] != 65 {
		t.Fatal("invalid encrypted record framing")
	}
	public, err := ecdh.P256().NewPublicKey(body[21:86])
	if err != nil {
		t.Fatal(err)
	}
	shared, err := key.ECDH(public)
	if err != nil {
		t.Fatal(err)
	}
	info := append([]byte("WebPush: info\x00"), key.PublicKey().Bytes()...)
	info = append(info, public.Bytes()...)
	intermediate := make([]byte, 32)
	if _, err := io.ReadFull(hkdf.New(sha256.New, shared, auth, info), intermediate); err != nil {
		t.Fatal(err)
	}
	secret := hkdf.Extract(sha256.New, intermediate, body[:16])
	contentKey, nonce := make([]byte, 16), make([]byte, 12)
	if _, err := io.ReadFull(hkdf.Expand(sha256.New, secret, []byte("Content-Encoding: aes128gcm\x00")), contentKey); err != nil {
		t.Fatal(err)
	}
	if _, err := io.ReadFull(hkdf.Expand(sha256.New, secret, []byte("Content-Encoding: nonce\x00")), nonce); err != nil {
		t.Fatal(err)
	}
	block, err := aes.NewCipher(contentKey)
	if err != nil {
		t.Fatal(err)
	}
	encryption, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatal(err)
	}
	plaintext, err := encryption.Open(nil, nonce, body[86:], nil)
	if err != nil {
		t.Fatal(err)
	}
	plaintext = bytes.TrimRight(plaintext, "\x00")
	if len(plaintext) == 0 || plaintext[len(plaintext)-1] != 2 {
		t.Fatal("missing final record delimiter")
	}
	return plaintext[:len(plaintext)-1]
}

// TestNativeEncryption verifies decrypted title/body, VAPID identity and native acceptance over TLS.
func TestNativeEncryption(t *testing.T) {
	config, key, auth := receiver(t)
	var status atomic.Int64
	status.Store(201)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.Header.Get("Content-Encoding") != "aes128gcm" || r.Header.Get("TTL") != "60" || r.Header.Get("Topic") != "tickets" {
			t.Error("native push headers")
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, 8192))
		if err != nil {
			t.Error(err)
			return
		}
		plaintext := decryptPush(t, body, key, auth)
		var message notification.TextPayload
		if json.Unmarshal(plaintext, &message) != nil || message.Body == "" {
			t.Error("decrypted notification content")
		}
		authorization := strings.TrimPrefix(r.Header.Get("Authorization"), "vapid t=")
		parts := strings.Split(authorization, ", k=")
		if len(parts) != 2 || parts[1] != config.VAPIDPublicKey {
			t.Error("VAPID public identity")
			return
		}
		// The subscription ECDH key is deliberately independent of the signing identity.
		vapidPublic, _ := decodeKey(config.VAPIDPublicKey)
		x, y := elliptic.Unmarshal(elliptic.P256(), vapidPublic)
		token, err := jwt.Parse(parts[0], func(*jwt.Token) (any, error) { return &ecdsa.PublicKey{Curve: elliptic.P256(), X: x, Y: y}, nil }, jwt.WithValidMethods([]string{"ES256"}))
		if err != nil || !token.Valid || token.Claims.(jwt.MapClaims)["sub"] != "mailto:helpdesk@example.com" || token.Claims.(jwt.MapClaims)["aud"] != "https://"+r.Host {
			t.Error("invalid VAPID signature or contact")
		}
		w.Header().Set("Retry-After", "7")
		w.WriteHeader(int(status.Load()))
	}))
	defer server.Close()
	config.Subscription.Endpoint = server.URL + "/push"
	config.HTTP.Transport = server.Client().Transport
	driver, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	notificationtest.Sender(t, func() notification.Sender { return driver })
	if err := driver.Close(); err != nil {
		t.Fatal(err)
	}
	if err := driver.Send(nil, notification.Message{Body: "borrowed transport retained"}); err != nil {
		t.Fatal(err)
	}
	for _, entry := range []struct {
		status int
		want   error
	}{{200, notification.ErrOutcomeUnknown}, {429, notification.ErrThrottled}, {401, notification.ErrUnauthorized}, {410, notification.ErrPermanent}, {408, notification.ErrRetryable}, {503, notification.ErrRetryable}} {
		status.Store(int64(entry.status))
		err := driver.Send(nil, notification.Message{Body: "status check"})
		if !errors.Is(err, entry.want) {
			t.Fatal("push status classification", entry.status, err)
		}
		if entry.want == notification.ErrRetryable {
			var detail *notification.DeliveryError
			if !errors.Is(err, notification.ErrOutcomeUnknown) || !errors.As(err, &detail) || detail.Status() != notification.Unknown || detail.RetryAfter() != 7*time.Second {
				t.Fatal("server error lost uncertainty or retry advice", err)
			}
		}
	}
	if err := driver.Send(nil, notification.Message{Body: strings.Repeat("x", 3900)}); !errors.Is(err, notification.ErrRender) {
		t.Fatal("encrypted record bound", err)
	}
	server.Close()
	if err := driver.Send(nil, notification.Message{Body: "network failure"}); !errors.Is(err, notification.ErrOutcomeUnknown) {
		t.Fatal("transport uncertainty", err)
	}
}

// roundTripperFunc scripts transport cancellation without bypassing real encryption and signing.
type roundTripperFunc func(*http.Request) (*http.Response, error)

// RoundTrip invokes the explicitly configured transport fixture.
func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// TestConfiguration validates native key material and the caller-owned transport boundary.
func TestConfiguration(t *testing.T) {
	base, _, _ := receiver(t)
	var variants []Config
	add := func(change func(*Config)) { value := base; change(&value); variants = append(variants, value) }
	add(func(c *Config) { c.Subscription.Endpoint = "http://example.com" })
	add(func(c *Config) { c.Subscription.Endpoint = "://" })
	add(func(c *Config) { c.Subscriber = strings.Repeat("x", 257) })
	add(func(c *Config) { c.VAPIDPublicKey = strings.Repeat("x", 129) })
	add(func(c *Config) { c.Subscription.Keys.Auth = "invalid!" })
	add(func(c *Config) { c.Subscription.Keys.Auth = base64.RawURLEncoding.EncodeToString([]byte("short")) })
	add(func(c *Config) { c.Subscription.Keys.P256dh = "invalid!" })
	add(func(c *Config) {
		c.Subscription.Keys.P256dh = base64.RawURLEncoding.EncodeToString([]byte("invalid point"))
	})
	add(func(c *Config) { c.VAPIDPrivateKey = "invalid!" })
	add(func(c *Config) { c.VAPIDPrivateKey = base64.RawURLEncoding.EncodeToString(make([]byte, 32)) })
	add(func(c *Config) { c.VAPIDPublicKey = "invalid!" })
	add(func(c *Config) { c.VAPIDPublicKey = c.Subscription.Keys.P256dh })
	add(func(c *Config) { c.Subscriber = "https://user:password@example.com" })
	add(func(c *Config) { c.Subscriber = "bad email" })
	add(func(c *Config) { c.TTL = -1 })
	add(func(c *Config) { c.TTL = 2419201 })
	add(func(c *Config) { c.Topic = "contains spaces" })
	add(func(c *Config) { c.Urgency = "invalid" })
	add(func(c *Config) { c.HTTP.Timeout = -1 })
	for _, config := range variants {
		if driver, err := New(config); err == nil {
			_ = driver.Close()
			t.Fatal("invalid Web Push configuration accepted")
		}
	}
	base.Subscriber = "https://example.com/contact"
	base.Topic = ""
	base.TTL = 0
	base.Urgency = UrgencyHigh
	owned, err := New(base)
	if err != nil {
		t.Fatal(err)
	}
	_ = owned.Close()
	// The default transport must reject loopback even after successful encryption and signing.
	base.Subscription.Endpoint = "https://127.0.0.1/push"
	owned, err = New(base)
	if err != nil {
		t.Fatal(err)
	}
	if err := owned.Send(nil, notification.Message{Body: "private destination"}); !errors.Is(err, notification.ErrPermanent) {
		t.Fatal("destination policy", err)
	}
	_ = owned.Close()
	ctx, cancel := context.WithCancel(context.Background())
	base.HTTP.Transport = roundTripperFunc(func(*http.Request) (*http.Response, error) { cancel(); return nil, context.Canceled })
	driver, err := New(base)
	if err != nil {
		t.Fatal(err)
	}
	if err := driver.Send(ctx, notification.Message{Body: "canceled during transport"}); !errors.Is(err, context.Canceled) || !errors.Is(err, notification.ErrOutcomeUnknown) {
		t.Fatal("in-flight cancellation evidence", err)
	}
	_ = driver.Close()
}
