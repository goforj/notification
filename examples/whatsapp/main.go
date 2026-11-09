// Command whatsapp sends an approved template to a local Meta Cloud API protocol fixture.
package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"

	"github.com/goforj/notification"
	"github.com/goforj/notification/driver/whatsapp"
)

// main keeps template provisioning and customer permissions outside the notification transport.
func main() {
	var accepted atomic.Int64
	service := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accepted.Add(1)
		_, _ = io.WriteString(w, `{"contacts":[{"input":"12125550101","wa_id":"12125550101"}],"messages":[{"id":"wamid.fixture"}]}`)
	}))
	defer service.Close()
	// snippet:start
	driver, err := whatsapp.New(whatsapp.Config{
		Token:         "local-fixture-token",
		PhoneNumberID: "123456789",
		To:            "+12125550101",
		APIVersion:    "v24.0",
		BaseURL:       service.URL,
		HTTP: notification.HTTPConfig{
			AllowPrivate: true,
		},
	})
	if err != nil {
		panic(err)
	}
	defer driver.Close()
	err = driver.SendTemplate(context.Background(), whatsapp.Template{
		Name:     "ticket_updated",
		Language: "en_US",
		BodyParameters: []string{
			"T-42",
		},
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("accepted WhatsApp template:", accepted.Load())
	// accepted WhatsApp template: 1
	// snippet:end
}
