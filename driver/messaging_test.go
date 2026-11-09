package driver_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/goforj/notification"
	"github.com/goforj/notification/driver/dingtalk"
	"github.com/goforj/notification/driver/feishu"
	"github.com/goforj/notification/driver/line"
	"github.com/goforj/notification/driver/twilio"
	"github.com/goforj/notification/driver/waha"
	"github.com/goforj/notification/driver/whapi"
	"github.com/goforj/notification/notificationtest"
)

// TestMessagingConstruction rejects invalid configuration before creating a transport.
func TestMessagingConstruction(t *testing.T) {
	for _, config := range []dingtalk.Config{{}, {URL: "https://example.com", Secret: "bad\n"}, {URL: "https://example.com", HTTP: notification.HTTPConfig{Timeout: -1}}} {
		if _, err := dingtalk.New(config); err == nil {
			t.Fatal("invalid DingTalk config")
		}
	}
	for _, config := range []feishu.Config{{URL: "https://example.com", Secret: "bad\n"}} {
		if _, err := feishu.New(config); err == nil {
			t.Fatal("invalid Feishu secret")
		}
	}
	if _, err := line.New(line.Config{}); err == nil {
		t.Fatal("invalid LINE configuration")
	}
	if _, err := waha.New(waha.Config{}); err == nil {
		t.Fatal("invalid WAHA configuration")
	}
	if _, err := whapi.New(whapi.Config{}); err == nil {
		t.Fatal("invalid Whapi configuration")
	}
	for _, open := range []func() (nativeSender, error){
		func() (nativeSender, error) { return dingtalk.New(dingtalk.Config{URL: "https://example.com"}) },
		func() (nativeSender, error) { return feishu.New(feishu.Config{URL: "https://example.com"}) },
		func() (nativeSender, error) {
			return line.New(line.Config{Token: "token", To: "U" + strings.Repeat("a", 32)})
		},
		func() (nativeSender, error) { return whapi.New(whapi.Config{Token: "token", To: "123456789"}) },
	} {
		driver, err := open()
		if err != nil {
			t.Fatal(err)
		}
		_ = driver.Close()
	}
}

// TestTwilio verifies form encoding, authentication and native message-state acknowledgements.
func TestTwilio(t *testing.T) {
	account := "AC" + strings.Repeat("a", 32)
	service := "MG" + strings.Repeat("b", 32)
	ack := `{"sid":"SM` + strings.Repeat("c", 32) + `","status":"queued"}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ParseForm() != nil {
			t.Error("invalid form")
		}
		user, pass, ok := r.BasicAuth()
		if !ok || user != account || pass != "token" || r.Method != "POST" || r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" || r.URL.Path != "/2010-04-01/Accounts/"+account+"/Messages.json" || r.Form.Get("To") != "+12125550101" || r.Form.Get("Body") == "" {
			t.Error("Twilio wire contract")
		}
		if (r.Form.Get("From") == "") == (r.Form.Get("MessagingServiceSid") == "") {
			t.Error("sender ambiguity")
		}
		_, _ = io.WriteString(w, ack)
	}))
	defer server.Close()
	config := twilio.Config{BaseURL: server.URL, AccountSID: account, AuthToken: "token", From: "+12125550100", To: "+12125550101", HTTP: notification.HTTPConfig{AllowPrivate: true}}
	driver, err := twilio.New(config)
	if err != nil {
		t.Fatal(err)
	}
	defer driver.Close()
	notificationtest.Sender(t, func() notification.Sender { return driver })
	for _, status := range []string{"accepted", "sending", "sent", "delivered", "failed", "undelivered", "canceled", "unknown"} {
		ack = `{"sid":"SM` + strings.Repeat("c", 32) + `","status":"` + status + `"}`
		err := driver.Send(nil, notification.Message{Body: "body"})
		if (status == "failed" || status == "undelivered" || status == "canceled" || status == "unknown") != (err != nil) {
			t.Fatal("native status", status, err)
		}
	}
	for _, bad := range []string{"invalid", `{}`, `{"sid":"private","status":"queued"}`} {
		ack = bad
		if err := driver.Send(nil, notification.Message{Body: "body"}); !errors.Is(err, notification.ErrOutcomeUnknown) {
			t.Fatal(err)
		}
	}
	if err := driver.Send(nil, notification.Message{Body: strings.Repeat("界", 1601)}); !errors.Is(err, notification.ErrRender) {
		t.Fatal(err)
	}
	ack = `{"sid":"SM` + strings.Repeat("c", 32) + `","status":"queued"}`
	config.From = ""
	config.MessagingServiceSID = service
	managed, err := twilio.New(config)
	if err != nil {
		t.Fatal(err)
	}
	defer managed.Close()
	if err := managed.Send(context.Background(), notification.Message{Body: strings.Repeat("界", 1600)}); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*twilio.Config){
		func(c *twilio.Config) { c.AccountSID = "bad" }, func(c *twilio.Config) { c.AuthToken = "" }, func(c *twilio.Config) { c.To = "bad" }, func(c *twilio.Config) { c.From = "+12125550100" }, func(c *twilio.Config) { c.MessagingServiceSID = "" }, func(c *twilio.Config) { c.MessagingServiceSID = "bad" }, func(c *twilio.Config) { c.BaseURL = "bad" }, func(c *twilio.Config) { c.HTTP.Timeout = -1 },
	} {
		invalid := config
		change(&invalid)
		if _, err := twilio.New(invalid); err == nil {
			t.Fatal("invalid Twilio config")
		}
	}
	config.BaseURL = ""
	plain, err := twilio.New(config)
	if err != nil {
		t.Fatal(err)
	}
	_ = plain.Close()
}

// TestUnsignedDingTalk preserves unsigned robot configuration without adding signature parameters.
func TestUnsignedDingTalk(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Error("unexpected signature")
		}
		_, _ = io.WriteString(w, `{"errcode":0}`)
	}))
	defer server.Close()
	driver, err := dingtalk.New(dingtalk.Config{URL: server.URL, HTTP: notification.HTTPConfig{AllowPrivate: true}})
	if err != nil {
		t.Fatal(err)
	}
	defer driver.Close()
	if err := driver.Send(nil, notification.Message{Body: "body"}); err != nil {
		t.Fatal(err)
	}
}
