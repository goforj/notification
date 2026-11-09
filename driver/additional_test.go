package driver_test

import (
	"errors"
	"github.com/goforj/notification"
	"github.com/goforj/notification/driver/bark"
	"github.com/goforj/notification/driver/ifttt"
	"github.com/goforj/notification/driver/join"
	"github.com/goforj/notification/driver/matrix"
	"github.com/goforj/notification/driver/mattermost"
	"github.com/goforj/notification/driver/pushbullet"
	"github.com/goforj/notification/driver/pushover"
	"github.com/goforj/notification/driver/rocketchat"
	"github.com/goforj/notification/driver/zulip"
	"github.com/goforj/notification/notificationtest"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
)

// TestAdditionalProviders exercises native protocols through owned and borrowed transports.
func TestAdditionalProviders(t *testing.T) {
	cases := []struct {
		name string
		new  func(string, notification.HTTPConfig) (nativeSender, error)
		ack  string
		bad  []string
	}{
		{"mattermost", func(base string, h notification.HTTPConfig) (nativeSender, error) {
			return mattermost.New(mattermost.Config{URL: base, HTTP: h})
		}, "ok", []string{"", "not ok"}},
		{"rocketchat", func(base string, h notification.HTTPConfig) (nativeSender, error) {
			return rocketchat.New(rocketchat.Config{URL: base, HTTP: h})
		}, `{"success":true}`, []string{`{}`, `null`, `{"success":false}`, `{"success":"yes"}`}},
		{"bark", func(base string, h notification.HTTPConfig) (nativeSender, error) {
			return bark.New(bark.Config{URL: base, DeviceKey: "fixture", Group: "alerts", HTTP: h})
		}, `{"code":200}`, []string{`{}`, `{"code":400}`, `{"code":"200"}`}},
		{"pushbullet", func(base string, h notification.HTTPConfig) (nativeSender, error) {
			return pushbullet.New(pushbullet.Config{BaseURL: base, Token: "fixture", DeviceID: "device", HTTP: h})
		}, `{"iden":"push-42"}`, []string{`{}`, `{"iden":42}`}},
		{"pushover", func(base string, h notification.HTTPConfig) (nativeSender, error) {
			return pushover.New(pushover.Config{BaseURL: base, Token: "fixture", User: "user", Device: "phone", HTTP: h})
		}, `{"status":1,"request":"request-42"}`, []string{`{}`, `{"status":0}`, `{"status":1}`, `{"status":"1"}`}},
		{"matrix", func(base string, h notification.HTTPConfig) (nativeSender, error) {
			return matrix.New(matrix.Config{URL: base, Token: "fixture", RoomID: "!room:example.com", HTTP: h})
		}, `{"event_id":"$event-42"}`, []string{`{}`, `{"event_id":"wrong"}`, `{"event_id":42}`}},
		{"zulip", func(base string, h notification.HTTPConfig) (nativeSender, error) {
			return zulip.New(zulip.Config{URL: base, Email: "bot@example.com", APIKey: "fixture", Channel: "alerts", Topic: "Tickets", HTTP: h})
		}, `{"result":"success","id":42}`, []string{`{}`, `{"result":"error"}`, `{"result":"success"}`, `{"result":"success","id":0}`}},
		{"ifttt", func(base string, h notification.HTTPConfig) (nativeSender, error) {
			return ifttt.New(ifttt.Config{BaseURL: base, Key: "fixture", Event: "ticket_updated", HTTP: h})
		}, "Congratulations! You have fired the event", []string{"", "not accepted"}},
		{"join", func(base string, h notification.HTTPConfig) (nativeSender, error) {
			return join.New(join.Config{BaseURL: base, APIKey: "fixture", DeviceID: "device", HTTP: h})
		}, `{"success":true}`, []string{`{}`, `{"success":false}`, `{"success":"yes"}`}},
	}
	for _, entry := range cases {
		t.Run(entry.name, func(t *testing.T) {
			var mu sync.Mutex
			acknowledgement, status := entry.ack, 200
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Error(err)
				}
				switch entry.name {
				case "matrix":
					if r.Method != "PUT" || !strings.Contains(r.URL.EscapedPath(), "/send/m.room.message/") || r.Header.Get("Authorization") != "Bearer fixture" {
						t.Error("Matrix request contract")
					}
				case "join":
					if r.Method != "GET" || r.URL.Query().Get("apikey") != "fixture" || r.URL.Query().Get("text") == "" {
						t.Error("Join request contract")
					}
				case "pushover", "zulip":
					if r.Method != "POST" || r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" || len(body) == 0 {
						t.Error("form request contract")
					}
				default:
					if r.Method != "POST" || r.Header.Get("Content-Type") != "application/json" || !strings.Contains(string(body), "T-42") {
						t.Error("JSON request contract")
					}
				}
				mu.Lock()
				ack, code := acknowledgement, status
				mu.Unlock()
				w.WriteHeader(code)
				_, _ = io.WriteString(w, ack)
			}))
			defer server.Close()
			driver, err := entry.new(server.URL, notification.HTTPConfig{AllowPrivate: true})
			if err != nil {
				t.Fatal(err)
			}
			defer driver.Close()
			notificationtest.Sender(t, func() notification.Sender { return driver })
			if err := driver.Send(nil, notification.Message{Title: "Ticket", Body: "T-42 changed"}); err != nil {
				t.Fatal(err)
			}
			base := &http.Transport{}
			defer base.CloseIdleConnections()
			borrowed := &borrowedTransport{base: base}
			other, err := entry.new(server.URL, notification.HTTPConfig{Transport: borrowed})
			if err != nil {
				t.Fatal(err)
			}
			notificationtest.Sender(t, func() notification.Sender { return other })
			if err := other.Close(); err != nil || borrowed.closes.Load() != 0 {
				t.Fatal("borrowed transport ownership", err)
			}
			if _, err := entry.new(server.URL, notification.HTTPConfig{Timeout: -1}); err == nil {
				t.Fatal("invalid HTTP policy accepted")
			}
			for _, bad := range append(entry.bad, "not json", strings.Repeat("x", 65537)) {
				mu.Lock()
				acknowledgement = bad
				mu.Unlock()
				if err := driver.Send(nil, notification.Message{Title: "Ticket", Body: "T-42 changed"}); err == nil {
					t.Fatal("bad acknowledgement accepted")
				}
			}
			mu.Lock()
			status = 503
			acknowledgement = "private provider prose"
			mu.Unlock()
			err = driver.Send(nil, notification.Message{Title: "Ticket", Body: "T-42 changed"})
			if !errors.Is(err, notification.ErrRetryable) || strings.Contains(err.Error(), "private") {
				t.Fatal("provider failure evidence", err)
			}
		})
	}
}

