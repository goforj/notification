package notification

import "context"

// Payload validates channel-specific bounds before a transport sees content.
type Payload interface{ Validate() error }

// RenderRequest contains one recipient's independent content and render context.
type RenderRequest struct {
	Content  Content
	Locale   string
	TimeZone string
}

// Renderer converts semantic intent into a concrete channel payload.
type Renderer[P Payload] interface {
	Render(context.Context, RenderRequest) (P, error)
}

// RendererFunc adapts a typed application rendering function.
type RendererFunc[P Payload] func(context.Context, RenderRequest) (P, error)

// Render invokes the configured typed renderer.
func (f RendererFunc[P]) Render(ctx context.Context, r RenderRequest) (P, error) { return f(ctx, r) }

// Delivery carries a typed payload and stable provider idempotency key.
type Delivery[P Payload] struct {
	Payload  P
	Endpoint Endpoint
	Key      string
}

// Driver performs one transport call without hidden retries.
type Driver[P Payload] interface {
	Deliver(context.Context, Delivery[P]) (Outcome, error)
}

// DriverFunc adapts a typed transport function.
type DriverFunc[P Payload] func(context.Context, Delivery[P]) (Outcome, error)

// Deliver invokes the configured transport function.
func (f DriverFunc[P]) Deliver(ctx context.Context, d Delivery[P]) (Outcome, error) { return f(ctx, d) }

// Adapter pairs a renderer and driver through a private typed closure.
type Adapter struct {
	channel  string
	name     string
	revision string
	send     func(context.Context, RenderRequest, Endpoint, string) (Outcome, error)
}

// NewAdapter enforces matching renderer and driver payload types at compile time.
// Required collaborators are wired dependencies and must be non-nil.
func NewAdapter[P Payload](channel, name, revision string, renderer Renderer[P], driver Driver[P]) Adapter {
	return Adapter{channel, name, revision, func(ctx context.Context, r RenderRequest, e Endpoint, key string) (Outcome, error) {
		payload, err := renderer.Render(ctx, r)
		if err != nil {
			return Outcome{Status: Failed}, safeError(err, ErrRender)
		}
		if err := payload.Validate(); err != nil {
			return Outcome{Status: Failed}, ErrRender
		}
		return driver.Deliver(ctx, Delivery[P]{payload, e, key})
	}}
}
