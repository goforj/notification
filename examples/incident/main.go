// Command incident exercises PagerDuty's protocol against a local fixture.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/goforj/notification"
	"github.com/goforj/notification/driver/pagerduty"
)

// main uses fake credentials and a local acknowledgement, never a hosted account.
func main() {
	server := httptest.NewServer(http.HandlerFunc(acknowledge))
	defer server.Close()
	driver, err := pagerduty.New(pagerduty.Config{
		RoutingKey: strings.Repeat("a", 32),
		URL:        server.URL,
		HTTP: notification.HTTPConfig{
			AllowPrivate: true,
		},
	})
	if err != nil {
		panic(err)
	}
	defer driver.Close()
	// snippet:start
	ctx := context.Background()
	key := notification.IncidentKey("tenant:42:monitor:17")
	err = driver.Trigger(ctx, notification.Incident{
		Key:      key,
		Title:    "API unavailable",
		Body:     "Health check failed.",
		Source:   "health-checks",
		Severity: notification.SeverityCritical,
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("trigger accepted by local fixture")
	// trigger accepted by local fixture

	if err := driver.Acknowledge(ctx, key); err != nil {
		panic(err)
	}
	if err := driver.Resolve(ctx, key); err != nil {
		panic(err)
	}
	fmt.Println("acknowledge and resolve use the same stable key")
	// acknowledge and resolve use the same stable key
	// snippet:end
}

// acknowledge supplies the native fixture response without implying hosted verification.
func acknowledge(w http.ResponseWriter, r *http.Request) {
	var event struct {
		Key string `json:"dedup_key"`
	}
	if json.NewDecoder(r.Body).Decode(&event) != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(struct {
		Status string `json:"status"`
		Key    string `json:"dedup_key"`
	}{"success", event.Key})
}
