// Package whatsapp sends native WhatsApp Cloud API text and approved template messages.
package whatsapp

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/goforj/notification"
	"github.com/goforj/notification/internal/provider"
)

// Config fixes a Meta Cloud API version, sending phone identity and E.164 recipient.
// APIVersion is explicit so an application controls provider-version migration.
type Config struct {
	Token         string
	PhoneNumberID string
	To            string
	APIVersion    string
	BaseURL       string
	HTTP          notification.HTTPConfig
}

// Template contains an existing approved template and optional plain text body parameters.
// This capability does not model media headers, buttons or template provisioning.
type Template struct {
	Name           string
	Language       string
	BodyParameters []string
}

// Driver borrows its optional transport and keeps credentials separate from message content.
type Driver struct {
	sender   *provider.Sender
	endpoint string
	headers  http.Header
	to       string
}

// New validates trusted configuration without sending, registering or looking up a phone number.
func New(config Config) (*Driver, error) {
	base, err := provider.BaseURL(config.BaseURL, "https://graph.facebook.com")
	phoneID, _ := regexp.MatchString(`^[0-9]{1,64}$`, config.PhoneNumberID)
	version, _ := regexp.MatchString(`^v[0-9]{1,3}\.[0-9]{1,2}$`, config.APIVersion)
	if err != nil || !provider.Token(config.Token) || !phoneID || !version || !provider.Phone(config.To) {
		return nil, notification.ErrInvalidRequest
	}
	to := strings.TrimPrefix(config.To, "+")
	endpoint := base + "/" + config.APIVersion + "/" + config.PhoneNumberID + "/messages"
	headers := http.Header{"Authorization": {"Bearer " + config.Token}}
	sender, err := provider.NewSender("whatsapp", config.HTTP, func(_ context.Context, message notification.Message) (provider.Request, error) {
		text := provider.Text(message)
		if utf8.RuneCountInString(text) > 4096 {
			return provider.Request{}, notification.ErrRender
		}
		return provider.PostJSON(endpoint, headers, struct {
			Product string `json:"messaging_product"`
			To      string `json:"to"`
			Type    string `json:"type"`
			Text    struct {
				Body    string `json:"body"`
				Preview bool   `json:"preview_url"`
			} `json:"text"`
		}{"whatsapp", to, "text", struct {
			Body    string `json:"body"`
			Preview bool   `json:"preview_url"`
		}{text, false}}), nil
	}, func(data []byte) error {
		var ack struct {
			Contacts []struct {
				Input string `json:"input"`
				ID    string `json:"wa_id"`
			} `json:"contacts"`
			Messages []struct {
				ID string `json:"id"`
			} `json:"messages"`
		}
		if json.Unmarshal(data, &ack) != nil || len(ack.Contacts) != 1 || strings.TrimPrefix(ack.Contacts[0].Input, "+") != to || ack.Contacts[0].ID == "" || len(ack.Messages) != 1 || !strings.HasPrefix(ack.Messages[0].ID, "wamid.") {
			return notification.ErrOutcomeUnknown
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &Driver{sender: sender, endpoint: endpoint, headers: headers, to: to}, nil
}

// Send submits free-form text within the provider's allowed customer-service window.
// Use SendTemplate for approved business-initiated messages outside that window.
func (d *Driver) Send(ctx context.Context, message notification.Message) error {
	return d.sender.Send(ctx, message)
}

// SendTemplate submits one approved template without guessing its name from title or body.
func (d *Driver) SendTemplate(ctx context.Context, template Template) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	valid, _ := regexp.MatchString(`^[a-z0-9_]{1,512}$`, template.Name)
	if !valid || !provider.Token(template.Language) || len(template.Language) > 64 || len(template.BodyParameters) > 32 {
		return notification.ErrInvalidRequest
	}
	type parameter struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	type component struct {
		Type       string      `json:"type"`
		Parameters []parameter `json:"parameters"`
	}
	parameters := make([]parameter, 0, len(template.BodyParameters))
	total := 0
	for _, value := range template.BodyParameters {
		total += len(value)
		if !utf8.ValidString(value) || len(value) > 4096 || total > 16384 {
			return notification.ErrInvalidMessage
		}
		parameters = append(parameters, parameter{Type: "text", Text: value})
	}
	var components []component
	if len(parameters) > 0 {
		components = []component{{Type: "body", Parameters: parameters}}
	}
	request := provider.PostJSON(d.endpoint, d.headers, struct {
		Product  string `json:"messaging_product"`
		To       string `json:"to"`
		Type     string `json:"type"`
		Template struct {
			Name     string `json:"name"`
			Language struct {
				Code string `json:"code"`
			} `json:"language"`
			Components []component `json:"components,omitempty"`
		} `json:"template"`
	}{"whatsapp", d.to, "template", struct {
		Name     string `json:"name"`
		Language struct {
			Code string `json:"code"`
		} `json:"language"`
		Components []component `json:"components,omitempty"`
	}{template.Name, struct {
		Code string `json:"code"`
	}{template.Language}, components}})
	if len(request.Body) > 16384 {
		return notification.NewDeliveryError("whatsapp", notification.Outcome{Status: notification.Failed}, notification.ErrRender)
	}
	return d.sender.Request(ctx, request)
}

// Close releases only owned idle connections; injected transports remain caller-owned.
func (d *Driver) Close() error { return d.sender.Close() }
