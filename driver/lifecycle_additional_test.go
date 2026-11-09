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
	"github.com/goforj/notification/driver/betterstack"
	"github.com/goforj/notification/driver/firehydrant"
	"github.com/goforj/notification/driver/goalert"
	"github.com/goforj/notification/driver/ilert"
	"github.com/goforj/notification/driver/incidentio"
	"github.com/goforj/notification/driver/pagertree"
	"github.com/goforj/notification/driver/signl4"
)

// correlatedSender exposes only the lifecycle shared by these native integrations.
type correlatedSender interface {
	nativeSender
	Trigger(context.Context, notification.Incident) error
	Resolve(context.Context, notification.IncidentKey) error
}

// TestAdditionalLifecycles verifies action payloads and stable correlation independently of wording.
func TestAdditionalLifecycles(t *testing.T) {
	cases := []struct {
		name, keyField, actionField, trigger, acknowledge, resolve string
		new                                                        func(string, notification.HTTPConfig) (correlatedSender, error)
	}{
		{"goalert", "dedup", "action", "", "", "close", func(url string, h notification.HTTPConfig) (correlatedSender, error) {
			return goalert.New(goalert.Config{URL: url, Token: "fixture", HTTP: h})
		}},
		{"signl4", "X-S4-ExternalID", "X-S4-Status", "new", "acknowledged", "resolved", func(url string, h notification.HTTPConfig) (correlatedSender, error) {
			return signl4.New(signl4.Config{URL: url, HTTP: h})
		}},
		{"pagertree", "Id", "event_type", "create", "acknowledge", "resolve", func(url string, h notification.HTTPConfig) (correlatedSender, error) {
			return pagertree.New(pagertree.Config{URL: url, HTTP: h})
		}},
		{"ilert", "alertKey", "eventType", "ALERT", "ACCEPT", "RESOLVE", func(url string, h notification.HTTPConfig) (correlatedSender, error) {
			return ilert.New(ilert.Config{URL: url, IntegrationKey: "fixture", HTTP: h})
		}},
		{"incidentio", "deduplication_key", "status", "firing", "", "resolved", func(url string, h notification.HTTPConfig) (correlatedSender, error) {
			return incidentio.New(incidentio.Config{BaseURL: url, Token: "fixture", SourceID: "source", HTTP: h})
		}},
		{"firehydrant", "idempotency_key", "status", "OPEN", "", "CLOSED", func(url string, h notification.HTTPConfig) (correlatedSender, error) {
			return firehydrant.New(firehydrant.Config{URL: url, HTTP: h})
		}},
	}
	for _, entry := range cases {
		t.Run(entry.name, func(t *testing.T) {
			captures := make(chan map[string]json.RawMessage, 16)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body map[string]json.RawMessage
				if json.NewDecoder(r.Body).Decode(&body) != nil {
					t.Error("invalid native event")
				}
				captures <- body
				if entry.name == "ilert" || entry.name == "incidentio" {
					w.WriteHeader(202)
				}
				if entry.name == "incidentio" {
					_ = json.NewEncoder(w).Encode(map[string]any{"status": "success", "deduplication_key": body["deduplication_key"]})
					if string(body["title"]) == "null" || len(body["title"]) <= 2 {
						t.Error("native title is required on recovery")
					}
				}
			}))
			defer server.Close()
			driver, err := entry.new(server.URL, notification.HTTPConfig{AllowPrivate: true})
			if err != nil {
				t.Fatal(err)
			}
			defer driver.Close()
			key := notification.IncidentKey("monitor/immutable-42")
			incident := notification.Incident{Key: key, Title: "Renamable monitor", Body: "Unavailable"}
			if err := driver.Trigger(nil, incident); err != nil {
				t.Fatal(err)
			}
			verifyEvent(t, <-captures, entry.keyField, entry.actionField, string(key), entry.trigger)
			if native, ok := driver.(notification.IncidentSender); ok {
				if entry.acknowledge == "" {
					t.Fatal("unsupported native acknowledgement advertised")
				}
				if err := native.Acknowledge(nil, key); err != nil {
					t.Fatal(err)
				}
				verifyEvent(t, <-captures, entry.keyField, entry.actionField, string(key), entry.acknowledge)
			} else if entry.acknowledge != "" {
				t.Fatal("missing native acknowledgement")
			}
			if err := driver.Resolve(nil, key); err != nil {
				t.Fatal(err)
			}
			verifyEvent(t, <-captures, entry.keyField, entry.actionField, string(key), entry.resolve)
			if err := driver.Send(nil, notification.Message{Title: "Title", Body: "Body"}); !errors.Is(err, notification.ErrInvalidRequest) {
				t.Fatal("ordinary Send guessed a correlation key", err)
			}
			if err := driver.Resolve(nil, ""); !errors.Is(err, notification.ErrInvalidRequest) {
				t.Fatal("invalid recovery key", err)
			}
			if err := driver.Trigger(nil, notification.Incident{Key: key, Body: "No title"}); !errors.Is(err, notification.ErrInvalidMessage) {
				t.Fatal("invalid incident content", err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			if err := driver.Resolve(ctx, key); !errors.Is(err, context.Canceled) {
				t.Fatal("recovery cancellation", err)
			}
			if entry.name == "goalert" {
				incident.Body = strings.Repeat("x", 6145)
				if err := driver.Trigger(nil, incident); !errors.Is(err, notification.ErrRender) {
					t.Fatal("GoAlert native detail bound", err)
				}
			}
			if entry.name == "firehydrant" {
				for _, severity := range []notification.Severity{notification.SeverityWarning, notification.SeverityError, notification.SeverityCritical} {
					incident.Severity = severity
					if err := driver.Trigger(nil, incident); err != nil {
						t.Fatal(err)
					}
					<-captures
				}
			}
		})
	}
}

