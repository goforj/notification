// Command apns sends a native HTTP/2 alert to a local TLS protocol fixture.
package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/goforj/notification"
	"github.com/goforj/notification/driver/apns"
)

// main supplies ephemeral credentials so the runnable example never contacts Apple.
func main() {
	service := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Header().Set("Apns-Id", "request-42") }))
	service.EnableHTTP2 = true
	service.StartTLS()
	defer service.Close()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(err)
	}
	// snippet:start
	driver, err := apns.New(apns.Config{
		Key:         key,
		KeyID:       "KEY0000042",
		TeamID:      "TEAM000042",
		Topic:       "example.app",
		DeviceToken: strings.Repeat("ab", 32),
		BaseURL:     service.URL,
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
	fmt.Println("APNs accepted over HTTP/2")
	// APNs accepted over HTTP/2
	// snippet:end
}
