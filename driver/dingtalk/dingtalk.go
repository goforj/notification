// Package dingtalk submits text to a DingTalk custom robot with optional signing.
package dingtalk

import (
	"context"
	"github.com/goforj/notification"
	"github.com/goforj/notification/internal/provider"
	"net/http"
	"strconv"
	"time"
)

// Config supplies a robot webhook URL and optional SEC signing secret.
type Config struct {
	URL    string
	Secret string
	HTTP   notification.HTTPConfig
}

// Driver owns immutable robot configuration and a bounded transport.
type Driver struct {
	client *provider.Client
	config Config
}

// New validates configuration without contacting the robot.
func New(config Config) (*Driver, error) {
	if provider.Endpoint(config.URL) != nil || (config.Secret != "" && !provider.Token(config.Secret)) {
		return nil, notification.ErrInvalidRequest
	}
	client, err := provider.New("dingtalk", config.HTTP)
	if err != nil {
		return nil, err
	}
	return &Driver{client, config}, nil
}

// Close releases owned idle HTTP connections.
func (d *Driver) Close() error { return d.client.Close() }

// Send computes the signature at send time and validates the native error code.
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
	endpoint := d.config.URL
	if d.config.Secret != "" {
		timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
		endpoint = provider.Query(endpoint, map[string]string{"timestamp": timestamp, "sign": provider.Signature(d.config.Secret, timestamp+"\n"+d.config.Secret)})
	}
	body := provider.JSON(struct {
		Type string `json:"msgtype"`
		Text struct {
			Content string `json:"content"`
		} `json:"text"`
	}{"text", struct {
		Content string `json:"content"`
	}{text}})
	return d.client.Request(ctx, http.MethodPost, endpoint, http.Header{"Content-Type": {"application/json"}}, body, provider.RobotAck("errcode"))
}
