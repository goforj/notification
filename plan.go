package notification

import (
	"context"
	"time"
)

// Policy controls channel outcomes independently for each recipient.
type Policy string

const (
	// AllRequired requires acceptance on every planned channel for each recipient.
	AllRequired Policy = "all_required"
	// Any requires at least one accepted channel for each recipient and attempts all channels.
	Any Policy = "any"
	// FirstSuccess stops ordered fallback separately for each recipient after acceptance.
	FirstSuccess Policy = "first_success"
	// BestEffort tolerates failed attempts while preserving their outcomes.
	BestEffort Policy = "best_effort"
)

// Route declares immutable policy; callers cannot weaken it through requests.
type Route struct {
	Name                string
	Revision            string
	Policy              Policy
	AllowSuppressed     bool
	ContinueOnRetryable bool
}

// SecretAddress keeps trusted endpoint material out of ordinary formatting and JSON.
type SecretAddress struct{ value string }

// Address wraps an endpoint supplied by a trusted application planner.
func Address(value string) SecretAddress { return SecretAddress{value} }

// Reveal exposes the endpoint only to code intentionally handling transport secrets.
func (a SecretAddress) Reveal() string { return a.value }

// String redacts the endpoint from ordinary logs.
func (a SecretAddress) String() string { return "[redacted]" }

// GoString redacts the endpoint from Go syntax formatting.
func (a SecretAddress) GoString() string { return a.String() }

// MarshalJSON prevents ordinary snapshots from serializing endpoint secrets.
func (a SecretAddress) MarshalJSON() ([]byte, error) { return []byte(`"[redacted]"`), nil }

// Endpoint is a trusted resolved destination with a stable opaque identity.
type Endpoint struct {
	Identity string
	Address  SecretAddress
}

// PlannedDelivery binds one recipient to a compiled channel adapter and locale.
type PlannedDelivery struct {
	Recipient RecipientRef
	Channel   string
	Driver    string
	Endpoint  Endpoint
	Locale    string
	TimeZone  string
}

// Disposition records a recipient with no immediate delivery.
type Disposition struct {
	Recipient RecipientRef
	Until     time.Time
}

// Plan classifies every requested recipient in the exact supplied scope.
type Plan struct {
	Scope        Scope
	Deliveries   []PlannedDelivery
	Suppressions []Disposition
	Deferrals    []Disposition
}

// PlanningRequest carries authoritative scope and immutable route policy.
type PlanningRequest struct {
	Scope   Scope
	Request Request
	Route   Route
}

// Planner resolves application identities, preferences, and trusted endpoints.
type Planner interface {
	Plan(context.Context, PlanningRequest) (Plan, error)
}

// PlannerFunc adapts application-owned resolution without framework dependencies.
type PlannerFunc func(context.Context, PlanningRequest) (Plan, error)

// Plan invokes application-owned policy resolution.
func (f PlannerFunc) Plan(ctx context.Context, r PlanningRequest) (Plan, error) { return f(ctx, r) }

// Clone returns an independent plan snapshot.
func (p Plan) Clone() Plan {
	p.Deliveries = append([]PlannedDelivery(nil), p.Deliveries...)
	p.Suppressions = append([]Disposition(nil), p.Suppressions...)
	p.Deferrals = append([]Disposition(nil), p.Deferrals...)
	return p
}

// MemoryPlanner contains copied route plans for explicit local configuration.
type MemoryPlanner struct{ plans map[string]Plan }

// NewMemoryPlanner copies configured plans; application resolution can use PlannerFunc.
func NewMemoryPlanner(plans map[string]Plan) *MemoryPlanner {
	p := &MemoryPlanner{plans: make(map[string]Plan, len(plans))}
	for name, plan := range plans {
		p.plans[name] = plan.Clone()
	}
	return p
}

// Plan returns an independent configured plan with the caller's authoritative scope.
func (p *MemoryPlanner) Plan(ctx context.Context, r PlanningRequest) (Plan, error) {
	if err := ctx.Err(); err != nil {
		return Plan{}, err
	}
	plan, ok := p.plans[r.Route.Name]
	if !ok {
		return Plan{}, ErrInvalidPlan
	}
	plan = plan.Clone()
	plan.Scope = r.Scope
	return plan, nil
}
