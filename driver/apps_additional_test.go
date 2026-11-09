package driver_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/goforj/notification"
	"github.com/goforj/notification/driver/expo"
	"github.com/goforj/notification/driver/fcm"
	"github.com/goforj/notification/driver/join"
	"github.com/goforj/notification/driver/matrix"
	"github.com/goforj/notification/driver/onesignal"
	"github.com/goforj/notification/driver/whatsapp"
)

// TestApplicationPushBoundaries exercises refreshing credentials, alternative targets and wire limits.
func TestApplicationPushBoundaries(t *testing.T) {
	config := fcm.Config{ProjectID: "project", Topic: "alerts", HTTP: notification.HTTPConfig{Transport: responseTransport(`{"name":"projects/project/messages/42"}`)}}
	calls := 0
	config.TokenSource = func(context.Context) (string, error) { calls++; return "", errors.New("private credential error") }
	driver, err := fcm.New(config)
	if err != nil {
		t.Fatal(err)
	}
	if err := driver.Send(nil, notification.Message{Body: "body"}); !errors.Is(err, notification.ErrRetryable) || strings.Contains(err.Error(), "private") {
		t.Fatal("credential failure classification", err)
	}
	_ = driver.Close()
	config.TokenSource = func(context.Context) (string, error) { return "bad\n", nil }
	driver, err = fcm.New(config)
	if err != nil {
		t.Fatal(err)
	}
	if err := driver.Send(nil, notification.Message{Body: "body"}); !errors.Is(err, notification.ErrUnauthorized) {
		t.Fatal("invalid bearer token", err)
	}
	_ = driver.Close()
	ctx, cancel := context.WithCancel(context.Background())
	config.TokenSource = func(context.Context) (string, error) { cancel(); return "", context.Canceled }
	driver, err = fcm.New(config)
	if err != nil {
		t.Fatal(err)
	}
	if err := driver.Send(ctx, notification.Message{Body: "body"}); !errors.Is(err, context.Canceled) {
		t.Fatal("credential cancellation", err)
	}
	_ = driver.Close()
	config.TokenSource = func(context.Context) (string, error) { return "fixture", nil }
	driver, err = fcm.New(config)
	if err != nil {
		t.Fatal(err)
	}
	if err := driver.Send(nil, notification.Message{Body: "body"}); err != nil {
		t.Fatal(err)
	}
	if err := driver.Send(nil, notification.Message{Body: strings.Repeat("x", 4096)}); !errors.Is(err, notification.ErrRender) {
		t.Fatal("FCM bound", err)
	}
	_ = driver.Close()
	if calls != 1 {
		t.Fatal("unexpected credential calls")
	}
	signal, err := onesignal.New(onesignal.Config{AppID: "application", Token: "fixture", ExternalID: "user-42", HTTP: notification.HTTPConfig{Transport: responseTransport(`{"id":"42","errors":[]}`)}})
	if err != nil {
		t.Fatal(err)
	}
	if err := signal.Send(nil, notification.Message{Body: "body"}); err != nil {
		t.Fatal(err)
	}
	if err := signal.Send(nil, notification.Message{Body: strings.Repeat("x", 4096)}); !errors.Is(err, notification.ErrRender) {
		t.Fatal("OneSignal bound", err)
	}
	_ = signal.Close()
	push, err := expo.New(expo.Config{PushToken: "ExponentPushToken[fixture]"})
	if err != nil {
		t.Fatal(err)
	}
	if err := push.Send(nil, notification.Message{Body: strings.Repeat("x", 4096)}); !errors.Is(err, notification.ErrRender) {
		t.Fatal("Expo bound", err)
	}
	_ = push.Close()
	if _, err := fcm.New(fcm.Config{ProjectID: "project"}); err == nil {
		t.Fatal("missing FCM destination")
	}
	if _, err := onesignal.New(onesignal.Config{AppID: "application", Token: "fixture"}); err == nil {
		t.Fatal("missing OneSignal destination")
	}
}