// verifyEvent checks the provider's native action and the immutable application correlation.
func verifyEvent(t *testing.T, body map[string]json.RawMessage, keyField, actionField, key, action string) {
	t.Helper()
	var gotKey, gotAction string
	_ = json.Unmarshal(body[keyField], &gotKey)
	_ = json.Unmarshal(body[actionField], &gotAction)
	if gotKey != key || gotAction != action {
		t.Fatalf("native lifecycle identity/action: %q %q", gotKey, gotAction)
	}
}

// TestBetterStackIdentity preserves native IDs without pretending they are caller-assigned keys.
func TestBetterStackIdentity(t *testing.T) {
	var malformed atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v3/incidents" {
			w.WriteHeader(201)
		}
		if malformed.Load() {
			_, _ = io.WriteString(w, `{"data":{"id":"other"}}`)
			return
		}
		_, _ = io.WriteString(w, `{"data":{"id":"42","type":"incident"}}`)
	}))
	defer server.Close()
	driver, err := betterstack.New(betterstack.Config{BaseURL: server.URL, Token: "fixture", HTTP: notification.HTTPConfig{AllowPrivate: true}})
	if err != nil {
		t.Fatal(err)
	}
	defer driver.Close()
	id, err := driver.Create(nil, notification.Message{Body: "Summary without a title"})
	if err != nil || id != "42" {
		t.Fatal("native creation ID", id, err)
	}
	if err := driver.Acknowledge(nil, id); err != nil {
		t.Fatal(err)
	}
	if err := driver.Resolve(nil, id); err != nil {
		t.Fatal(err)
	}
	malformed.Store(true)
	if err := driver.Resolve(nil, id); !errors.Is(err, notification.ErrOutcomeUnknown) {
		t.Fatal("mismatched native ID", err)
	}
	for _, id := range []string{"", strings.Repeat("x", 257)} {
		if err := driver.Resolve(nil, id); !errors.Is(err, notification.ErrInvalidRequest) {
			t.Fatal("invalid native ID", err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := driver.Create(ctx, notification.Message{Body: "Canceled"}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if err := driver.Resolve(ctx, "42"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

// TestPagerTreeAuthentication keeps the optional integration token out of the event body.
func TestPagerTreeAuthentication(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("pagertree-token") != "fixture-token" {
			t.Error("integration token missing")
		}
	}))
	defer server.Close()
	driver, err := pagertree.New(pagertree.Config{URL: server.URL, Token: "fixture-token", Key: "monitor/42", HTTP: notification.HTTPConfig{AllowPrivate: true}})
	if err != nil {
		t.Fatal(err)
	}
	defer driver.Close()
	if err := driver.Send(nil, notification.Message{Title: "Alert", Body: "Unavailable"}); err != nil {
		t.Fatal(err)
	}
	if _, err := pagertree.New(pagertree.Config{URL: server.URL, Token: "invalid\n"}); !errors.Is(err, notification.ErrInvalidRequest) {
		t.Fatal("bad token accepted", err)
	}
}

// TestBetterStackActionResource requires the native incident resource type as well as the echoed ID.
func TestBetterStackActionResource(t *testing.T) {
	for _, body := range []string{`{"data":{"id":"42"}}`, `{"data":{"id":"42","type":"monitor"}}`} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, body) }))
		defer server.Close()
		driver, err := betterstack.New(betterstack.Config{Token: "fixture", BaseURL: server.URL, HTTP: notification.HTTPConfig{AllowPrivate: true}})
		if err != nil {
			t.Fatal(err)
		}
		defer driver.Close()
		for _, err := range []error{driver.Acknowledge(nil, "42"), driver.Resolve(nil, "42")} {
			if !errors.Is(err, notification.ErrOutcomeUnknown) {
				t.Fatal("wrong resource acknowledgement accepted", err)
			}
		}
	}
}
