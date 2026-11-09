package driver_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/goforj/notification"
	"github.com/goforj/notification/driver/jsm"
	"github.com/goforj/notification/driver/opsgenie"
	"github.com/goforj/notification/driver/pagerduty"
	"github.com/goforj/notification/notificationtest"
)

// incidentDriver exposes both portable capabilities and resource ownership.
type incidentDriver interface {
	nativeSender
	notification.IncidentSender
}

// TestIncidentProviders exercises stable correlation through every native lifecycle action.
func TestIncidentProviders(t *testing.T) {
	for _, name := range []string{"pagerduty", "jsm", "opsgenie"} {
		t.Run(name, func(t *testing.T) {
			var status = http.StatusAccepted
			var response string
			var action, priority string
			var requests atomic.Int64
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if r.Method == "GET" {
					if r.URL.Path != "/alerts/alias" || r.URL.Query().Get("alias") != "tenant:monitor:42" {
						t.Error("unstable alias lookup")
					}
					w.WriteHeader(status)
					if response != "" {
						_, _ = io.WriteString(w, response)
					} else {
						_, _ = io.WriteString(w, `{"id":"alert-id"}`)
					}
					return
				}
				var payload map[string]json.RawMessage
				if json.NewDecoder(r.Body).Decode(&payload) != nil {
					t.Error("invalid payload")
				}
				if name == "pagerduty" {
					if string(payload["dedup_key"]) != `"tenant:monitor:42"` || string(payload["event_action"]) != `"`+action+`"` {
						t.Error("uncorrelated PagerDuty action")
					}
					if action == "trigger" && !strings.Contains(string(payload["payload"]), `"severity":"`+priority+`"`) {
						t.Error("severity lost")
					}
					if action != "trigger" && payload["payload"] != nil {
						t.Error("trigger payload in lifecycle action")
					}
				} else {
					if name == "opsgenie" {
						if r.Header.Get("Authorization") != "GenieKey fixture-token" {
							t.Error("Opsgenie auth")
						}
						if action != "trigger" && (r.URL.Path != "/v2/alerts/tenant:monitor:42/"+action || r.URL.Query().Get("identifierType") != "alias") {
							t.Error("Opsgenie alias action")
						}
						if action == "trigger" && (r.URL.Path != "/v2/alerts" || string(payload["alias"]) != `"tenant:monitor:42"` || string(payload["priority"]) != `"`+priority+`"`) {
							t.Error("Opsgenie trigger")
						}
					} else {
						user, pass, ok := r.BasicAuth()
						if !ok || user != "fixture@example.com" || pass != "fixture-token" {
							t.Error("JSM authentication")
						}
						if action == "trigger" {
							if r.URL.Path != "/alerts" || string(payload["alias"]) != `"tenant:monitor:42"` || string(payload["priority"]) != `"`+priority+`"` {
								t.Error("JSM trigger fields")
							}
						} else if r.URL.Path != "/alerts/alert-id/"+action {
							t.Error("JSM action path")
						}
					}
				}
				w.WriteHeader(status)
				if response != "" {
					_, _ = io.WriteString(w, response)
				} else if name == "pagerduty" {
					_, _ = io.WriteString(w, `{"status":"success","dedup_key":"tenant:monitor:42"}`)
				} else {
					_, _ = io.WriteString(w, `{"requestId":"request-id"}`)
				}
			}))
			defer server.Close()
			var driver incidentDriver
			var err error
			h := notification.HTTPConfig{AllowPrivate: true}
			if name == "pagerduty" {
				driver, err = pagerduty.New(pagerduty.Config{URL: server.URL, RoutingKey: strings.Repeat("a", 32), Key: "tenant:monitor:42", HTTP: h})
			} else if name == "jsm" {
				driver, err = jsm.New(jsm.Config{BaseURL: server.URL, CloudID: "cloud", Email: "fixture@example.com", Token: "fixture-token", Key: "tenant:monitor:42", HTTP: h})
			}
			if name == "opsgenie" {
				driver, err = opsgenie.New(opsgenie.Config{BaseURL: server.URL, Token: "fixture-token", Key: "tenant:monitor:42", HTTP: h})
			}
			if err != nil {
				t.Fatal(err)
			}
			defer driver.Close()
			action = "trigger"
			priority = "warning"
			if name != "pagerduty" {
				priority = "P3"
			}
			if err := notification.New(driver).Send(nil, notification.Message{Title: "Monitor renamed", Body: "down"}); err != nil {
				t.Fatal(err)
			}
			notificationtest.Sender(t, func() notification.Sender { return driver })
			for i, severity := range []notification.Severity{notification.SeverityInfo, notification.SeverityWarning, notification.SeverityError, notification.SeverityCritical} {
				priority = string(severity)
				if name != "pagerduty" {
					priority = []string{"P4", "P3", "P2", "P1"}[i]
				}
				if err := driver.Trigger(nil, notification.Incident{Key: "tenant:monitor:42", Title: "Renamed again", Body: "still down", Source: "monitor", Severity: severity}); err != nil {
					t.Fatal(err)
				}
			}
			action = "acknowledge"
			if err := driver.Acknowledge(nil, "tenant:monitor:42"); err != nil {
				t.Fatal(err)
			}
			action = "resolve"
			if name != "pagerduty" {
				action = "close"
			}
			if err := driver.Resolve(nil, "tenant:monitor:42"); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			before := requests.Load()
			for _, call := range []func() error{
				func() error { return driver.Trigger(ctx, notification.Incident{}) },
				func() error { return driver.Acknowledge(ctx, "valid") },
				func() error { return driver.Resolve(ctx, "valid") },
				func() error { return driver.Trigger(nil, notification.Incident{}) },
				func() error { return driver.Acknowledge(nil, "") },
				func() error { return driver.Resolve(nil, "") },
			} {
				if call() == nil {
					t.Fatal("invalid operation accepted")
				}
			}
			if before != requests.Load() {
				t.Fatal("invalid operation reached server")
			}
			for _, bad := range []string{"invalid", `{}`, `{"status":"success","dedup_key":"other"}`, `{"id":""}`} {
				response = bad
				if err := driver.Resolve(nil, "tenant:monitor:42"); !errors.Is(err, notification.ErrOutcomeUnknown) {
					t.Fatal("bad acknowledgement accepted", err)
				}
			}
			if name == "jsm" {
				action = "trigger"
				priority = "P3"
				if err := driver.Trigger(nil, notification.Incident{Key: "tenant:monitor:42", Title: "title", Body: "body"}); !errors.Is(err, notification.ErrOutcomeUnknown) {
					t.Fatal(err)
				}
			}
			response = ""
			action = "resolve"
			if name != "pagerduty" {
				action = "close"
			}
			status = 403
			if err := driver.Resolve(nil, "tenant:monitor:42"); !errors.Is(err, notification.ErrUnauthorized) {
				t.Fatal(err)
			}
			if name == "jsm" {
				status = 200
				response = `{"id":"` + strings.Repeat("x", 257) + `"}`
				if err := driver.Resolve(nil, "tenant:monitor:42"); !errors.Is(err, notification.ErrOutcomeUnknown) {
					t.Fatal(err)
				}
			}
		})
	}
}

