package apns

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/goforj/notification"
	"github.com/goforj/notification/notificationtest"
	"github.com/golang-jwt/jwt/v5"
)

// signingKey creates disposable signing material for local native-protocol verification.
func signingKey(t *testing.T, curve elliptic.Curve) *ecdsa.PrivateKey {
	t.Helper()
	key, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

// configuration supplies one valid application identity without touching a hosted account.
func configuration(t *testing.T) Config {
	return Config{Key: signingKey(t, elliptic.P256()), KeyID: "KEY0000042", TeamID: "TEAM000042", Topic: "example.app", DeviceToken: strings.Repeat("ab", 32), Sound: "default", CollapseID: "tickets"}
}

// TestNativeHTTP2 verifies JWT signing, payloads, concurrent token reuse and borrowed TLS transport ownership.
func TestNativeHTTP2(t *testing.T) {
	config := configuration(t)
	var code atomic.Int64
	code.Store(200)
	var missingID atomic.Bool
	var mu sync.Mutex
	bearers := map[string]bool{}
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor != 2 || r.Method != "POST" || r.URL.Path != "/3/device/"+config.DeviceToken || r.Header.Get("Apns-Topic") != "example.app" || r.Header.Get("Apns-Push-Type") != "alert" || r.Header.Get("Apns-Priority") != "10" || r.Header.Get("Apns-Collapse-Id") != "tickets" {
			t.Error("APNs native routing or headers")
		}
		bearer := strings.TrimPrefix(r.Header.Get("Authorization"), "bearer ")
		parsed, err := jwt.Parse(bearer, func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodES256 {
				t.Error("wrong APNs signing algorithm")
			}
			return &config.Key.PublicKey, nil
		}, jwt.WithValidMethods([]string{"ES256"}))
		if err != nil || !parsed.Valid || parsed.Header["kid"] != config.KeyID {
			t.Error("invalid provider JWT")
		}
		mu.Lock()
		bearers[bearer] = true
		mu.Unlock()
		var payload struct {
			APS struct {
				Alert struct {
					Body string `json:"body"`
				} `json:"alert"`
				Sound string `json:"sound"`
			} `json:"aps"`
		}
		if json.NewDecoder(r.Body).Decode(&payload) != nil || payload.APS.Alert.Body == "" || payload.APS.Sound != "default" {
			t.Error("APNs payload")
		}
		if !missingID.Load() {
			w.Header().Set("apns-id", "request-42")
		}
		w.Header().Set("Retry-After", "7")
		w.WriteHeader(int(code.Load()))
	}))
	server.EnableHTTP2 = true
	server.StartTLS()
	defer server.Close()
	config.BaseURL = server.URL
	config.HTTP.Transport = server.Client().Transport
	driver, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	// The driver's owned copy must survive subsequent caller changes to the input key.
	config.Key.D.SetInt64(0)
	notificationtest.Sender(t, func() notification.Sender { return driver })
	if err := driver.Close(); err != nil {
		t.Fatal(err)
	}
	if err := driver.Send(nil, notification.Message{Body: "after Close"}); err != nil {
		t.Fatal("borrowed transport was closed", err)
	}
	mu.Lock()
	count := len(bearers)
	mu.Unlock()
	if count != 1 {
		t.Fatal("provider JWT was regenerated for each request")
	}
	for _, entry := range []struct {
		status int
		want   error
	}{{202, notification.ErrOutcomeUnknown}, {429, notification.ErrThrottled}, {403, notification.ErrUnauthorized}, {410, notification.ErrPermanent}, {408, notification.ErrRetryable}, {503, notification.ErrRetryable}} {
		code.Store(int64(entry.status))
		err := driver.Send(nil, notification.Message{Body: "status check"})
		if !errors.Is(err, entry.want) {
			t.Fatal("APNs status classification", entry.status, err)
		}
		if entry.want == notification.ErrRetryable {
			var detail *notification.DeliveryError
			if !errors.Is(err, notification.ErrOutcomeUnknown) || !errors.As(err, &detail) || detail.Status() != notification.Unknown || detail.RetryAfter() != 7*time.Second {
				t.Fatal("APNs server failure lost uncertainty or retry advice", err)
			}
		}
	}
	code.Store(200)
	missingID.Store(true)
	if err := driver.Send(nil, notification.Message{Body: "missing ID"}); !errors.Is(err, notification.ErrOutcomeUnknown) {
		t.Fatal("missing native ID accepted", err)
	}
	if err := driver.Send(nil, notification.Message{Body: strings.Repeat("x", 4096)}); !errors.Is(err, notification.ErrRender) {
		t.Fatal("APNs payload bound", err)
	}
	server.Close()
	if err := driver.Send(nil, notification.Message{Body: "network failure"}); !errors.Is(err, notification.ErrOutcomeUnknown) {
		t.Fatal("unsafe transport outcome", err)
	}
}