// TestWhatsAppTemplates verifies approved-template structure independently of free-form text.
func TestWhatsAppTemplates(t *testing.T) {
	captures := make(chan map[string]json.RawMessage, 4)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var value map[string]json.RawMessage
		if json.NewDecoder(r.Body).Decode(&value) != nil {
			t.Error("invalid native JSON")
		}
		captures <- value
		_, _ = io.WriteString(w, `{"contacts":[{"input":"+12125550101","wa_id":"12125550101"}],"messages":[{"id":"wamid.fixture"}]}`)
	}))
	defer server.Close()
	driver, err := whatsapp.New(whatsapp.Config{Token: "fixture", PhoneNumberID: "123456789", To: "+12125550101", APIVersion: "v24.0", BaseURL: server.URL, HTTP: notification.HTTPConfig{AllowPrivate: true}})
	if err != nil {
		t.Fatal(err)
	}
	defer driver.Close()
	for _, parameters := range [][]string{nil, {"T-42"}} {
		if err := driver.SendTemplate(nil, whatsapp.Template{Name: "ticket_updated", Language: "en_US", BodyParameters: parameters}); err != nil {
			t.Fatal(err)
		}
		value := <-captures
		if string(value["type"]) != `"template"` || !strings.Contains(string(value["template"]), "ticket_updated") {
			t.Fatal("template native contract")
		}
		if (len(parameters) > 0) != strings.Contains(string(value["template"]), "components") {
			t.Fatal("template parameters changed")
		}
	}
	for _, template := range []whatsapp.Template{
		{}, {Name: "bad name", Language: "en"}, {Name: "valid", Language: strings.Repeat("x", 65)},
		{Name: "valid", Language: "en", BodyParameters: make([]string, 33)},
		{Name: "valid", Language: "en", BodyParameters: []string{string([]byte{255})}},
		{Name: "valid", Language: "en", BodyParameters: []string{strings.Repeat("x", 4097)}},
		{Name: "valid", Language: "en", BodyParameters: []string{strings.Repeat("x", 4096), strings.Repeat("x", 4096), strings.Repeat("x", 4096), strings.Repeat("x", 4096), "overflow"}},
	} {
		if err := driver.SendTemplate(nil, template); err == nil {
			t.Fatal("invalid template accepted")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := driver.SendTemplate(ctx, whatsapp.Template{}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if err := driver.Send(nil, notification.Message{Body: strings.Repeat("界", 4097)}); !errors.Is(err, notification.ErrRender) {
		t.Fatal("WhatsApp text bound", err)
	}
}

// TestNativeNoRecipientAndURLLimits rejects explicit no-delivery and oversized GET envelopes.
func TestNativeNoRecipientAndURLLimits(t *testing.T) {
	signal, err := onesignal.New(onesignal.Config{AppID: "fixture", Token: "fixture", ExternalID: "user", HTTP: notification.HTTPConfig{Transport: responseTransport(`{"id":"42","recipients":0}`)}})
	if err != nil {
		t.Fatal(err)
	}
	defer signal.Close()
	if err := signal.Send(nil, notification.Message{Body: "body"}); !errors.Is(err, notification.ErrPermanent) {
		t.Fatal("zero recipients accepted", err)
	}
	relay, err := join.New(join.Config{APIKey: "fixture", DeviceID: "device"})
	if err != nil {
		t.Fatal(err)
	}
	defer relay.Close()
	if err := relay.Send(nil, notification.Message{Body: strings.Repeat("界", 2000)}); !errors.Is(err, notification.ErrRender) {
		t.Fatal("oversized Join URL", err)
	}
}

// TestModernMatrixRoomIDs preserves opaque room-version-12 IDs while rejecting empty or overlong IDs.
func TestModernMatrixRoomIDs(t *testing.T) {
	for _, id := range []string{"!opaque-room", "!legacy:example.com"} {
		driver, err := matrix.New(matrix.Config{URL: "https://example.com", Token: "fixture", RoomID: id})
		if err != nil {
			t.Fatal(id, err)
		}
		_ = driver.Close()
	}
	for _, id := range []string{"!", "#alias:example.com", "!" + strings.Repeat("x", 255)} {
		if _, err := matrix.New(matrix.Config{URL: "https://example.com", Token: "fixture", RoomID: id}); !errors.Is(err, notification.ErrInvalidRequest) {
			t.Fatal("invalid room accepted", err)
		}
	}
}
