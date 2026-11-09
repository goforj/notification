// Command transport supplies a caller-owned HTTP transport with an explicit trust store.
package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/goforj/notification"
	"github.com/goforj/notification/driver/slack"
)

// main uses a local TLS fixture so the example requires no hosted credentials.
func main() {
	provider := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, "ok") }))
	defer provider.Close()
	roots := x509.NewCertPool()
	roots.AddCert(provider.Certificate())

	// snippet:start
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs:    roots,
			MinVersion: tls.VersionTLS12,
		},
		ForceAttemptHTTP2: true,
	}
	defer transport.CloseIdleConnections()

	driver, err := slack.New(slack.Config{
		URL: provider.URL,
		HTTP: notification.HTTPConfig{
			Transport: transport,
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
	fmt.Println("accepted through custom transport")
	// accepted through custom transport
	// snippet:end
}
