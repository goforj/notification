package driver_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/goforj/notification"
	"github.com/goforj/notification/driver/dingtalk"
	"github.com/goforj/notification/driver/discord"
	"github.com/goforj/notification/driver/feishu"
	"github.com/goforj/notification/driver/line"
	"github.com/goforj/notification/driver/waha"
	"github.com/goforj/notification/driver/wecom"
	"github.com/goforj/notification/driver/whapi"

	"github.com/goforj/notification/driver/googlechat"
	"github.com/goforj/notification/driver/gotify"
	"github.com/goforj/notification/driver/ntfy"
	"github.com/goforj/notification/driver/slack"
	"github.com/goforj/notification/driver/teams"
	"github.com/goforj/notification/driver/telegram"
	"github.com/goforj/notification/notificationtest"
)

// nativeSender retains explicit transport ownership in the conformance fixture.
type nativeSender interface {
	notification.Sender
	Close() error
}

// TestNativeProviders executes the shared contract and native payload/acknowledgement evidence.
func TestNativeProviders(t *testing.T) {
	cases := []struct {
		name    string
		new     func(string, notification.HTTPConfig) (nativeSender, error)
		ack     string
		inspect func(*testing.T, *http.Request, map[string]json.RawMessage)
		badAcks []string
		limit   int
	}{
		{"dingtalk", func(url string, h notification.HTTPConfig) (nativeSender, error) {
			return dingtalk.New(dingtalk.Config{URL: url, Secret: "secret", HTTP: h})
		}, `{"errcode":0}`, func(t *testing.T, r *http.Request, m map[string]json.RawMessage) {
			timestamp := r.URL.Query().Get("timestamp")
			mac := hmac.New(sha256.New, []byte("secret"))
			_, _ = mac.Write([]byte(timestamp + "\nsecret"))
			if timestamp == "" || r.URL.Query().Get("sign") != base64.StdEncoding.EncodeToString(mac.Sum(nil)) || string(m["msgtype"]) != `"text"` {
				t.Error("DingTalk signature/payload")
			}
		}, []string{`{}`, `{"errcode":null}`, `{"errcode":"0"}`, `{"errcode":310000}`}, 0},
		{"feishu", func(url string, h notification.HTTPConfig) (nativeSender, error) {
			return feishu.New(feishu.Config{URL: url, Secret: "secret", HTTP: h})
		}, `{"code":0}`, func(t *testing.T, r *http.Request, m map[string]json.RawMessage) {
			var timestamp, sign string
			_ = json.Unmarshal(m["timestamp"], &timestamp)
			_ = json.Unmarshal(m["sign"], &sign)
			mac := hmac.New(sha256.New, []byte(timestamp+"\nsecret"))
			if timestamp == "" || sign != base64.StdEncoding.EncodeToString(mac.Sum(nil)) || string(m["msg_type"]) != `"text"` {
				t.Error("Feishu signature/payload")
			}
		}, []string{`{}`, `{"code":null}`, `{"code":"0"}`, `{"code":19024}`}, 0},
		{"wecom", func(url string, h notification.HTTPConfig) (nativeSender, error) {
			return wecom.New(wecom.Config{URL: url, HTTP: h})
		}, `{"errcode":0}`, func(t *testing.T, r *http.Request, m map[string]json.RawMessage) {
			if string(m["msgtype"]) != `"text"` || !strings.Contains(string(m["text"]), "T-42 changed") {
				t.Error("WeCom payload")
			}
		}, []string{`{}`, `{"errcode":40014}`}, 0},
		{"line", func(url string, h notification.HTTPConfig) (nativeSender, error) {
			return line.New(line.Config{BaseURL: url, Token: "secret", To: "U" + strings.Repeat("a", 32), Silent: true, HTTP: h})
		}, `{"sentMessages":[{"id":"42"}]}`, func(t *testing.T, r *http.Request, m map[string]json.RawMessage) {
			if r.URL.Path != "/v2/bot/message/push" || r.Header.Get("Authorization") != "Bearer secret" || string(m["notificationDisabled"]) != "true" || !strings.Contains(string(m["messages"]), `"type":"text"`) {
				t.Error("LINE payload")
			}
		}, []string{`{}`, `{"sentMessages":[]}`, `{"sentMessages":[{}]}`}, 0},
		{"waha", func(url string, h notification.HTTPConfig) (nativeSender, error) {
			return waha.New(waha.Config{URL: url, APIKey: "secret", Session: "fixture", ChatID: "123@c.us", HTTP: h})
		}, `{}`, func(t *testing.T, r *http.Request, m map[string]json.RawMessage) {
			if r.URL.Path != "/api/sendText" || r.Header.Get("X-Api-Key") != "secret" || string(m["session"]) != `"fixture"` || string(m["chatId"]) != `"123@c.us"` || string(m["linkPreview"]) != "false" {
				t.Error("WAHA payload")
			}
		}, nil, 0},
		{"whapi", func(url string, h notification.HTTPConfig) (nativeSender, error) {
			return whapi.New(whapi.Config{BaseURL: url, Token: "secret", To: "123456789", HTTP: h})
		}, `{"sent":true}`, func(t *testing.T, r *http.Request, m map[string]json.RawMessage) {
			if r.URL.Path != "/messages/text" || r.Header.Get("Authorization") != "Bearer secret" || string(m["to"]) != `"123456789"` || string(m["no_link_preview"]) != "true" {
				t.Error("Whapi payload")
			}
		}, []string{`{}`, `{"sent":false}`, `{"sent":"true"}`}, 0},
		{"slack", func(url string, h notification.HTTPConfig) (nativeSender, error) {
			return slack.New(slack.Config{URL: url, HTTP: h})
		}, "ok", func(t *testing.T, r *http.Request, m map[string]json.RawMessage) {
			if string(m["mrkdwn"]) != "false" || !strings.Contains(string(m["text"]), "T-42 changed") {
				t.Error("Slack payload")
			}
		}, []string{"invalid_payload"}, 0},
		{"discord", func(url string, h notification.HTTPConfig) (nativeSender, error) {
			return discord.New(discord.Config{URL: url, ThreadID: "42", HTTP: h})
		}, `{"id":"123","channel_id":"456"}`, func(t *testing.T, r *http.Request, m map[string]json.RawMessage) {
			if r.URL.Query().Get("wait") != "true" || r.URL.Query().Get("thread_id") != "42" || string(m["allowed_mentions"]) != `{"parse":[]}` || !strings.Contains(string(m["content"]), "T-42 changed") {
				t.Error("Discord payload")
			}
		}, []string{`{}`, `{"id":"123"}`, `{"id":1}`}, 2000},
		{"googlechat", func(url string, h notification.HTTPConfig) (nativeSender, error) {
			return googlechat.New(googlechat.Config{URL: url, HTTP: h})
		}, `{"name":"spaces/1/messages/2"}`, func(t *testing.T, r *http.Request, m map[string]json.RawMessage) {
			if !strings.Contains(string(m["text"]), "T-42 changed") {
				t.Error("Chat payload")
			}
		}, []string{`{}`, `{"name":"private"}`, `{"name":"spaces/1"}`}, 0},
		{"teams", func(url string, h notification.HTTPConfig) (nativeSender, error) {
			return teams.New(teams.Config{URL: url, HTTP: h})
		}, "", func(t *testing.T, r *http.Request, m map[string]json.RawMessage) {
			var wire struct {
				Type        string `json:"type"`
				Attachments []struct {
					ContentType string `json:"contentType"`
					Content     struct {
						Type, Version string
						Body          []struct {
							Type, Text string
							Wrap       bool
						}
					}
				} `json:"attachments"`
			}
			raw, _ := json.Marshal(m)
			if json.Unmarshal(raw, &wire) != nil || wire.Type != "message" || len(wire.Attachments) != 1 || wire.Attachments[0].ContentType != "application/vnd.microsoft.card.adaptive" || wire.Attachments[0].Content.Type != "AdaptiveCard" || wire.Attachments[0].Content.Version != "1.2" || len(wire.Attachments[0].Content.Body) != 2 || !wire.Attachments[0].Content.Body[1].Wrap {
				t.Error("Teams Adaptive Card payload")
			}
		}, nil, 0},
		{"telegram", func(url string, h notification.HTTPConfig) (nativeSender, error) {
			return telegram.New(telegram.Config{BaseURL: url, Token: "123:abc", ChatID: "-42", ThreadID: 2, Silent: true, ProtectContent: true, HTTP: h})
		}, `{"ok":true,"result":{"message_id":42}}`, func(t *testing.T, r *http.Request, m map[string]json.RawMessage) {
			if !strings.HasSuffix(r.URL.Path, "/bot123:abc/sendMessage") || string(m["chat_id"]) != `"-42"` || string(m["message_thread_id"]) != "2" || string(m["disable_notification"]) != "true" || string(m["protect_content"]) != "true" || m["parse_mode"] != nil || string(m["link_preview_options"]) != `{"is_disabled":true}` {
				t.Error("Telegram payload")
			}
		}, []string{`{}`, `{"ok":true}`, `{"ok":true,"result":{}}`, `{"ok":true,"result":{"message_id":0}}`}, 4096},
		{"ntfy", func(url string, h notification.HTTPConfig) (nativeSender, error) {
			return ntfy.New(ntfy.Config{URL: url, Topic: "fixture", Token: "secret", Priority: 4, HTTP: h})
		}, `{"id":"abc","event":"message","topic":"fixture"}`, func(t *testing.T, r *http.Request, m map[string]json.RawMessage) {
			if r.URL.Path != "/" || r.Header.Get("Authorization") != "Bearer secret" || string(m["topic"]) != `"fixture"` || string(m["priority"]) != "4" {
				t.Error("ntfy native payload")
			}
		}, []string{`{}`, `{"id":"x","event":"open","topic":"fixture"}`, `{"id":"x","event":"message","topic":"other"}`}, 0},
		{"gotify", func(url string, h notification.HTTPConfig) (nativeSender, error) {
			return gotify.New(gotify.Config{URL: url, Token: "secret", Priority: 8, HTTP: h})
		}, `{"id":42,"title":"Ticket","message":"T-42 changed"}`, func(t *testing.T, r *http.Request, m map[string]json.RawMessage) {
			if r.URL.Path != "/message" || r.Header.Get("X-Gotify-Key") != "secret" || r.URL.Query().Get("token") != "" || string(m["priority"]) != "8" {
				t.Error("Gotify native payload")
			}
		}, []string{`{}`, `{"id":0}`, `{"id":1,"title":"wrong","message":"T-42 changed"}`, `{"id":1,"title":"Ticket","message":"wrong"}`}, 0},
	}
	for _, entry := range cases {
		t.Run(entry.name, func(t *testing.T) {
			var mu sync.Mutex
			status := 200
			body := entry.ack
			inspect := true
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				calls++
				if r.Method != "POST" || r.Header.Get("Content-Type") != "application/json" {
					t.Error("native request envelope")
				}
				var payload map[string]json.RawMessage
				if json.NewDecoder(r.Body).Decode(&payload) != nil {
					t.Error("invalid native JSON")
				}
				if inspect {
					entry.inspect(t, r, payload)
				}
				w.Header().Set("Retry-After", "7")
				w.WriteHeader(status)
				_, _ = io.WriteString(w, body)
			}))
			defer server.Close()
			driver, err := entry.new(server.URL, notification.HTTPConfig{AllowPrivate: true})
			if err != nil {
				t.Fatal(err)
			}
			defer driver.Close()
			notificationtest.Sender(t, func() notification.Sender { return driver })
			if _, err := entry.new("https://user:secret@example.com", notification.HTTPConfig{}); !errors.Is(err, notification.ErrInvalidRequest) {
				t.Fatal("invalid URL accepted")
			}
			if _, err := entry.new(server.URL, notification.HTTPConfig{Timeout: -1}); !errors.Is(err, notification.ErrInvalidRequest) {
				t.Fatal("invalid transport accepted")
			}
			message := notification.Message{Title: "Ticket", Body: "T-42 changed"}
			for _, bad := range append(entry.badAcks, "invalid json", strings.Repeat("x", 65537)) {
				if entry.badAcks == nil {
					break
				}
				mu.Lock()
				body = bad
				mu.Unlock()
				err := driver.Send(context.Background(), message)
				var failure *notification.DeliveryError
				if (!errors.Is(err, notification.ErrOutcomeUnknown) && !errors.Is(err, notification.ErrPermanent)) || !errors.As(err, &failure) || failure.Driver() != entry.name {
					t.Fatal("malformed success was accepted", err)
				}
				expected := notification.Unknown
				if errors.Is(err, notification.ErrPermanent) {
					expected = notification.Failed
				}
				if failure.Status() != expected {
					t.Fatal("native rejection versus uncertainty was lost")
				}
			}
			mu.Lock()
			status = 429
			body = "private provider response"
			mu.Unlock()
			err = notification.New(driver).Send(nil, message)
			var failure *notification.DeliveryError
			if !errors.Is(err, notification.ErrThrottled) || !errors.As(err, &failure) || failure.RetryAfter() != 7*time.Second || strings.Contains(err.Error(), "private") {
				t.Fatal("retry evidence lost", err)
			}
			if entry.limit > 0 {
				mu.Lock()
				before := calls
				mu.Unlock()
				if err := driver.Send(context.Background(), notification.Message{Body: strings.Repeat("x", entry.limit+1)}); !errors.Is(err, notification.ErrRender) {
					t.Fatal("oversized native content sent")
				}
				mu.Lock()
				after := calls
				mu.Unlock()
				if before != after {
					t.Fatal("unsupported content reached provider")
				}
				mu.Lock()
				inspect = false
				status = 200
				body = entry.ack
				mu.Unlock()
				if err := driver.Send(context.Background(), notification.Message{Body: strings.Repeat("界", entry.limit)}); err != nil {
					t.Fatal("native Unicode boundary rejected", err)
				}

			}
			if entry.name == "wecom" || entry.name == "feishu" || entry.name == "line" {
				mu.Lock()
				inspect = false
				mu.Unlock()
				content := strings.Repeat("界", 683)
				if entry.name == "feishu" {
					content = strings.Repeat("<", 4000)
				}
				if entry.name == "line" {
					content = strings.Repeat("😀", 2501)
				}
				if err := driver.Send(nil, notification.Message{Body: content}); !errors.Is(err, notification.ErrRender) {
					t.Fatal("native render bound", err)
				}
			}
			if entry.name == "teams" {
				mu.Lock()
				inspect = false
				status = 202
				body = ""
				mu.Unlock()
				if err := driver.Send(context.Background(), notification.Message{Body: "no title"}); err != nil {
					t.Fatal(err)
				}
			}
			if entry.name == "telegram" {
				for _, code := range []int{401, 403, 429, 400} {
					mu.Lock()
					status = 200
					body = fmt.Sprintf(`{"ok":false,"error_code":%d,"parameters":{"retry_after":9}}`, code)
					mu.Unlock()
					err := notification.New(driver).Send(nil, message)
					if err == nil {
						t.Fatal("Telegram native rejection accepted")
					}
					if code == 429 {
						var detail *notification.DeliveryError
						if !errors.As(err, &detail) || detail.RetryAfter() != 9*time.Second {
							t.Fatal("Telegram native retry advice lost")
						}
					}
				}
			}
		})
	}
}

