// Command webpush encrypts a browser notification for a local TLS push-service fixture.
package main

import (
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/goforj/notification"
	"github.com/goforj/notification/driver/webpush"
)

// main generates disposable subscription and VAPID keys for this local example only.
func main() {
	service := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusCreated) }))
	defer service.Close()
	browserKey, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	signingKey, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	auth := make([]byte, 16)
	_, _ = rand.Read(auth)
	subscription := webpush.Subscription{
		Endpoint: service.URL + "/push",
		Keys: webpush.Keys{
			Auth:   base64.RawURLEncoding.EncodeToString(auth),
			P256dh: base64.RawURLEncoding.EncodeToString(browserKey.PublicKey().Bytes()),
		},
	}
	// snippet:start
	driver, err := webpush.New(webpush.Config{
		Subscription:    subscription,
		Subscriber:      "helpdesk@example.com",
		VAPIDPrivateKey: base64.RawURLEncoding.EncodeToString(signingKey.Bytes()),
		VAPIDPublicKey:  base64.RawURLEncoding.EncodeToString(signingKey.PublicKey().Bytes()),
		TTL:             60,
		Urgency:         webpush.UrgencyHigh,
		HTTP: notification.HTTPConfig{
			Transport: service.Client().Transport,
		},
	})
	if err != nil {
		panic(err)
	}
	defer driver.Close()
	err = driver.Send(context.Background(), notification.Message{
		Title: "Ticket updated",
		Body:  "Ticket T-42 has a new owner.",
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("encrypted browser push accepted")
	// encrypted browser push accepted
	// snippet:end
}
