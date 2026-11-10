// Command providers demonstrates native provider configurations against local fixtures.
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
	"github.com/goforj/notification/driver/dingtalk"
	"github.com/goforj/notification/driver/discord"
	"github.com/goforj/notification/driver/expo"
	"github.com/goforj/notification/driver/fcm"
	"github.com/goforj/notification/driver/feishu"
	"github.com/goforj/notification/driver/firehydrant"
	"github.com/goforj/notification/driver/goalert"
	"github.com/goforj/notification/driver/googlechat"
	"github.com/goforj/notification/driver/gotify"
	"github.com/goforj/notification/driver/homeassistant"
	"github.com/goforj/notification/driver/ifttt"
	"github.com/goforj/notification/driver/ilert"
	"github.com/goforj/notification/driver/incidentio"
	"github.com/goforj/notification/driver/join"
	"github.com/goforj/notification/driver/jsm"
	"github.com/goforj/notification/driver/line"
	"github.com/goforj/notification/driver/matrix"
	"github.com/goforj/notification/driver/mattermost"
	"github.com/goforj/notification/driver/nextcloudtalk"
	"github.com/goforj/notification/driver/ntfy"
	"github.com/goforj/notification/driver/onesignal"
	"github.com/goforj/notification/driver/opsgenie"
	"github.com/goforj/notification/driver/pagertree"
	"github.com/goforj/notification/driver/plivo"
	"github.com/goforj/notification/driver/pushbullet"
	"github.com/goforj/notification/driver/pushover"
	"github.com/goforj/notification/driver/rocketchat"
	"github.com/goforj/notification/driver/signal"
	"github.com/goforj/notification/driver/signl4"
	"github.com/goforj/notification/driver/teams"
	"github.com/goforj/notification/driver/telegram"
	"github.com/goforj/notification/driver/telnyx"
	"github.com/goforj/notification/driver/twilio"
	"github.com/goforj/notification/driver/vonage"
	"github.com/goforj/notification/driver/waha"
	"github.com/goforj/notification/driver/wecom"
	"github.com/goforj/notification/driver/whapi"
	"github.com/goforj/notification/driver/whatsapp"
	"github.com/goforj/notification/driver/zulip"
)

// configuredProvider binds a readable typed configuration to one local native protocol fixture.
type configuredProvider struct {
	name string
	open func() (notification.Sender, error)
}

