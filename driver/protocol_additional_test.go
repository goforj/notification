package driver_test

import (
	"encoding/json"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// protocolValue inspects nested native fields without sharing production wire types.
func protocolValue(value any, path string) any {
	for _, segment := range strings.Split(path, ".") {
		switch container := value.(type) {
		case map[string]any:
			value = container[segment]
		case []any:
			index, err := strconv.Atoi(segment)
			if err != nil || index < 0 || index >= len(container) {
				return nil
			}
			value = container[index]
		default:
			return nil
		}
	}
	return value
}

// inspectAdditionalProtocol checks native field names, destination identity and authentication.
func inspectAdditionalProtocol(t *testing.T, name string, r *http.Request, body []byte) {
	t.Helper()
	var object map[string]any
	_ = json.Unmarshal(body, &object)
	check := func(path string, want any) {
		t.Helper()
		if got := protocolValue(object, path); !reflect.DeepEqual(got, want) {
			t.Errorf("%s native field %s: got %v, want %v", name, path, got, want)
		}
	}
	header := func(key, want string) {
		t.Helper()
		if r.Header.Get(key) != want {
			t.Errorf("%s native header %s", name, key)
		}
	}
	path := func(want string) {
		t.Helper()
		if r.URL.Path != want {
			t.Errorf("%s native path: %s", name, r.URL.Path)
		}
	}
	form := func(want map[string]string) {
		t.Helper()
		values, err := url.ParseQuery(string(body))
		if err != nil {
			t.Error(err)
		}
		for key, expected := range want {
			if values.Get(key) != expected {
				t.Errorf("%s native form field %s", name, key)
			}
		}
	}
	switch name {
	case "mattermost", "rocketchat":
		check("text", "Ticket\nT-42 changed")
	case "bark":
		path("/push")
		check("device_key", "fixture")
		check("title", "Ticket")
		check("body", "T-42 changed")
		check("group", "alerts")
	case "pushbullet":
		path("/v2/pushes")
		header("Access-Token", "fixture")
		check("type", "note")
		check("device_iden", "device")
		check("title", "Ticket")
		check("body", "T-42 changed")
	case "pushover":
		path("/1/messages.json")
		form(map[string]string{"token": "fixture", "user": "user", "device": "phone", "priority": "0", "title": "Ticket", "message": "T-42 changed"})
	case "matrix":
		header("Authorization", "Bearer fixture")
		check("msgtype", "m.text")
		check("body", "Ticket\nT-42 changed")
		if !strings.HasPrefix(r.URL.Path, "/_matrix/client/v3/rooms/!room:example.com/send/m.room.message/") {
			t.Error("Matrix room or event type")
		}
	case "zulip":
		path("/api/v1/messages")
		user, key, ok := r.BasicAuth()
		if !ok || user != "bot@example.com" || key != "fixture" {
			t.Error("Zulip bot authentication")
		}
		form(map[string]string{"type": "stream", "to": "alerts", "topic": "Tickets", "content": "Ticket\nT-42 changed"})
	case "ifttt":
		path("/trigger/ticket_updated/with/key/fixture")
		check("value1", "Ticket")
		check("value2", "T-42 changed")
	case "join":
		path("/_ah/api/messaging/v1/sendPush")
		if r.URL.Query().Get("deviceId") != "device" || r.URL.Query().Get("title") != "Ticket" {
			t.Error("Join configured destination")
		}
	case "homeassistant":
		path("/api/services/notify/mobile_app_phone")
		header("Authorization", "Bearer fixture")
		check("title", "Ticket")
		check("message", "T-42 changed")
	case "nextcloudtalk":
		path("/ocs/v2.php/apps/spreed/api/v1/chat/room42")
		header("OCS-APIRequest", "true")
		check("message", "Ticket\nT-42 changed")
		check("silent", false)
		if r.URL.Query().Get("format") != "json" {
			t.Error("missing OCS response format")
		}
	case "signal":
		path("/v2/send")
		header("Authorization", "Bearer fixture")
		check("number", "+12125550100")
		check("recipients.0", "+12125550101")
		check("message", "Ticket\nT-42 changed")
		check("text_mode", "normal")
	case "apprise":
		path("/notify")
		header("X-Apprise-Config-ID", "fixture")
		check("title", "Ticket")
		check("body", "T-42 changed")
		check("format", "text")
		check("tag", "alerts")
		check("urls", nil)
	case "plivo":
		path("/v1/Account/fixture/Message/")
		check("src", "+12125550100")
		check("dst", "+12125550101")
		check("text", "Ticket\nT-42 changed")
	case "vonage":
		path("/sms/json")
		form(map[string]string{"from": "Helpdesk", "to": "12125550101", "text": "Ticket\nT-42 changed", "type": "unicode"})
	case "telnyx":
		path("/v2/messages")
		header("Authorization", "Bearer fixture")
		check("from", "+12125550100")
		check("to", "+12125550101")
		check("text", "Ticket\nT-42 changed")
		check("messaging_profile_id", "profile")
	case "clicksend":
		path("/v3/sms/send")
		check("messages.0.to", "+12125550101")
		check("messages.0.from", "Helpdesk")
		check("messages.0.body", "Ticket\nT-42 changed")
		check("messages.0.source", "go")
	case "goalert":
		path("/api/v2/generic/incoming")
		header("Authorization", "Bearer fixture")
		check("dedup", "monitor/42")
		check("summary", "Ticket")
		check("details", "T-42 changed")
	case "signl4":
		check("X-S4-ExternalID", "monitor/42")
		check("X-S4-Status", "new")
		check("title", "Ticket")
		check("body", "T-42 changed")
	case "pagertree":
		check("Id", "monitor/42")
		check("event_type", "create")
		check("Title", "Ticket")
		check("Description", "T-42 changed")
	case "ilert":
		check("integrationKey", "fixture")
		check("eventType", "ALERT")
		check("alertKey", "monitor/42")
		check("summary", "Ticket")
		check("details", "T-42 changed")
	case "incidentio":
		path("/v2/alert_events/http/source")
		header("Authorization", "Bearer fixture")
		check("deduplication_key", "monitor/42")
		check("status", "firing")
		check("title", "Ticket")
		check("description", "T-42 changed")
	case "firehydrant":
		check("idempotency_key", "monitor/42")
		check("status", "OPEN")
		check("level", "INFO")
		check("summary", "Ticket")
		check("body", "T-42 changed")
	case "betterstack":
		path("/api/v3/incidents")
		header("Authorization", "Bearer fixture")
		check("summary", "Ticket")
		check("description", "T-42 changed")
		check("requester_email", "user@example.com")
		check("policy_id", "policy")
		check("team_name", "Helpdesk")
		check("email", true)
	case "fcm":
		path("/v1/projects/project/messages:send")
		header("Authorization", "Bearer fixture")
		check("message.token", "device")
		check("message.notification.title", "Ticket")
		check("message.notification.body", "T-42 changed")
	case "onesignal":
		path("/notifications")
		header("Authorization", "Key fixture")
		check("app_id", "application")
		check("target_channel", "push")
		check("include_subscription_ids.0", "device")
		check("contents.en", "T-42 changed")
		check("headings.en", "Ticket")
	case "expo":
		path("/--/api/v2/push/send")
		header("Authorization", "Bearer fixture")
		check("to", "ExpoPushToken[fixture]")
		check("title", "Ticket")
		check("body", "T-42 changed")
	case "whatsapp":
		path("/v24.0/123456789/messages")
		header("Authorization", "Bearer fixture")
		check("messaging_product", "whatsapp")
		check("to", "12125550101")
		check("type", "text")
		check("text.body", "Ticket\nT-42 changed")
		check("text.preview_url", false)
	}
}
