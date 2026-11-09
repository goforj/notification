package driver_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/goforj/notification"
	"github.com/goforj/notification/driver/clicksend"
	"github.com/goforj/notification/driver/vonage"
)

// TestVonageSegmentEvidence preserves accepted parts and refuses to retry partial or ambiguous submissions.
func TestVonageSegmentEvidence(t *testing.T) {
	for _, entry := range []struct {
		name, body string
		want       error
	}{
		{"all accepted", `{"message-count":"2","messages":[{"status":"0","message-id":"first","to":"12125550101"},{"status":"0","message-id":"second","to":"12125550101"}]}`, nil},
		{"partial rejection", `{"message-count":"2","messages":[{"status":"0","message-id":"first","to":"12125550101"},{"status":"2"}]}`, notification.ErrOutcomeUnknown},
		{"all throttled", `{"message-count":"2","messages":[{"status":"1"},{"status":"1"}]}`, notification.ErrThrottled},
		{"unauthorized", `{"message-count":"1","messages":[{"status":"4"}]}`, notification.ErrUnauthorized},
		{"retryable", `{"message-count":"1","messages":[{"status":"5"}]}`, notification.ErrRetryable},
		{"permanent", `{"message-count":"1","messages":[{"status":"2"}]}`, notification.ErrPermanent},
		{"missing count", `{"messages":[{"status":"0","message-id":"first","to":"12125550101"}]}`, notification.ErrOutcomeUnknown},
		{"wrong count", `{"message-count":"2","messages":[{"status":"0","message-id":"first","to":"12125550101"}]}`, notification.ErrOutcomeUnknown},
		{"missing status", `{"message-count":"1","messages":[{"message-id":"first"}]}`, notification.ErrOutcomeUnknown},
		{"empty status", `{"message-count":"1","messages":[{"status":""}]}`, notification.ErrOutcomeUnknown},
		{"unknown status", `{"message-count":"1","messages":[{"status":"future"}]}`, notification.ErrOutcomeUnknown},
		{"wrong recipient", `{"message-count":"1","messages":[{"status":"0","message-id":"first","to":"12125550102"}]}`, notification.ErrOutcomeUnknown},
		{"missing ID", `{"message-count":"1","messages":[{"status":"0","to":"12125550101"}]}`, notification.ErrOutcomeUnknown},
		{"duplicate IDs", `{"message-count":"2","messages":[{"status":"0","message-id":"first","to":"12125550101"},{"status":"0","message-id":"first","to":"12125550101"}]}`, notification.ErrOutcomeUnknown},
		{"negative status", `{"message-count":"1","messages":[{"status":"-1"}]}`, notification.ErrOutcomeUnknown},
		{"zero count", `{"message-count":"0","messages":[]}`, notification.ErrOutcomeUnknown},
		{"null count", `{"message-count":null,"messages":[]}`, notification.ErrOutcomeUnknown},
		{"too many parts", `{"message-count":"65","messages":[]}`, notification.ErrOutcomeUnknown},
	} {
		t.Run(entry.name, func(t *testing.T) {
			driver, err := vonage.New(vonage.Config{APIKey: "fixture", APISecret: "fixture", From: "Helpdesk", To: "+12125550101", HTTP: notification.HTTPConfig{Transport: responseTransport(entry.body)}})
			if err != nil {
				t.Fatal(err)
			}
			defer driver.Close()
			err = driver.Send(nil, notification.Message{Body: strings.Repeat("a", 100)})
			if entry.want == nil {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if !errors.Is(err, entry.want) {
				t.Fatal("wrong segment classification", err)
			}
			if entry.want == notification.ErrOutcomeUnknown {
				var calls atomic.Int64
				fallback := notification.Fallback(driver, notification.SenderFunc(func(context.Context, notification.Message) error { calls.Add(1); return nil }))
				if err := fallback.Send(nil, notification.Message{Body: "body"}); !errors.Is(err, entry.want) || calls.Load() != 0 {
					t.Fatal("ambiguous submission advanced fallback", err)
				}
			}
		})
	}
}

// TestClickSendRejectionEvidence distinguishes native throttling, credential rejection and incomplete evidence.
func TestClickSendRejectionEvidence(t *testing.T) {
	for _, entry := range []struct {
		code string
		want error
	}{
		{"THROTTLED", notification.ErrThrottled},
		{"INVALID_CREDENTIALS", notification.ErrUnauthorized},
		{"MISSING_CREDENTIALS", notification.ErrUnauthorized},
		{"INVALID_RECIPIENT", notification.ErrPermanent},
		{"INTERNAL_ERROR", notification.ErrOutcomeUnknown},
		{"SOMETHING_IS_WRONG", notification.ErrOutcomeUnknown},
		{"", notification.ErrOutcomeUnknown},
		{"FUTURE_STATUS", notification.ErrOutcomeUnknown},
	} {
		for _, body := range []string{fmt.Sprintf(`{"response_code":%q}`, entry.code), fmt.Sprintf(`{"response_code":"SUCCESS","data":{"messages":[{"message_id":"first","status":%q}]}}`, entry.code)} {
			driver, err := clicksend.New(clicksend.Config{Username: "fixture", APIKey: "fixture", To: "+12125550101", HTTP: notification.HTTPConfig{Transport: responseTransport(body)}})
			if err != nil {
				t.Fatal(err)
			}
			err = driver.Send(nil, notification.Message{Body: "body"})
			_ = driver.Close()
			if !errors.Is(err, entry.want) {
				t.Fatal("wrong native rejection classification", entry.code, err)
			}
		}
	}
}
