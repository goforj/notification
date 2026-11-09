// Package log provides a redacted local transport that never claims provider acceptance.
package log

import (
	"context"
	"log/slog"

	"github.com/goforj/notification"
)

// Driver writes categorical attempt information to a borrowed concurrency-safe logger.
type Driver struct{ logger *slog.Logger }

// New constructs a log driver. Logger is a required wired dependency.
func New(logger *slog.Logger) *Driver { return &Driver{logger} }

// Deliver logs no endpoint, key or payload and returns Unknown because logging is not delivery.
func (d *Driver) Deliver(ctx context.Context, delivery notification.Delivery[notification.TextPayload]) (notification.Outcome, error) {
	if err := ctx.Err(); err != nil {
		return notification.Outcome{Status: notification.Failed}, err
	}
	if err := delivery.Payload.Validate(); err != nil {
		return notification.Outcome{Status: notification.Failed}, err
	}
	d.logger.LogAttrs(ctx, slog.LevelInfo, "notification captured", slog.String("status", "captured"))
	return notification.Outcome{Status: notification.Unknown}, notification.ErrOutcomeUnknown
}

// Send captures a validated message without logging its content. A nil error
// means local capture succeeded and makes no claim about provider acceptance.
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
	d.logger.LogAttrs(ctx, slog.LevelInfo, "notification captured", slog.String("status", "captured"))
	return nil
}