// configured lists complete constructor examples while preserving provider-specific configuration.
func configured(baseURL string, connection notification.HTTPConfig) []configuredProvider {
	return []configuredProvider{
		{
			name: "dingtalk",
			open: func() (notification.Sender, error) {
				return dingtalk.New(dingtalk.Config{
					URL:  baseURL + "/dingtalk",
					HTTP: connection,
				})
			},
		},
		{
			name: "discord",
			open: func() (notification.Sender, error) {
				return discord.New(discord.Config{
					URL:  baseURL + "/discord",
					HTTP: connection,
				})
			},
		},
		{
			name: "feishu",
			open: func() (notification.Sender, error) {
				return feishu.New(feishu.Config{
					URL:  baseURL + "/feishu",
					HTTP: connection,
				})
			},
		},
		{
			name: "googlechat",
			open: func() (notification.Sender, error) {
				return googlechat.New(googlechat.Config{
					URL:  baseURL + "/googlechat",
					HTTP: connection,
				})
			},
		},
		{
			name: "gotify",
			open: func() (notification.Sender, error) {
				return gotify.New(gotify.Config{
					URL:      baseURL + "/gotify",
					Token:    "fixture",
					Priority: 5,
					HTTP:     connection,
				})
			},
		},
		{
			name: "jsm",
			open: func() (notification.Sender, error) {
				return jsm.New(jsm.Config{
					BaseURL: baseURL + "/jsm",
					CloudID: "cloud",
					Email:   "bot@example.com",
					Token:   "fixture",
					Key:     "monitor/42",
					HTTP:    connection,
				})
			},
		},
		{
			name: "line",
			open: func() (notification.Sender, error) {
				return line.New(line.Config{
					BaseURL: baseURL + "/line",
					Token:   "fixture",
					To:      "U00000000000000000000000000000000",
					HTTP:    connection,
				})
			},
		},
		{
			name: "ntfy",
			open: func() (notification.Sender, error) {
				return ntfy.New(ntfy.Config{
					URL:   baseURL + "/ntfy",
					Topic: "alerts",
					HTTP:  connection,
				})
			},
		},
		{
			name: "opsgenie",
			open: func() (notification.Sender, error) {
				return opsgenie.New(opsgenie.Config{
					BaseURL: baseURL + "/opsgenie",
					Token:   "fixture",
					Key:     "monitor/42",
					HTTP:    connection,
				})
			},
		},
		{
			name: "teams",
			open: func() (notification.Sender, error) {
				return teams.New(teams.Config{
					URL:  baseURL + "/teams",
					HTTP: connection,
				})
			},
		},
		{
			name: "telegram",
			open: func() (notification.Sender, error) {
				return telegram.New(telegram.Config{
					BaseURL: baseURL + "/telegram",
					Token:   "123456:fixture",
					ChatID:  "42",
					HTTP:    connection,
				})
			},
		},
		{
			name: "twilio",
			open: func() (notification.Sender, error) {
				return twilio.New(twilio.Config{
					BaseURL:    baseURL + "/twilio",
					AccountSID: "AC00000000000000000000000000000000",
					AuthToken:  "fixture",
					From:       "+12125550100",
					To:         "+12125550101",
					HTTP:       connection,
				})
			},
		},
		{
			name: "waha",
			open: func() (notification.Sender, error) {
				return waha.New(waha.Config{
					URL:     baseURL + "/waha",
					APIKey:  "fixture",
					Session: "default",
					ChatID:  "12125550101@c.us",
					HTTP:    connection,
				})
			},
		},
		{
			name: "wecom",
			open: func() (notification.Sender, error) {
				return wecom.New(wecom.Config{
					URL:  baseURL + "/wecom",
					HTTP: connection,
				})
			},
		},
		{
			name: "whapi",
			open: func() (notification.Sender, error) {
				return whapi.New(whapi.Config{
					BaseURL: baseURL + "/whapi",
					Token:   "fixture",
					To:      "12125550101",
					HTTP:    connection,
				})
			},
		},
		{
			name: "mattermost",
			open: func() (notification.Sender, error) {
				return mattermost.New(mattermost.Config{
					URL:  baseURL + "/mattermost",
					HTTP: connection,
				})
			},
		},
		{
			name: "rocketchat",
			open: func() (notification.Sender, error) {
				return rocketchat.New(rocketchat.Config{
					URL:  baseURL + "/rocketchat",
					HTTP: connection,
				})
			},
		},
		{
			name: "bark",
			open: func() (notification.Sender, error) {
				return bark.New(bark.Config{
					URL:       baseURL + "/bark",
					DeviceKey: "fixture",
					Group:     "alerts",
					HTTP:      connection,
				})
			},
		},
		{
			name: "pushbullet",
			open: func() (notification.Sender, error) {
				return pushbullet.New(pushbullet.Config{
					BaseURL:  baseURL + "/pushbullet",
					Token:    "fixture",
					DeviceID: "device",
					HTTP:     connection,
				})
			},
		},
		{
			name: "pushover",
			open: func() (notification.Sender, error) {
				return pushover.New(pushover.Config{
					BaseURL: baseURL + "/pushover",
					Token:   "fixture",
					User:    "user",
					Device:  "phone",
					HTTP:    connection,
				})
			},
		},
		{
			name: "matrix",
			open: func() (notification.Sender, error) {
				return matrix.New(matrix.Config{
					URL:    baseURL + "/matrix",
					Token:  "fixture",
					RoomID: "!room:example.com",
					HTTP:   connection,
				})
			},
		},
		{
			name: "zulip",
			open: func() (notification.Sender, error) {
				return zulip.New(zulip.Config{
					URL:     baseURL + "/zulip",
					Email:   "bot@example.com",
					APIKey:  "fixture",
					Channel: "alerts",
					Topic:   "Tickets",
					HTTP:    connection,
				})
			},
		},
		{
			name: "ifttt",
			open: func() (notification.Sender, error) {
				return ifttt.New(ifttt.Config{
					BaseURL: baseURL + "/ifttt",
					Key:     "fixture",
					Event:   "ticket_updated",
					HTTP:    connection,
				})
			},
		},
		{
			name: "join",
			open: func() (notification.Sender, error) {
				return join.New(join.Config{
					BaseURL:  baseURL + "/join",
					APIKey:   "fixture",
					DeviceID: "device",
					HTTP:     connection,
				})
			},
		},
		{
			name: "homeassistant",
			open: func() (notification.Sender, error) {
				return homeassistant.New(homeassistant.Config{
					URL:     baseURL + "/homeassistant",
					Token:   "fixture",
					Service: "mobile_app_phone",
					HTTP:    connection,
				})
			},
		},
		{
			name: "nextcloudtalk",
			open: func() (notification.Sender, error) {
				return nextcloudtalk.New(nextcloudtalk.Config{
					URL:       baseURL + "/nextcloudtalk",
					Username:  "fixture",
					Password:  "secret",
					RoomToken: "room42",
					HTTP:      connection,
				})
			},
		},
		{
			name: "signal",
			open: func() (notification.Sender, error) {
				return signal.New(signal.Config{
					URL:       baseURL + "/signal",
					Number:    "+12125550100",
					Recipient: "+12125550101",
					Token:     "fixture",
					HTTP:      connection,
				})
			},
		},
		{
			name: "apprise",
			open: func() (notification.Sender, error) {
				return apprise.New(apprise.Config{
					URL:      baseURL + "/apprise",
					Key:      "fixture",
					Tag:      "alerts",
					Username: "fixture",
					Password: "secret",
					HTTP:     connection,
				})
			},
		},
		{
			name: "plivo",
			open: func() (notification.Sender, error) {
				return plivo.New(plivo.Config{
					BaseURL:   baseURL + "/plivo",
					AuthID:    "fixture",
					AuthToken: "secret",
					From:      "+12125550100",
					To:        "+12125550101",
					HTTP:      connection,
				})
			},
		},
		{
			name: "vonage",
			open: func() (notification.Sender, error) {
				return vonage.New(vonage.Config{
					BaseURL:   baseURL + "/vonage",
					APIKey:    "fixture1",
					APISecret: "secret",
					From:      "Helpdesk",
					To:        "+12125550101",
					HTTP:      connection,
				})
			},
		},
		{
			name: "telnyx",
			open: func() (notification.Sender, error) {
				return telnyx.New(telnyx.Config{
					BaseURL:            baseURL + "/telnyx",
					Token:              "fixture",
					From:               "+12125550100",
					To:                 "+12125550101",
					MessagingProfileID: "profile",
					HTTP:               connection,
				})
			},
		},
		{
			name: "clicksend",
			open: func() (notification.Sender, error) {
				return clicksend.New(clicksend.Config{
					BaseURL:  baseURL + "/clicksend",
					Username: "fixture",
					APIKey:   "secret",
					From:     "Helpdesk",
					To:       "+12125550101",
					HTTP:     connection,
				})
			},
		},
		{
			name: "goalert",
			open: func() (notification.Sender, error) {
				return goalert.New(goalert.Config{
					URL:   baseURL + "/goalert",
					Token: "fixture",
					Key:   "monitor/42",
					HTTP:  connection,
				})
			},
		},
		{
			name: "signl4",
			open: func() (notification.Sender, error) {
				return signl4.New(signl4.Config{
					URL:  baseURL + "/signl4",
					Key:  "monitor/42",
					HTTP: connection,
				})
			},
		},
		{
			name: "pagertree",
			open: func() (notification.Sender, error) {
				return pagertree.New(pagertree.Config{
					URL:  baseURL + "/pagertree",
					Key:  "monitor/42",
					HTTP: connection,
				})
			},
		},
		{
			name: "ilert",
			open: func() (notification.Sender, error) {
				return ilert.New(ilert.Config{
					URL:            baseURL + "/ilert",
					IntegrationKey: "fixture",
					Key:            "monitor/42",
					HTTP:           connection,
				})
			},
		},
		{
			name: "incidentio",
			open: func() (notification.Sender, error) {
				return incidentio.New(incidentio.Config{
					BaseURL:  baseURL + "/incidentio",
					Token:    "fixture",
					SourceID: "source",
					Key:      "monitor/42",
					HTTP:     connection,
				})
			},
		},
		{
			name: "firehydrant",
			open: func() (notification.Sender, error) {
				return firehydrant.New(firehydrant.Config{
					URL:  baseURL + "/firehydrant",
					Key:  "monitor/42",
					HTTP: connection,
				})
			},
		},
		{
			name: "betterstack",
			open: func() (notification.Sender, error) {
				return betterstack.New(betterstack.Config{
					BaseURL:        baseURL + "/betterstack",
					Token:          "fixture",
					RequesterEmail: "user@example.com",
					PolicyID:       "policy",
					TeamName:       "Helpdesk",
					Email:          true,
					HTTP:           connection,
				})
			},
		},
		{
			name: "fcm",
			open: func() (notification.Sender, error) {
				return fcm.New(fcm.Config{
					BaseURL:     baseURL + "/fcm",
					ProjectID:   "project",
					DeviceToken: "device",
					TokenSource: func(context.Context) (string, error) { return "fixture", nil },
					HTTP:        connection,
				})
			},
		},
		{
			name: "onesignal",
			open: func() (notification.Sender, error) {
				return onesignal.New(onesignal.Config{
					BaseURL:        baseURL + "/onesignal",
					AppID:          "application",
					Token:          "fixture",
					SubscriptionID: "device",
					HTTP:           connection,
				})
			},
		},
		{
			name: "expo",
			open: func() (notification.Sender, error) {
				return expo.New(expo.Config{
					BaseURL:     baseURL + "/expo",
					PushToken:   "ExpoPushToken[fixture]",
					AccessToken: "fixture",
					HTTP:        connection,
				})
			},
		},
		{
			name: "whatsapp",
			open: func() (notification.Sender, error) {
				return whatsapp.New(whatsapp.Config{
					BaseURL:       baseURL + "/whatsapp",
					Token:         "fixture",
					PhoneNumberID: "123456789",
					To:            "+12125550101",
					APIVersion:    "v24.0",
					HTTP:          connection,
				})
			},
		},
	}
}

