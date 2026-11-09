package driver_test

import (
	"context"
	"errors"
	"github.com/goforj/notification"
	"github.com/goforj/notification/driver/apprise"
	"github.com/goforj/notification/driver/bark"
	"github.com/goforj/notification/driver/betterstack"
	"github.com/goforj/notification/driver/clicksend"
	"github.com/goforj/notification/driver/expo"
	"github.com/goforj/notification/driver/fcm"
	"github.com/goforj/notification/driver/firehydrant"
	"github.com/goforj/notification/driver/goalert"
	"github.com/goforj/notification/driver/homeassistant"
	"github.com/goforj/notification/driver/ifttt"
	"github.com/goforj/notification/driver/ilert"
	"github.com/goforj/notification/driver/incidentio"
	"github.com/goforj/notification/driver/join"
	"github.com/goforj/notification/driver/matrix"
	"github.com/goforj/notification/driver/mattermost"
	"github.com/goforj/notification/driver/nextcloudtalk"
	"github.com/goforj/notification/driver/onesignal"
	"github.com/goforj/notification/driver/pagertree"
	"github.com/goforj/notification/driver/plivo"
	"github.com/goforj/notification/driver/pushbullet"
	"github.com/goforj/notification/driver/pushover"
	"github.com/goforj/notification/driver/rocketchat"
	"github.com/goforj/notification/driver/signal"
	"github.com/goforj/notification/driver/signl4"
	"github.com/goforj/notification/driver/telnyx"
	"github.com/goforj/notification/driver/vonage"
	"github.com/goforj/notification/driver/whatsapp"
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
		{"homeassistant", func(base string, h notification.HTTPConfig) (nativeSender, error) {
			return homeassistant.New(homeassistant.Config{URL: base, Token: "fixture", Service: "mobile_app_phone", HTTP: h})
		}, `[]`, []string{`null`, `{}`, `invalid`}},
		{"nextcloudtalk", func(base string, h notification.HTTPConfig) (nativeSender, error) {
			return nextcloudtalk.New(nextcloudtalk.Config{URL: base, Username: "fixture", Password: "secret", RoomToken: "room42", HTTP: h})
		}, `{"ocs":{"meta":{"statuscode":100},"data":{"id":42}}}`, []string{`{}`, `{"ocs":{"meta":{"statuscode":400}}}`, `{"ocs":{"meta":{"statuscode":100},"data":{}}}`, `{"ocs":{"meta":{"statuscode":100},"data":{"id":0}}}`}},
		{"signal", func(base string, h notification.HTTPConfig) (nativeSender, error) {
			return signal.New(signal.Config{URL: base, Number: "+12125550100", Recipient: "+12125550101", Token: "fixture", HTTP: h})
		}, `{"timestamp":"1700000000000"}`, []string{`{}`, `{"timestamp":0}`, `{"timestamp":-1}`, `{"timestamp":"invalid"}`}},
		{"apprise", func(base string, h notification.HTTPConfig) (nativeSender, error) {
			return apprise.New(apprise.Config{URL: base, Key: "fixture", Tag: "alerts", Username: "fixture", Password: "secret", HTTP: h})
		}, `{"status":"SUCCESS"}`, []string{}},
		{"plivo", func(base string, h notification.HTTPConfig) (nativeSender, error) {
			return plivo.New(plivo.Config{BaseURL: base, AuthID: "fixture", AuthToken: "secret", From: "+12125550100", To: "+12125550101", HTTP: h})
		}, `{"api_id":"request-42","message_uuid":["message-42"]}`, []string{`{}`, `{"api_id":"42","message_uuid":[]}`, `{"api_id":"42","message_uuid":[""]}`}},
		{"vonage", func(base string, h notification.HTTPConfig) (nativeSender, error) {
			return vonage.New(vonage.Config{BaseURL: base, APIKey: "fixture1", APISecret: "secret", From: "Helpdesk", To: "+12125550101", HTTP: h})
		}, `{"messages":[{"status":"0","message-id":"message-42"}]}`, []string{`{}`, `{"messages":[{}]}`, `{"messages":[{"status":"0"}]}`, `{"messages":[{"status":"1"}]}`, `{"messages":[{"status":"4"}]}`, `{"messages":[{"status":"5"}]}`, `{"messages":[{"status":"2"}]}`}},
		{"telnyx", func(base string, h notification.HTTPConfig) (nativeSender, error) {
			return telnyx.New(telnyx.Config{BaseURL: base, Token: "fixture", From: "+12125550100", To: "+12125550101", MessagingProfileID: "profile", HTTP: h})
		}, `{"data":{"id":"message-42","to":[{"phone_number":"+12125550101","status":"queued"}]}}`, []string{`{}`, `{"data":{"id":"42","to":[]}}`, `{"data":{"id":"42","to":[{"phone_number":"wrong","status":"queued"}]}}`, `{"data":{"id":"42","to":[{"phone_number":"+12125550101","status":"failed"}]}}`, `{"data":{"id":"42","to":[{"phone_number":"+12125550101","status":"unknown"}]}}`}},
		{"clicksend", func(base string, h notification.HTTPConfig) (nativeSender, error) {
			return clicksend.New(clicksend.Config{BaseURL: base, Username: "fixture", APIKey: "secret", From: "Helpdesk", To: "+12125550101", HTTP: h})
		}, `{"response_code":"SUCCESS","data":{"messages":[{"status":"SUCCESS","message_id":"message-42"}]}}`, []string{`{}`, `{"response_code":"FAILED"}`, `{"response_code":"SUCCESS","data":{"messages":[]}}`, `{"response_code":"SUCCESS","data":{"messages":[{"status":"FAILED"}]}}`, `{"response_code":"SUCCESS","data":{"messages":[{"status":"SUCCESS"}]}}`}},
		{"goalert", func(base string, h notification.HTTPConfig) (nativeSender, error) {
			return goalert.New(goalert.Config{URL: base, Token: "fixture", Key: "monitor/42", HTTP: h})
		}, `{}`, []string{}},
		{"signl4", func(base string, h notification.HTTPConfig) (nativeSender, error) {
			return signl4.New(signl4.Config{URL: base, Key: "monitor/42", HTTP: h})
		}, `{}`, []string{}},
		{"pagertree", func(base string, h notification.HTTPConfig) (nativeSender, error) {
			return pagertree.New(pagertree.Config{URL: base, Key: "monitor/42", HTTP: h})
		}, `{}`, []string{}},
		{"ilert", func(base string, h notification.HTTPConfig) (nativeSender, error) {
			return ilert.New(ilert.Config{URL: base, IntegrationKey: "fixture", Key: "monitor/42", HTTP: h})
		}, `{}`, []string{}},
		{"incidentio", func(base string, h notification.HTTPConfig) (nativeSender, error) {
			return incidentio.New(incidentio.Config{BaseURL: base, Token: "fixture", SourceID: "source", Key: "monitor/42", HTTP: h})
		}, `{"status":"success","deduplication_key":"monitor/42"}`, []string{`{}`, `{"status":"failure","deduplication_key":"monitor/42"}`, `{"status":"success","deduplication_key":""}`}},
		{"firehydrant", func(base string, h notification.HTTPConfig) (nativeSender, error) {
			return firehydrant.New(firehydrant.Config{URL: base, Key: "monitor/42", HTTP: h})
		}, `{}`, []string{}},
		{"betterstack", func(base string, h notification.HTTPConfig) (nativeSender, error) {
			return betterstack.New(betterstack.Config{BaseURL: base, Token: "fixture", RequesterEmail: "user@example.com", PolicyID: "policy", TeamName: "Helpdesk", Email: true, HTTP: h})
		}, `{"data":{"id":"42","type":"incident"}}`, []string{`{}`, `{"data":{"id":"42","type":"unknown"}}`}},
		{"fcm", func(base string, h notification.HTTPConfig) (nativeSender, error) {
			return fcm.New(fcm.Config{BaseURL: base, ProjectID: "project", DeviceToken: "device", TokenSource: func(context.Context) (string, error) { return "fixture", nil }, HTTP: h})
		}, `{"name":"projects/project/messages/message-42"}`, []string{`{}`, `{"name":"projects/other/messages/42"}`, `{"name":"projects/project/messages/"}`}},
		{"onesignal", func(base string, h notification.HTTPConfig) (nativeSender, error) {
			return onesignal.New(onesignal.Config{BaseURL: base, AppID: "application", Token: "fixture", SubscriptionID: "device", HTTP: h})
		}, `{"id":"message-42"}`, []string{`{}`, `{"id":""}`, `{"id":"42","errors":["invalid recipient"]}`}},
		{"expo", func(base string, h notification.HTTPConfig) (nativeSender, error) {
			return expo.New(expo.Config{BaseURL: base, PushToken: "ExpoPushToken[fixture]", AccessToken: "fixture", HTTP: h})
		}, `{"data":{"status":"ok","id":"ticket-42"}}`, []string{`{}`, `{"data":{"status":"ok"}}`, `{"data":{"status":"error","details":{"error":"InvalidCredentials"}}}`, `{"data":{"status":"error","details":{"error":"MessageRateExceeded"}}}`, `{"data":{"status":"error","details":{"error":"MessageTooBig"}}}`, `{"data":{"status":"error","details":{"error":"DeviceNotRegistered"}}}`}},
		{"whatsapp", func(base string, h notification.HTTPConfig) (nativeSender, error) {
			return whatsapp.New(whatsapp.Config{BaseURL: base, Token: "fixture", PhoneNumberID: "123456789", To: "+12125550101", APIVersion: "v24.0", HTTP: h})
		}, `{"contacts":[{"input":"12125550101","wa_id":"12125550101"}],"messages":[{"id":"wamid.fixture"}]}`, []string{`{}`, `{"contacts":[{"input":"other","wa_id":"42"}],"messages":[{"id":"wamid.fixture"}]}`, `{"contacts":[{"input":"12125550101","wa_id":"42"}],"messages":[{"id":"wrong"}]}`}},
	}
	for _, entry := range cases {
		t.Run(entry.name, func(t *testing.T) {
			var mu sync.Mutex
			acknowledgement, status := entry.ack, 200
			if entry.name == "ilert" || entry.name == "incidentio" {
				status = 202
			}
			if entry.name == "betterstack" {
				status = 201
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Error(err)
				}
				inspectAdditionalProtocol(t, entry.name, r, body)
				switch entry.name {
				case "matrix":
					if r.Method != "PUT" || !strings.Contains(r.URL.EscapedPath(), "/send/m.room.message/") || r.Header.Get("Authorization") != "Bearer fixture" {
						t.Error("Matrix request contract")
					}
				case "join":
					if r.Method != "GET" || r.URL.Query().Get("apikey") != "fixture" || r.URL.Query().Get("text") == "" {
						t.Error("Join request contract")
					}
				case "pushover", "zulip", "vonage":
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
				if entry.name == "apprise" || entry.name == "goalert" || entry.name == "signl4" || entry.name == "pagertree" || entry.name == "ilert" || entry.name == "firehydrant" {
					break
				}
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
		{homeassistant.Config{URL: "https://example.com", Token: "fixture", Service: "mobile_app_phone", HTTP: notification.HTTPConfig{}}, func(value any) (nativeSender, error) { return homeassistant.New(value.(homeassistant.Config)) }},
		{nextcloudtalk.Config{URL: "https://example.com", Username: "fixture", Password: "secret", RoomToken: "room42", HTTP: notification.HTTPConfig{}}, func(value any) (nativeSender, error) { return nextcloudtalk.New(value.(nextcloudtalk.Config)) }},
		{signal.Config{URL: "https://example.com", Number: "+12125550100", Recipient: "+12125550101", Token: "fixture", HTTP: notification.HTTPConfig{}}, func(value any) (nativeSender, error) { return signal.New(value.(signal.Config)) }},
		{apprise.Config{URL: "https://example.com", Key: "fixture", Tag: "alerts", Username: "fixture", Password: "secret", HTTP: notification.HTTPConfig{}}, func(value any) (nativeSender, error) { return apprise.New(value.(apprise.Config)) }},
		{plivo.Config{BaseURL: "https://example.com", AuthID: "fixture", AuthToken: "secret", From: "+12125550100", To: "+12125550101", HTTP: notification.HTTPConfig{}}, func(value any) (nativeSender, error) { return plivo.New(value.(plivo.Config)) }},
		{vonage.Config{BaseURL: "https://example.com", APIKey: "fixture1", APISecret: "secret", From: "Helpdesk", To: "+12125550101", HTTP: notification.HTTPConfig{}}, func(value any) (nativeSender, error) { return vonage.New(value.(vonage.Config)) }},
		{telnyx.Config{BaseURL: "https://example.com", Token: "fixture", From: "+12125550100", To: "+12125550101", MessagingProfileID: "profile", HTTP: notification.HTTPConfig{}}, func(value any) (nativeSender, error) { return telnyx.New(value.(telnyx.Config)) }},
		{clicksend.Config{BaseURL: "https://example.com", Username: "fixture", APIKey: "secret", From: "Helpdesk", To: "+12125550101", HTTP: notification.HTTPConfig{}}, func(value any) (nativeSender, error) { return clicksend.New(value.(clicksend.Config)) }},
		{goalert.Config{URL: "https://example.com", Token: "fixture", Key: "monitor/42", HTTP: notification.HTTPConfig{}}, func(value any) (nativeSender, error) { return goalert.New(value.(goalert.Config)) }},
		{signl4.Config{URL: "https://example.com", Key: "monitor/42", HTTP: notification.HTTPConfig{}}, func(value any) (nativeSender, error) { return signl4.New(value.(signl4.Config)) }},
		{pagertree.Config{URL: "https://example.com", Key: "monitor/42", HTTP: notification.HTTPConfig{}}, func(value any) (nativeSender, error) { return pagertree.New(value.(pagertree.Config)) }},
		{ilert.Config{URL: "https://example.com", IntegrationKey: "fixture", Key: "monitor/42", HTTP: notification.HTTPConfig{}}, func(value any) (nativeSender, error) { return ilert.New(value.(ilert.Config)) }},
		{incidentio.Config{BaseURL: "https://example.com", Token: "fixture", SourceID: "source", Key: "monitor/42", HTTP: notification.HTTPConfig{}}, func(value any) (nativeSender, error) { return incidentio.New(value.(incidentio.Config)) }},
		{firehydrant.Config{URL: "https://example.com", Key: "monitor/42", HTTP: notification.HTTPConfig{}}, func(value any) (nativeSender, error) { return firehydrant.New(value.(firehydrant.Config)) }},
		{betterstack.Config{BaseURL: "https://example.com", Token: "fixture", RequesterEmail: "user@example.com", PolicyID: "policy", TeamName: "Helpdesk", Email: true, HTTP: notification.HTTPConfig{}}, func(value any) (nativeSender, error) { return betterstack.New(value.(betterstack.Config)) }},
		{fcm.Config{BaseURL: "https://example.com", ProjectID: "project", DeviceToken: "device", TokenSource: func(context.Context) (string, error) { return "fixture", nil }, HTTP: notification.HTTPConfig{}}, func(value any) (nativeSender, error) { return fcm.New(value.(fcm.Config)) }},
		{onesignal.Config{BaseURL: "https://example.com", AppID: "application", Token: "fixture", SubscriptionID: "device", HTTP: notification.HTTPConfig{}}, func(value any) (nativeSender, error) { return onesignal.New(value.(onesignal.Config)) }},
		{expo.Config{BaseURL: "https://example.com", PushToken: "ExpoPushToken[fixture]", AccessToken: "fixture", HTTP: notification.HTTPConfig{}}, func(value any) (nativeSender, error) { return expo.New(value.(expo.Config)) }},
		{whatsapp.Config{BaseURL: "https://example.com", Token: "fixture", PhoneNumberID: "123456789", To: "+12125550101", APIVersion: "v24.0", HTTP: notification.HTTPConfig{}}, func(value any) (nativeSender, error) { return whatsapp.New(value.(whatsapp.Config)) }},
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