// TestProviderConstruction rejects invalid native fields before any network operation.
func TestProviderConstruction(t *testing.T) {
	for _, config := range []ntfy.Config{
		{}, {Topic: "bad topic"}, {Topic: "fixture", Priority: -1}, {Topic: "fixture", Priority: 6},
		{Topic: "fixture", Token: "bad\n"}, {Topic: "fixture", Token: "token", Username: "u"},
		{Topic: "fixture", Username: "u"}, {Topic: "fixture", Password: "p"},
		{Topic: "fixture", Username: "bad\n", Password: "p"},
	} {
		if _, err := ntfy.New(config); err == nil {
			t.Fatal("invalid ntfy config")
		}
	}
	for _, config := range []gotify.Config{
		{}, {URL: "https://example.com"}, {URL: "https://example.com", Token: "bad\n"},
		{URL: "https://example.com", Token: "token", Priority: -1}, {URL: "https://example.com", Token: "token", Priority: 2147483648},
	} {
		if _, err := gotify.New(config); err == nil {
			t.Fatal("invalid Gotify config")
		}
	}
	for _, thread := range []string{"0", "bad", "18446744073709551616"} {
		if _, err := discord.New(discord.Config{URL: "https://example.com", ThreadID: thread}); err == nil {
			t.Fatal("invalid Discord thread")
		}
	}
	for _, config := range []telegram.Config{
		{}, {Token: "bad", ChatID: "42"}, {Token: "1:t", ChatID: ""}, {Token: "1:t", ChatID: "bad\n"},
		{Token: "1:t", ChatID: strings.Repeat("x", 129)}, {Token: "1:t", ChatID: "42", ThreadID: -1},
		{BaseURL: "https://example.com?bad=query", Token: "1:t", ChatID: "42"},
	} {
		if _, err := telegram.New(config); err == nil {
			t.Fatal("invalid Telegram config")
		}
	}
	driver, err := discord.New(discord.Config{URL: "https://example.com"})
	if err != nil {
		t.Fatal(err)
	}
	_ = driver.Close()
	plain, err := ntfy.New(ntfy.Config{Topic: "fixture"})
	if err != nil {
		t.Fatal(err)
	}
	_ = plain.Close()
	basic, err := ntfy.New(ntfy.Config{URL: "https://example.com", Topic: "fixture", Username: "u", Password: "p"})
	if err != nil {
		t.Fatal(err)
	}
	_ = basic.Close()
}
