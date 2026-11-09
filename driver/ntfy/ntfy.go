// Package ntfy publishes native JSON messages to ntfy servers.
package ntfy

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"regexp"

	"github.com/goforj/notification"
	"github.com/goforj/notification/internal/provider"
)

// Config supplies a trusted topic and authentication. URL defaults to ntfy.sh.
// Priority zero uses the server default; nonzero priority must be between one and five.
type Config struct {
	URL      string
	Topic    string
	Token    string
	Username string
	Password string
	Priority int
	HTTP     notification.HTTPConfig
}

// Driver owns a transport and immutable publish configuration.
type Driver struct {
	client   *provider.Client
	url      string
	topic    string
	auth     string
	priority int
}

var topicName = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// New validates ntfy configuration without contacting the server.
func New(config Config) (*Driver, error) {
	endpoint, err := provider.BaseURL(config.URL, "https://ntfy.sh")
	if err != nil || !topicName.MatchString(config.Topic) || config.Priority < 0 || config.Priority > 5 {
		return nil, notification.ErrInvalidRequest
	}
	auth := ""
	if config.Token != "" {
		if !provider.Token(config.Token) || config.Username != "" || config.Password != "" {
			return nil, notification.ErrInvalidRequest
		}
		auth = "Bearer " + config.Token
	} else if config.Username != "" || config.Password != "" {
		if !provider.Token(config.Username) || !provider.Token(config.Password) {
			return nil, notification.ErrInvalidRequest
		}
		auth = "Basic " + base64.StdEncoding.EncodeToString([]byte(config.Username+":"+config.Password))
	}
	client, err := provider.New("ntfy", config.HTTP)
	if err != nil {
		return nil, err
	}
	return &Driver{client, endpoint + "/", config.Topic, auth, config.Priority}, nil
}

// Close releases only owned idle connections; injected transports remain caller-owned.
func (d *Driver) Close() error { return d.client.Close() }

// Send requires a native message acknowledgement for the configured topic.
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
	body := provider.JSON(struct {
		Topic    string `json:"topic"`
		Title    string `json:"title,omitempty"`
		Message  string `json:"message"`
		Priority int    `json:"priority,omitempty"`
	}{d.topic, message.Title, message.Body, d.priority})
	headers := http.Header{"Content-Type": []string{"application/json"}}
	if d.auth != "" {
		headers.Set("Authorization", d.auth)
	}
	return d.client.Request(ctx, http.MethodPost, d.url, headers, body, func(data []byte) error {
		var ack struct {
			ID    string `json:"id"`
			Event string `json:"event"`
			Topic string `json:"topic"`
		}
		if json.Unmarshal(data, &ack) != nil || ack.ID == "" || ack.Event != "message" || ack.Topic != d.topic {
			return notification.ErrOutcomeUnknown
		}
		return nil
	})
}
