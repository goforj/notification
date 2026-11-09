// Command providers demonstrates every standard-library provider added by the coverage review.
package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"

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
)

// configuredProvider binds a readable typed configuration to one local native protocol fixture.
type configuredProvider struct {
	name string
	open func() (notification.Sender, error)
}

// configured lists complete constructor examples while preserving provider-specific configuration.
func configured(base string, h notification.HTTPConfig) []configuredProvider {
	return []configuredProvider{
		{
			name: "mattermost",
			open: func() (notification.Sender, error) {
				return mattermost.New(mattermost.Config{
					URL:  base + "/mattermost",
					HTTP: h,
				})
			},
		},
		{
			name: "rocketchat",
			open: func() (notification.Sender, error) {
				return rocketchat.New(rocketchat.Config{
					URL:  base + "/rocketchat",
					HTTP: h,
				})
			},
		},
		{
			name: "bark",
			open: func() (notification.Sender, error) {
				return bark.New(bark.Config{
					URL:       base + "/bark",
					DeviceKey: "fixture",
					Group:     "alerts",
					HTTP:      h,
				})
			},
		},
		{
			name: "pushbullet",
			open: func() (notification.Sender, error) {
				return pushbullet.New(pushbullet.Config{
					BaseURL:  base + "/pushbullet",
					Token:    "fixture",
					DeviceID: "device",
					HTTP:     h,
				})
			},
		},
		{
			name: "pushover",
			open: func() (notification.Sender, error) {
				return pushover.New(pushover.Config{
					BaseURL: base + "/pushover",
					Token:   "fixture",
					User:    "user",
					Device:  "phone",
					HTTP:    h,
				})
			},
		},
		{
			name: "matrix",
			open: func() (notification.Sender, error) {
				return matrix.New(matrix.Config{
					URL:    base + "/matrix",
					Token:  "fixture",
					RoomID: "!room:example.com",
					HTTP:   h,
				})
			},
		},
		{
			name: "zulip",
			open: func() (notification.Sender, error) {
				return zulip.New(zulip.Config{
					URL:     base + "/zulip",
					Email:   "bot@example.com",
					APIKey:  "fixture",
					Channel: "alerts",
					Topic:   "Tickets",
					HTTP:    h,
				})
			},
		},
		{
			name: "ifttt",
			open: func() (notification.Sender, error) {
				return ifttt.New(ifttt.Config{
					BaseURL: base + "/ifttt",
					Key:     "fixture",
					Event:   "ticket_updated",
					HTTP:    h,
				})
			},
		},
		{
			name: "join",
			open: func() (notification.Sender, error) {
				return join.New(join.Config{
					BaseURL:  base + "/join",
					APIKey:   "fixture",
					DeviceID: "device",
					HTTP:     h,
				})
			},
		},
		{
			name: "homeassistant",
			open: func() (notification.Sender, error) {
				return homeassistant.New(homeassistant.Config{
					URL:     base + "/homeassistant",
					Token:   "fixture",
					Service: "mobile_app_phone",
					HTTP:    h,
				})
			},
		},
		{
			name: "nextcloudtalk",
			open: func() (notification.Sender, error) {
				return nextcloudtalk.New(nextcloudtalk.Config{
					URL:       base + "/nextcloudtalk",
					Username:  "fixture",
					Password:  "secret",
					RoomToken: "room42",
					HTTP:      h,
				})
			},
		},
		{
			name: "signal",
			open: func() (notification.Sender, error) {
				return signal.New(signal.Config{
					URL:       base + "/signal",
					Number:    "+12125550100",
					Recipient: "+12125550101",
					Token:     "fixture",
					HTTP:      h,
				})
			},
		},
		{
			name: "apprise",
			open: func() (notification.Sender, error) {
				return apprise.New(apprise.Config{
					URL:      base + "/apprise",
					Key:      "fixture",
					Tag:      "alerts",
					Username: "fixture",
					Password: "secret",
					HTTP:     h,
				})
			},
		},
		{
			name: "plivo",
			open: func() (notification.Sender, error) {
				return plivo.New(plivo.Config{
					BaseURL:   base + "/plivo",
					AuthID:    "fixture",
					AuthToken: "secret",
					From:      "+12125550100",
					To:        "+12125550101",
					HTTP:      h,
				})
			},
		},
		{
			name: "vonage",
			open: func() (notification.Sender, error) {
				return vonage.New(vonage.Config{
					BaseURL:   base + "/vonage",
					APIKey:    "fixture1",
					APISecret: "secret",
					From:      "Helpdesk",
					To:        "+12125550101",
					HTTP:      h,
				})
			},
		},
		{
			name: "telnyx",
			open: func() (notification.Sender, error) {
				return telnyx.New(telnyx.Config{
					BaseURL:            base + "/telnyx",
					Token:              "fixture",
					From:               "+12125550100",
					To:                 "+12125550101",
					MessagingProfileID: "profile",
					HTTP:               h,
				})
			},
		},
		{
			name: "clicksend",
			open: func() (notification.Sender, error) {
				return clicksend.New(clicksend.Config{
					BaseURL:  base + "/clicksend",
					Username: "fixture",
					APIKey:   "secret",
					From:     "Helpdesk",
					To:       "+12125550101",
					HTTP:     h,
				})
			},
		},
		{
			name: "goalert",
			open: func() (notification.Sender, error) {
				return goalert.New(goalert.Config{
					URL:   base + "/goalert",
					Token: "fixture",
					Key:   "monitor/42",
					HTTP:  h,
				})
			},
		},
		{
			name: "signl4",
			open: func() (notification.Sender, error) {
				return signl4.New(signl4.Config{
					URL:  base + "/signl4",
					Key:  "monitor/42",
					HTTP: h,
				})
			},
		},
		{
			name: "pagertree",
			open: func() (notification.Sender, error) {
				return pagertree.New(pagertree.Config{
					URL:  base + "/pagertree",
					Key:  "monitor/42",
					HTTP: h,
				})
			},
		},
		{
			name: "ilert",
			open: func() (notification.Sender, error) {
				return ilert.New(ilert.Config{
					URL:            base + "/ilert",
					IntegrationKey: "fixture",
					Key:            "monitor/42",
					HTTP:           h,
				})
			},
		},
		{
			name: "incidentio",
			open: func() (notification.Sender, error) {
				return incidentio.New(incidentio.Config{
					BaseURL:  base + "/incidentio",
					Token:    "fixture",
					SourceID: "source",
					Key:      "monitor/42",
					HTTP:     h,
				})
			},
		},
		{
			name: "firehydrant",
			open: func() (notification.Sender, error) {
				return firehydrant.New(firehydrant.Config{
					URL:  base + "/firehydrant",
					Key:  "monitor/42",
					HTTP: h,
				})
			},
		},
		{
			name: "betterstack",
			open: func() (notification.Sender, error) {
				return betterstack.New(betterstack.Config{
					BaseURL:        base + "/betterstack",
					Token:          "fixture",
					RequesterEmail: "user@example.com",
					PolicyID:       "policy",
					TeamName:       "Helpdesk",
					Email:          true,
					HTTP:           h,
				})
			},
		},
		{
			name: "fcm",
			open: func() (notification.Sender, error) {
				return fcm.New(fcm.Config{
					BaseURL:     base + "/fcm",
					ProjectID:   "project",
					DeviceToken: "device",
					TokenSource: func(context.Context) (string, error) { return "fixture", nil },
					HTTP:        h,
				})
			},
		},
		{
			name: "onesignal",
			open: func() (notification.Sender, error) {
				return onesignal.New(onesignal.Config{
					BaseURL:        base + "/onesignal",
					AppID:          "application",
					Token:          "fixture",
					SubscriptionID: "device",
					HTTP:           h,
				})
			},
		},
		{
			name: "expo",
			open: func() (notification.Sender, error) {
				return expo.New(expo.Config{
					BaseURL:     base + "/expo",
					PushToken:   "ExpoPushToken[fixture]",
					AccessToken: "fixture",
					HTTP:        h,
				})
			},
		},
		{
			name: "whatsapp",
			open: func() (notification.Sender, error) {
				return whatsapp.New(whatsapp.Config{
					BaseURL:       base + "/whatsapp",
					Token:         "fixture",
					PhoneNumberID: "123456789",
					To:            "+12125550101",
					APIVersion:    "v24.0",
					HTTP:          h,
				})
			},
		},
	}
}