// TestConfigurationAndTokenCache checks malformed key state, invalid settings and JWT refresh boundaries.
func TestConfigurationAndTokenCache(t *testing.T) {
	base := configuration(t)
	variants := []Config{}
	add := func(change func(*Config)) { value := base; change(&value); variants = append(variants, value) }
	add(func(c *Config) { c.Key = nil })
	add(func(c *Config) { c.Key = signingKey(t, elliptic.P384()) })
	add(func(c *Config) { c.Key = &ecdsa.PrivateKey{PublicKey: base.Key.PublicKey, D: big.NewInt(0)} })
	add(func(c *Config) {
		c.Key = &ecdsa.PrivateKey{PublicKey: base.Key.PublicKey, D: new(big.Int).Set(elliptic.P256().Params().N)}
	})
	add(func(c *Config) {
		c.Key = &ecdsa.PrivateKey{PublicKey: ecdsa.PublicKey{Curve: elliptic.P256()}, D: big.NewInt(1)}
	})
	add(func(c *Config) {
		c.Key = &ecdsa.PrivateKey{PublicKey: ecdsa.PublicKey{Curve: elliptic.P256(), X: big.NewInt(1), Y: big.NewInt(1)}, D: big.NewInt(1)}
	})
	add(func(c *Config) { c.Key = &ecdsa.PrivateKey{PublicKey: base.Key.PublicKey, D: big.NewInt(1)} })
	add(func(c *Config) { c.DeviceToken = "not hex" })
	add(func(c *Config) { c.DeviceToken = "" })
	add(func(c *Config) { c.DeviceToken = strings.Repeat("ab", 513) })
	add(func(c *Config) { c.KeyID = "bad\n" })
	add(func(c *Config) { c.TeamID = "" })
	add(func(c *Config) { c.Topic = "" })
	add(func(c *Config) { c.Sound = "bad\n" })
	add(func(c *Config) { c.CollapseID = strings.Repeat("x", 65) })
	add(func(c *Config) { c.Priority = 1 })
	add(func(c *Config) { c.BaseURL = "://" })
	add(func(c *Config) { c.BaseURL = "http://example.com" })
	add(func(c *Config) { c.HTTP.Timeout = -1 })
	for _, value := range variants {
		if driver, err := New(value); err == nil {
			_ = driver.Close()
			t.Fatal("invalid APNs configuration accepted")
		}
	}
	base.Priority = 5
	base.Sandbox = true
	driver, err := New(base)
	if err != nil {
		t.Fatal(err)
	}
	defer driver.Close()
	if !strings.Contains(driver.config.BaseURL, "sandbox") {
		t.Fatal("sandbox ignored")
	}
	now := time.Now()
	first, err := driver.token(now)
	if err != nil {
		t.Fatal(err)
	}
	second, err := driver.token(now.Add(time.Minute))
	if err != nil || first != second {
		t.Fatal("cached provider JWT not reused")
	}
	refreshed, err := driver.token(now.Add(time.Hour))
	if err != nil || refreshed == first {
		t.Fatal("expired provider JWT not refreshed")
	}
	if _, err := driver.token(now.Add(-time.Hour)); err != nil {
		t.Fatal("clock rollback", err)
	}
	// A private fault injection exercises signing failure without weakening the public constructor.
	driver.key = signingKey(t, elliptic.P384())
	driver.bearer = ""
	if err := driver.Send(nil, notification.Message{Body: "signing failure"}); !errors.Is(err, notification.ErrInvalidRequest) {
		t.Fatal("signing failure lost", err)
	}
}

// TestHTTP1IsNotAPNs rejects a success-looking response over an unsupported protocol.
func TestHTTP1IsNotAPNs(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Header().Set("Apns-Id", "fixture") }))
	defer server.Close()
	config := configuration(t)
	config.BaseURL = server.URL
	config.HTTP.Transport = server.Client().Transport
	driver, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	defer driver.Close()
	if err := driver.Send(nil, notification.Message{Body: "HTTP/1 response"}); !errors.Is(err, notification.ErrOutcomeUnknown) {
		t.Fatal("non-native protocol accepted", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := driver.Send(ctx, notification.Message{Body: "canceled"}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

// TestCertificateTransport verifies native HTTP/2 client-certificate authentication without a provider JWT.
func TestCertificateTransport(t *testing.T) {
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor != 2 || len(r.TLS.PeerCertificates) != 1 || r.Header.Get("Authorization") != "" {
			t.Error("certificate authentication or HTTP/2 was bypassed")
		}
		w.Header().Set("Apns-Id", "request-42")
	}))
	server.EnableHTTP2 = true
	server.TLS = &tls.Config{ClientAuth: tls.RequireAnyClientCert}
	server.StartTLS()
	defer server.Close()
	transport := server.Client().Transport.(*http.Transport).Clone()
	transport.TLSClientConfig.Certificates = []tls.Certificate{server.TLS.Certificates[0]}
	defer transport.CloseIdleConnections()
	config := Config{Topic: "example.app", DeviceToken: strings.Repeat("ab", 32), BaseURL: server.URL, HTTP: notification.HTTPConfig{Transport: transport}}
	driver, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	defer driver.Close()
	if err := driver.Send(nil, notification.Message{Body: "client certificate"}); err != nil {
		t.Fatal(err)
	}
}
