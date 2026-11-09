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