// TestAdditionalConfiguration rejects malformed destinations and credentials without I/O.
func TestAdditionalConfiguration(t *testing.T) {
	cases := []struct {
		config any
		new    func(any) (nativeSender, error)
	}{
		{mattermost.Config{URL: "https://example.com", HTTP: notification.HTTPConfig{}}, func(value any) (nativeSender, error) { return mattermost.New(value.(mattermost.Config)) }},
		{rocketchat.Config{URL: "https://example.com", HTTP: notification.HTTPConfig{}}, func(value any) (nativeSender, error) { return rocketchat.New(value.(rocketchat.Config)) }},
		{bark.Config{URL: "https://example.com", DeviceKey: "fixture", Group: "alerts", HTTP: notification.HTTPConfig{}}, func(value any) (nativeSender, error) { return bark.New(value.(bark.Config)) }},
		{pushbullet.Config{BaseURL: "https://example.com", Token: "fixture", DeviceID: "device", HTTP: notification.HTTPConfig{}}, func(value any) (nativeSender, error) { return pushbullet.New(value.(pushbullet.Config)) }},
		{pushover.Config{BaseURL: "https://example.com", Token: "fixture", User: "user", Device: "phone", HTTP: notification.HTTPConfig{}}, func(value any) (nativeSender, error) { return pushover.New(value.(pushover.Config)) }},
		{matrix.Config{URL: "https://example.com", Token: "fixture", RoomID: "!room:example.com", HTTP: notification.HTTPConfig{}}, func(value any) (nativeSender, error) { return matrix.New(value.(matrix.Config)) }},
		{zulip.Config{URL: "https://example.com", Email: "bot@example.com", APIKey: "fixture", Channel: "alerts", Topic: "Tickets", HTTP: notification.HTTPConfig{}}, func(value any) (nativeSender, error) { return zulip.New(value.(zulip.Config)) }},
		{ifttt.Config{BaseURL: "https://example.com", Key: "fixture", Event: "ticket_updated", HTTP: notification.HTTPConfig{}}, func(value any) (nativeSender, error) { return ifttt.New(value.(ifttt.Config)) }},
		{join.Config{BaseURL: "https://example.com", APIKey: "fixture", DeviceID: "device", HTTP: notification.HTTPConfig{}}, func(value any) (nativeSender, error) { return join.New(value.(join.Config)) }},
	}
	for _, entry := range cases {
		value := reflect.ValueOf(entry.config)
		for index := 0; index < value.NumField(); index++ {
			if value.Field(index).Kind() != reflect.String {
				continue
			}
			invalid := reflect.New(value.Type()).Elem()
			invalid.Set(value)
			invalid.Field(index).SetString("bad\n")
			if driver, err := entry.new(invalid.Interface()); err == nil {
				_ = driver.Close()
				t.Fatalf("invalid %s.%s", value.Type(), value.Type().Field(index).Name)
			}
		}
		driver, err := entry.new(entry.config)
		if err != nil {
			t.Fatal(err)
		}
		_ = driver.Close()
	}
	for _, config := range []pushover.Config{
		{Token: "fixture", User: "user", Priority: -3},
		{Token: "fixture", User: "user", Priority: 2},
	} {
		if _, err := pushover.New(config); err == nil {
			t.Fatal("unsupported Pushover priority")
		}
	}
	push, err := pushover.New(pushover.Config{Token: "fixture", User: "user"})
	if err != nil {
		t.Fatal(err)
	}
	defer push.Close()
	for _, message := range []notification.Message{{Title: strings.Repeat("x", 251), Body: "body"}, {Body: strings.Repeat("x", 1025)}} {
		if err := push.Send(nil, message); !errors.Is(err, notification.ErrRender) {
			t.Fatal("Pushover bounds", err)
		}
	}
	if _, err := matrix.New(matrix.Config{URL: "https://example.com", Token: "fixture", RoomID: "not-a-room"}); err == nil {
		t.Fatal("invalid Matrix room")
	}
	if _, err := pushbullet.New(pushbullet.Config{Token: "fixture", DeviceID: "device", Email: "user@example.com"}); err == nil {
		t.Fatal("ambiguous Pushbullet destination")
	}
	for _, open := range []func() (nativeSender, error){
		func() (nativeSender, error) { return bark.New(bark.Config{DeviceKey: "fixture"}) },
		func() (nativeSender, error) { return pushbullet.New(pushbullet.Config{Token: "fixture"}) },
		func() (nativeSender, error) { return ifttt.New(ifttt.Config{Key: "fixture", Event: "event"}) },
		func() (nativeSender, error) { return join.New(join.Config{APIKey: "fixture", DeviceID: "device"}) },
	} {
		driver, err := open()
		if err != nil {
			t.Fatal(err)
		}
		_ = driver.Close()
	}
}