// TestIncidentConstruction validates credentials, defaults and HTTP configuration without I/O.
func TestIncidentConstruction(t *testing.T) {
	for _, config := range []pagerduty.Config{{}, {RoutingKey: strings.Repeat("a", 32), URL: "bad"}, {RoutingKey: strings.Repeat("a", 32), Key: "bad\n"}, {RoutingKey: strings.Repeat("a", 32), HTTP: notification.HTTPConfig{Timeout: -1}}} {
		if _, err := pagerduty.New(config); err == nil {
			t.Fatal("invalid PagerDuty configuration")
		}
	}
	p, err := pagerduty.New(pagerduty.Config{RoutingKey: strings.Repeat("a", 32)})
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	if err := p.Send(nil, notification.Message{Title: "title", Body: "body"}); err == nil {
		t.Fatal("missing default key")
	}
	for _, config := range []jsm.Config{{}, {CloudID: "cloud", Email: "name <fixture@example.com>", Token: "token"}, {CloudID: "cloud", Email: "fixture@example.com", Token: "token", Priority: "P6"}, {CloudID: "cloud", Email: "fixture@example.com", Token: "token", BaseURL: "bad"}, {CloudID: "cloud", Email: "fixture@example.com", Token: "token", HTTP: notification.HTTPConfig{Timeout: -1}}} {
		if _, err := jsm.New(config); err == nil {
			t.Fatal("invalid JSM configuration")
		}
	}
	j, err := jsm.New(jsm.Config{CloudID: "cloud", Email: "fixture@example.com", Token: "token"})
	if err != nil {
		t.Fatal(err)
	}
	defer j.Close()
	if err := j.Send(nil, notification.Message{Title: "title", Body: "body"}); err == nil {
		t.Fatal("missing default key")
	}
}

// TestOpsgenieConstructionAndBounds covers legacy defaults and native content limits.
func TestOpsgenieConstructionAndBounds(t *testing.T) {
	for _, config := range []opsgenie.Config{{}, {Token: "token", Priority: "P6"}, {Token: "token", Source: strings.Repeat("x", 101)}, {Token: "token", BaseURL: "bad"}, {Token: "token", HTTP: notification.HTTPConfig{Timeout: -1}}} {
		if _, err := opsgenie.New(config); err == nil {
			t.Fatal("invalid legacy config")
		}
	}
	driver, err := opsgenie.New(opsgenie.Config{Token: "token"})
	if err != nil {
		t.Fatal(err)
	}
	defer driver.Close()
	if driver.Send(nil, notification.Message{Title: "title", Body: "body"}) == nil {
		t.Fatal("missing default key")
	}
	j, err := jsm.New(jsm.Config{Email: "fixture@example.com", CloudID: "cloud", Token: "token"})
	if err != nil {
		t.Fatal(err)
	}
	defer j.Close()
	for _, driver := range []notification.IncidentSender{driver, j} {
		for _, incident := range []notification.Incident{
			{Key: "stable", Title: strings.Repeat("x", 131), Body: "body"},
			{Key: "stable", Title: "title", Body: strings.Repeat("x", 15001)},
			{Key: "stable", Title: "title", Body: "body", Source: strings.Repeat("x", 101)},
		} {
			if err := driver.Trigger(nil, incident); !errors.Is(err, notification.ErrRender) {
				t.Fatal("native bound", err)
			}
		}
	}
}
