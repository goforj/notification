package provider

import (
	"context"

	"github.com/goforj/notification"
)

// Incident centralizes safe validation while native drivers retain lifecycle wire contracts.
type Incident struct {
	client      *Client
	name        string
	key         notification.IncidentKey
	prepare     func(string, notification.Incident) (Request, error)
	acknowledge func(notification.Incident, []byte) error
	statuses    []int
}

// NewIncident constructs a native lifecycle without provisioning or contacting a provider.
func NewIncident(name string, config notification.HTTPConfig, key notification.IncidentKey, prepare func(string, notification.Incident) (Request, error), acknowledge func(notification.Incident, []byte) error, statuses ...int) (*Incident, error) {
	if key != "" && key.Validate() != nil {
		return nil, notification.ErrInvalidRequest
	}
	client, err := New(name, config)
	if err != nil {
		return nil, err
	}
	return &Incident{client: client, name: name, key: key, prepare: prepare, acknowledge: acknowledge, statuses: append([]int(nil), statuses...)}, nil
}

// Send uses the fixed application-owned key, so message wording never determines correlation.
func (d *Incident) Send(ctx context.Context, message notification.Message) error {
	return d.Trigger(ctx, notification.Incident{Key: d.key, Title: message.Title, Body: message.Body})
}

// Trigger validates explicit incident intent before constructing a native event.
func (d *Incident) Trigger(ctx context.Context, incident notification.Incident) error {
	return d.send(ctx, "trigger", incident)
}

// Acknowledge sends a native acknowledgement only when exposed by the concrete provider.
func (d *Incident) Acknowledge(ctx context.Context, key notification.IncidentKey) error {
	return d.send(ctx, "acknowledge", notification.Incident{Key: key})
}

// Resolve sends a native recovery only when exposed by the concrete provider.
func (d *Incident) Resolve(ctx context.Context, key notification.IncidentKey) error {
	return d.send(ctx, "resolve", notification.Incident{Key: key})
}

// send keeps cancellation, validation and sanitization identical across supported native actions.
func (d *Incident) send(ctx context.Context, action string, incident notification.Incident) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := incident.Key.Validate(); err != nil {
		return err
	}
	if action == "trigger" {
		if err := incident.Validate(); err != nil {
			return err
		}
	}
	request, err := d.prepare(action, incident)
	if err != nil {
		return notification.NewDeliveryError(d.name, notification.Outcome{Status: notification.Failed}, err)
	}
	var acknowledge func([]byte) error
	if d.acknowledge != nil {
		acknowledge = func(data []byte) error { return d.acknowledge(incident, data) }
	}
	return d.client.Request(ctx, request.Method, request.URL, request.Header, request.Body, acknowledge, d.statuses...)
}

// Close releases owned idle connections without closing borrowed transports.
func (d *Incident) Close() error { return d.client.Close() }
