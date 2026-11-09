package notification_test

import (
	"strings"
	"testing"

	"github.com/goforj/notification"
)

// TestIncidentValidation rejects unstable or unbounded correlation before transport.
func TestIncidentValidation(t *testing.T) {
	for _, key := range []notification.IncidentKey{"", "bad\nkey", notification.IncidentKey(strings.Repeat("x", 256))} {
		if key.Validate() == nil {
			t.Fatal("invalid key accepted")
		}
	}
	for _, incident := range []notification.Incident{
		{}, {Key: "stable", Body: "body"}, {Key: "stable", Title: "title"},
		{Key: "stable", Title: "title", Body: "body", Source: "bad\nsource"},
		{Key: "stable", Title: "title", Body: "body", Severity: "invented"},
	} {
		if incident.Validate() == nil {
			t.Fatal("invalid incident accepted")
		}
	}
	for _, severity := range []notification.Severity{"", notification.SeverityInfo, notification.SeverityWarning, notification.SeverityError, notification.SeverityCritical} {
		if err := (notification.Incident{Key: "tenant:monitor:42", Title: "title", Body: "body", Severity: severity}).Validate(); err != nil {
			t.Fatal(err)
		}
	}
}
