package driver_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/goforj/notification"
	"github.com/goforj/notification/driver/apprise"
	"github.com/goforj/notification/driver/clicksend"
	"github.com/goforj/notification/driver/homeassistant"
	"github.com/goforj/notification/driver/ilert"
	"github.com/goforj/notification/driver/nextcloudtalk"
	"github.com/goforj/notification/driver/plivo"
	"github.com/goforj/notification/driver/signal"
	"github.com/goforj/notification/driver/telnyx"
	"github.com/goforj/notification/driver/vonage"
)

// TestRelayAndSMSBoundaries checks explicit relay acceptance and provider-specific size limits.
func TestRelayAndSMSBoundaries(t *testing.T) {
	driver, err := apprise.New(apprise.Config{URL: "https://example.com", Key: "fixture", HTTP: notification.HTTPConfig{Transport: responseTransport(`{}`)}})
	if err != nil {
		t.Fatal(err)
	}
	defer driver.Close()
	if err := driver.Send(nil, notification.Message{Body: "relay request"}); !errors.Is(err, notification.ErrOutcomeUnknown) {
		t.Fatal("unexpected relay success status accepted", err)
	}
	if _, err := apprise.New(apprise.Config{URL: "https://example.com", Key: "fixture", Username: "without-password"}); err == nil {
		t.Fatal("incomplete relay authentication")
	}
	for _, open := range []func() (nativeSender, error){
		func() (nativeSender, error) {
			return homeassistant.New(homeassistant.Config{URL: "https://example.com", Token: "fixture"})
		},
		func() (nativeSender, error) {
			return signal.New(signal.Config{URL: "https://example.com", Number: "+12125550100", Recipient: "group.fixture"})
		},
		func() (nativeSender, error) {
			return nextcloudtalk.New(nextcloudtalk.Config{URL: "https://example.com", Username: "fixture", Password: "secret", RoomToken: "room"})
		},
		func() (nativeSender, error) { return ilert.New(ilert.Config{IntegrationKey: "fixture"}) },
	} {
		driver, err := open()
		if err != nil {
			t.Fatal(err)
		}
		_ = driver.Close()
	}
	for _, open := range []func() (nativeSender, error){
		func() (nativeSender, error) {
			return plivo.New(plivo.Config{AuthID: "fixture", AuthToken: "secret", From: "+12125550100", To: "+12125550101"})
		},
		func() (nativeSender, error) {
			return vonage.New(vonage.Config{APIKey: "fixture1", APISecret: "secret", From: "Helpdesk", To: "+12125550101"})
		},
		func() (nativeSender, error) {
			return telnyx.New(telnyx.Config{Token: "fixture", From: "+12125550100", To: "+12125550101"})
		},
		func() (nativeSender, error) {
			return clicksend.New(clicksend.Config{Username: "fixture", APIKey: "secret", To: "+12125550101"})
		},
	} {
		driver, err := open()
		if err != nil {
			t.Fatal(err)
		}
		if err := driver.Send(nil, notification.Message{Body: strings.Repeat("界", 1601)}); !errors.Is(err, notification.ErrRender) {
			t.Fatal("SMS bound did not reject without I/O", err)
		}
		_ = driver.Close()
	}
}
