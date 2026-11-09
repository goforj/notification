// Package apns sends alert notifications over the native APNs HTTP/2 protocol.
package apns

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/hex"
	"math/big"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/goforj/notification"
	"github.com/goforj/notification/internal/httptransport"
	"github.com/goforj/notification/internal/provider"
	"github.com/golang-jwt/jwt/v5"
)

// Config supplies a token-authentication signing key and one configured application device.
// Key is validated and copied when supplied. For certificate authentication, leave Key nil
// and explicitly supply an HTTP/2 Transport configured with the caller-owned client certificate.
// HTTP.Transport owns certificate, routing and TLS policy in either mode.
type Config struct {
	Key         *ecdsa.PrivateKey
	KeyID       string
	TeamID      string
	Topic       string
	DeviceToken string
	Sound       string
	CollapseID  string
	Priority    int
	Sandbox     bool
	BaseURL     string
	HTTP        notification.HTTPConfig
}

// Driver owns a bounded APNs client and a synchronized cache of short-lived provider JWTs.
type Driver struct {
	client    *http.Client
	transport *http.Transport
	key       *ecdsa.PrivateKey
	config    Config
	mu        sync.Mutex
	bearer    string
	issued    time.Time
}

// New validates credential configuration without signing, connecting or registering a device.
func New(config Config) (*Driver, error) {
	key := config.Key
	var cloned *ecdsa.PrivateKey
	if key == nil {
		if config.HTTP.Transport == nil {
			return nil, notification.ErrInvalidRequest
		}
	} else {
		if key.Curve != elliptic.P256() || key.D == nil || key.D.Sign() <= 0 || key.D.Cmp(elliptic.P256().Params().N) >= 0 || key.X == nil || key.Y == nil || !elliptic.P256().IsOnCurve(key.X, key.Y) || !provider.Token(config.KeyID) || !provider.Token(config.TeamID) {
			return nil, notification.ErrInvalidRequest
		}
		x, y := elliptic.P256().ScalarBaseMult(key.D.Bytes())
		if x.Cmp(key.X) != 0 || y.Cmp(key.Y) != 0 {
			return nil, notification.ErrInvalidRequest
		}
		cloned = &ecdsa.PrivateKey{PublicKey: ecdsa.PublicKey{Curve: elliptic.P256(), X: new(big.Int).Set(key.X), Y: new(big.Int).Set(key.Y)}, D: new(big.Int).Set(key.D)}
	}
	if len(config.DeviceToken) > 1024 {
		return nil, notification.ErrInvalidRequest
	}
	token, err := hex.DecodeString(config.DeviceToken)
	if err != nil || len(token) == 0 || len(token) > 512 || !provider.Token(config.Topic) || (config.Sound != "" && !provider.Token(config.Sound)) || len(config.CollapseID) > 64 || (config.CollapseID != "" && !provider.Token(config.CollapseID)) {
		return nil, notification.ErrInvalidRequest
	}
	if config.Priority == 0 {
		config.Priority = 10
	}
	if config.Priority != 5 && config.Priority != 10 {
		return nil, notification.ErrInvalidRequest
	}
	fallback := "https://api.push.apple.com"
	if config.Sandbox {
		fallback = "https://api.sandbox.push.apple.com"
	}
	base, err := provider.BaseURL(config.BaseURL, fallback)
	if err != nil {
		return nil, err
	}
	endpoint, err := httptransport.Destination(base)
	if err != nil || endpoint.Scheme != "https" {
		return nil, notification.ErrInvalidRequest
	}
	config.BaseURL = base
	client, transport, err := httptransport.New(config.HTTP)
	if err != nil {
		return nil, err
	}
	config.Key = nil
	return &Driver{client: client, transport: transport, key: cloned, config: config}, nil
}

// token reuses a JWT for fifty minutes to avoid provider token-update throttling.
func (d *Driver) token(now time.Time) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.bearer != "" && now.Sub(d.issued) >= 0 && now.Sub(d.issued) < 50*time.Minute {
		return d.bearer, nil
	}
	token := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{"iss": d.config.TeamID, "iat": now.Unix()})
	token.Header["kid"] = d.config.KeyID
	bearer, err := token.SignedString(d.key)
	if err != nil {
		return "", notification.ErrInvalidRequest
	}
	d.bearer, d.issued = bearer, now
	return bearer, nil
}

// Send submits one alert and requires native HTTP/2 acceptance with an APNs request identity.
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
	payload := provider.JSON(struct {
		APS struct {
			Alert struct {
				Title string `json:"title,omitempty"`
				Body  string `json:"body"`
			} `json:"alert"`
			Sound string `json:"sound,omitempty"`
		} `json:"aps"`
	}{struct {
		Alert struct {
			Title string `json:"title,omitempty"`
			Body  string `json:"body"`
		} `json:"alert"`
		Sound string `json:"sound,omitempty"`
	}{struct {
		Title string `json:"title,omitempty"`
		Body  string `json:"body"`
	}{message.Title, message.Body}, d.config.Sound}})
	if len(payload) > 4096 {
		return notification.NewDeliveryError("apns", notification.Outcome{Status: notification.Failed}, notification.ErrRender)
	}
	headers := http.Header{"Content-Type": {"application/json"}, "Apns-Topic": {d.config.Topic}, "Apns-Push-Type": {"alert"}, "Apns-Priority": {strconv.Itoa(d.config.Priority)}}
	if d.key != nil {
		bearer, err := d.token(time.Now())
		if err != nil {
			return notification.NewDeliveryError("apns", notification.Outcome{Status: notification.Failed}, err)
		}
		headers.Set("Authorization", "bearer "+bearer)
	}
	if d.config.CollapseID != "" {
		headers.Set("Apns-Collapse-Id", d.config.CollapseID)
	}
	response, outcome, err := httptransport.Request(d.client, ctx, http.MethodPost, d.config.BaseURL+"/3/device/"+d.config.DeviceToken, headers, payload)
	if err != nil {
		return notification.NewDeliveryError("apns", outcome, err)
	}
	if response.StatusCode != http.StatusOK || response.ProtocolMajor != 2 || response.Header.Get("Apns-Id") == "" {
		return notification.NewDeliveryError("apns", notification.Outcome{Status: notification.Unknown}, notification.ErrOutcomeUnknown)
	}
	return nil
}

// Close releases only owned idle connections; injected transports remain caller-owned.
func (d *Driver) Close() error {
	if d.transport != nil {
		d.transport.CloseIdleConnections()
	}
	return nil
}