// main verifies native acknowledgements without contacting a hosted provider.
func main() {
	acknowledgements := map[string]string{
		"mattermost":    "ok",
		"rocketchat":    `{"success":true}`,
		"bark":          `{"code":200}`,
		"pushbullet":    `{"iden":"push-42"}`,
		"pushover":      `{"status":1,"request":"request-42"}`,
		"matrix":        `{"event_id":"$event-42"}`,
		"zulip":         `{"result":"success","id":42}`,
		"ifttt":         "Congratulations! You have fired the event",
		"join":          `{"success":true}`,
		"homeassistant": `[]`,
		"nextcloudtalk": `{"ocs":{"meta":{"statuscode":100},"data":{"id":42}}}`,
		"signal":        `{"timestamp":"1700000000000"}`,
		"apprise":       `{"status":"SUCCESS"}`,
		"plivo":         `{"api_id":"request-42","message_uuid":["message-42"]}`,
		"vonage":        `{"messages":[{"status":"0","message-id":"message-42"}]}`,
		"telnyx":        `{"data":{"id":"message-42","to":[{"phone_number":"+12125550101","status":"queued"}]}}`,
		"clicksend":     `{"response_code":"SUCCESS","data":{"messages":[{"status":"SUCCESS","message_id":"message-42"}]}}`,
		"goalert":       `{}`,
		"signl4":        `{}`,
		"pagertree":     `{}`,
		"ilert":         `{}`,
		"incidentio":    `{"status":"success","deduplication_key":"monitor/42"}`,
		"firehydrant":   `{}`,
		"betterstack":   `{"data":{"id":"42","type":"incident"}}`,
		"fcm":           `{"name":"projects/project/messages/message-42"}`,
		"onesignal":     `{"id":"message-42"}`,
		"expo":          `{"data":{"status":"ok","id":"ticket-42"}}`,
		"whatsapp":      `{"contacts":[{"input":"12125550101","wa_id":"12125550101"}],"messages":[{"id":"wamid.fixture"}]}`,
	}
	var accepted atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.Split(strings.Trim(r.URL.Path, "/"), "/")[0]
		body, ok := acknowledgements[name]
		if !ok {
			w.WriteHeader(404)
			return
		}
		accepted.Add(1)
		if name == "ilert" || name == "incidentio" {
			w.WriteHeader(202)
		} else if name == "betterstack" {
			w.WriteHeader(201)
		}
		_, _ = io.WriteString(w, body)
	}))
	defer server.Close()
	// snippet:start
	connection := notification.HTTPConfig{
		AllowPrivate: true,
	}
	mobile, err := pushover.New(pushover.Config{
		BaseURL: server.URL + "/pushover",
		Token:   "fixture-application-token",
		User:    "fixture-user-key",
		HTTP:    connection,
	})
	if err != nil {
		panic(err)
	}
	defer mobile.Close()
	room, err := mattermost.New(mattermost.Config{
		URL:  server.URL + "/mattermost",
		HTTP: connection,
	})
	if err != nil {
		panic(err)
	}
	defer room.Close()
	sender := notification.Fanout(mobile, room)
	err = sender.Send(context.Background(), notification.Message{
		Title: "Ticket updated",
		Body:  "Ticket T-42 has a new owner.",
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("accepted through push and chat:", accepted.Load())
	// accepted through push and chat: 2
	// snippet:end
	verified := 0
	for _, entry := range configured(server.URL, connection) {
		driver, err := entry.open()
		if err != nil {
			panic(entry.name + ": constructor failed")
		}
		err = driver.Send(context.Background(), notification.Message{
			Title: "Ticket",
			Body:  "T-42 changed",
		})
		if closer, ok := driver.(interface{ Close() error }); ok {
			_ = closer.Close()
		}
		if err != nil {
			panic(entry.name + ": local native protocol failed")
		}
		verified++
	}
	fmt.Println("native provider examples verified:", verified)
	// native provider examples verified: 28
}
