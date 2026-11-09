package driver_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/goforj/notification"
	"github.com/goforj/notification/driver/incidentio"
	"github.com/goforj/notification/driver/whatsapp"
)

// TestWhatsAppTemplateWireLimit measures the serialized envelope, including escaping and metadata.
func TestWhatsAppTemplateWireLimit(t *testing.T) {
	var calls, size atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		size.Store(int64(len(body)))
		calls.Add(1)
		_, _ = io.WriteString(w, `{"contacts":[{"input":"12125550101","wa_id":"12125550101"}],"messages":[{"id":"wamid.fixture"}]}`)
	}))
	defer server.Close()
	driver, err := whatsapp.New(whatsapp.Config{Token: "fixture", PhoneNumberID: "123", To: "+12125550101", APIVersion: "v24.0", BaseURL: server.URL, HTTP: notification.HTTPConfig{AllowPrivate: true}})
	if err != nil {
		t.Fatal(err)
	}
	defer driver.Close()
	parameters := []string{"a", "a", "a", "a"}
	template := whatsapp.Template{Name: "ticket_updated", Language: "en_US", BodyParameters: parameters}
	if err := driver.SendTemplate(nil, template); err != nil {
		t.Fatal(err)
	}
	capacity := 16384 - int(size.Load()) + len(parameters)
	for i := range parameters {
		n := capacity / len(parameters)
		if i == len(parameters)-1 {
			n += capacity % len(parameters)
		}
		parameters[i] = strings.Repeat("a", n)
	}
	if err := driver.SendTemplate(nil, template); err != nil || size.Load() != 16384 {
		t.Fatal("exact wire boundary rejected", size.Load(), err)
	}
	parameters[0] += "\""
	err = driver.SendTemplate(nil, template)
	if !errors.Is(err, notification.ErrRender) || calls.Load() != 2 {
		t.Fatal("escaped envelope exceeded bound or reached transport", calls.Load(), err)
	}
}

// TestIncidentIOResponseCorrelation prevents another incident's success from proving this call succeeded.
func TestIncidentIOResponseCorrelation(t *testing.T) {
	var wrong atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var event struct {
			Key string `json:"deduplication_key"`
		}
		if json.NewDecoder(r.Body).Decode(&event) != nil {
			t.Error("invalid native event")
		}
		if wrong.Load() {
			event.Key = "another/incident"
		}
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "success", "deduplication_key": event.Key})
	}))
	defer server.Close()
	driver, err := incidentio.New(incidentio.Config{Token: "fixture", SourceID: "source", BaseURL: server.URL, HTTP: notification.HTTPConfig{AllowPrivate: true}})
	if err != nil {
		t.Fatal(err)
	}
	defer driver.Close()
	wrong.Store(true)
	for _, err := range []error{driver.Trigger(nil, notification.Incident{Key: "monitor/42", Title: "Failure", Body: "Unavailable"}), driver.Resolve(nil, "monitor/42")} {
		var detail *notification.DeliveryError
		if !errors.Is(err, notification.ErrOutcomeUnknown) || !errors.As(err, &detail) || detail.Status() != notification.Unknown {
			t.Fatal("mismatched native key accepted", err)
		}
	}
	wrong.Store(false)
	var workers sync.WaitGroup
	for i := 0; i < 16; i++ {
		workers.Add(1)
		go func(i int) {
			defer workers.Done()
			err := driver.Trigger(context.Background(), notification.Incident{Key: notification.IncidentKey(fmt.Sprintf("monitor/%d", i)), Title: "Failure", Body: "Unavailable"})
			if err != nil {
				t.Error("concurrent request correlation", err)
			}
		}(i)
	}
	workers.Wait()
}
