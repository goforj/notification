// Package twilio submits SMS messages through Twilio's Messages REST resource.
package twilio

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
	"unicode/utf8"

	"github.com/goforj/notification"
	"github.com/goforj/notification/internal/provider"
)

// Config supplies a Twilio account and either a sender or Messaging Service.
// Addresses use E.164 phone numbers. BaseURL supports a trusted API proxy.
type Config struct {
	AccountSID          string
	AuthToken           string
	From                string
	To                  string
	MessagingServiceSID string
	BaseURL             string
	HTTP                notification.HTTPConfig
}

// Driver owns one immutable SMS recipient and bounded transport.
type Driver struct {
	client *provider.Client
	config Config
	auth   string
}

var accountPattern = regexp.MustCompile(`^AC[0-9a-fA-F]{32}$`)
var servicePattern = regexp.MustCompile(`^MG[0-9a-fA-F]{32}$`)
var messagePattern = regexp.MustCompile(`^(SM|MM)[0-9a-fA-F]{32}$`)
var phonePattern = regexp.MustCompile(`^\+[1-9][0-9]{1,14}$`)

// New validates SMS configuration without sending or provisioning a number.
func New(config Config) (*Driver, error) {
	if !accountPattern.MatchString(config.AccountSID) || !provider.Token(config.AuthToken) || !phonePattern.MatchString(config.To) || (config.From == "") == (config.MessagingServiceSID == "") || (config.From != "" && !phonePattern.MatchString(config.From)) || (config.MessagingServiceSID != "" && !servicePattern.MatchString(config.MessagingServiceSID)) {
		return nil, notification.ErrInvalidRequest
	}
	base, err := provider.BaseURL(config.BaseURL, "https://api.twilio.com")
	if err != nil {
		return nil, err
	}
	config.BaseURL = base
	client, err := provider.New("twilio", config.HTTP)
	if err != nil {
		return nil, err
	}
	return &Driver{client, config, "Basic " + base64.StdEncoding.EncodeToString([]byte(config.AccountSID+":"+config.AuthToken))}, nil
}

// Close releases owned idle HTTP connections.
func (d *Driver) Close() error { return d.client.Close() }

// Send submits one SMS and never interprets queue acceptance as handset delivery.
func (d *Driver) Send(ctx context.Context, message notification.Message) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := message.Validate(); err != nil {
		return err
	}
	text := provider.Text(message)
	if utf8.RuneCountInString(text) > 1600 {
		return notification.NewDeliveryError("twilio", notification.Outcome{Status: notification.Failed}, notification.ErrRender)
	}
	form := url.Values{"To": {d.config.To}, "Body": {text}}
	if d.config.From != "" {
		form.Set("From", d.config.From)
	} else {
		form.Set("MessagingServiceSid", d.config.MessagingServiceSID)
	}
	return d.client.Request(ctx, http.MethodPost, d.config.BaseURL+"/2010-04-01/Accounts/"+d.config.AccountSID+"/Messages.json", http.Header{"Content-Type": {"application/x-www-form-urlencoded"}, "Authorization": {d.auth}}, []byte(form.Encode()), func(data []byte) error {
		var ack struct {
			SID    string `json:"sid"`
			Status string `json:"status"`
		}
		if json.Unmarshal(data, &ack) != nil || !messagePattern.MatchString(ack.SID) {
			return notification.ErrOutcomeUnknown
		}
		switch ack.Status {
		case "accepted", "queued", "sending", "sent", "delivered":
			return nil
		case "failed", "undelivered", "canceled":
			return notification.ErrPermanent
		default:
			return notification.ErrOutcomeUnknown
		}
	})
}