// main verifies native acknowledgements without contacting a hosted provider.
func main() {
	acknowledgements := map[string]string{
		"dingtalk":   `{"errcode":0}`,
		"discord":    `{"id":"message42","channel_id":"channel42"}`,
		"feishu":     `{"code":0}`,
		"googlechat": `{"name":"spaces/room/messages/message42"}`,
		"gotify":     `{"id":42,"title":"Ticket","message":"T-42 changed"}`,
		"jsm":        `{"requestId":"request42"}`,
		"line":       `{"sentMessages":[{"id":"message42"}]}`,
		"ntfy":       `{"id":"message42","event":"message","topic":"alerts"}`,
		"opsgenie":   `{"requestId":"request42"}`,
		"teams":      `{}`,
		"telegram":   `{"ok":true,"result":{"message_id":42}}`,
		"twilio":     `{"sid":"SM00000000000000000000000000000000","status":"queued"}`,
		"waha":       `{}`,
		"wecom":      `{"errcode":0}`,
		"whapi":      `{"sent":true}`,

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
		"vonage":        `{"message-count":"1","messages":[{"status":"0","message-id":"message-42","to":"12125550101"}]}`,
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
		if name == "ilert" || name == "incidentio" || name == "jsm" || name == "opsgenie" {
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
	// native provider examples verified: 43
}
