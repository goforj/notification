package notification

import (
	"context"
	"errors"
	"time"
)

// Sender is the synchronous application-facing delivery capability.
type Sender interface {
	Send(context.Context, Request) (Result, error)
}

// Notifier holds immutable catalogs and borrows concurrency-safe collaborators.
// It owns no background tasks or transport lifecycle resources.
type Notifier struct {
	scope    Scope
	planner  Planner
	routes   map[string]Route
	adapters map[string]Adapter
	journal  *MemoryJournal
	observer Observer
}

// Option configures an explicitly optional local capability.
type Option func(*Notifier)

// WithJournal enables process-local identity conflicts and partial retry convergence.
func WithJournal(journal *MemoryJournal) Option { return func(n *Notifier) { n.journal = journal } }

// WithObserver adds a borrowed non-blocking observer with panic isolation.
func WithObserver(observer Observer) Option { return func(n *Notifier) { n.observer = observer } }

// New constructs a scoped sender without performing I/O. Planner is a required
// wired dependency. Catalog entries must be bounded and unique.
func New(scope Scope, planner Planner, routes []Route, adapters []Adapter, options ...Option) (*Notifier, error) {
	if !validID(scope.Application) || (scope.Tenant != "" && !validID(scope.Tenant)) || len(routes) == 0 || len(routes) > 128 || len(adapters) == 0 || len(adapters) > 64 {
		return nil, ErrInvalidRequest
	}
	n := &Notifier{scope: scope, planner: planner, routes: map[string]Route{}, adapters: map[string]Adapter{}}
	for _, r := range routes {
		if !identifier.MatchString(r.Name) || !validID(r.Revision) || n.routes[r.Name].Name != "" {
			return nil, ErrInvalidRequest
		}
		switch r.Policy {
		case AllRequired, Any, FirstSuccess, BestEffort:
		default:
			return nil, ErrInvalidRequest
		}
		n.routes[r.Name] = r
	}
	for _, a := range adapters {
		key := a.channel + "/" + a.name
		if !identifier.MatchString(a.channel) || !identifier.MatchString(a.name) || !validID(a.revision) || a.send == nil || n.adapters[key].send != nil {
			return nil, ErrInvalidRequest
		}
		n.adapters[key] = a
	}
	for _, option := range options {
		option(n)
	}
	return n, nil
}

// Send resolves and executes each planned delivery once, with per-recipient policy.
// Repeated journaled calls retry only transient failures; unknown outcomes are
// retained for application reconciliation. Cancellation never erases acceptance.
func (n *Notifier) Send(ctx context.Context, r Request) (Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := r.Validate(); err != nil {
		return Result{}, err
	}
	r = r.Clone()
	route, ok := n.routes[r.Route]
	if !ok {
		return Result{}, ErrUnknownRoute
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	e := &entry{route: route}
	if n.journal != nil {
		var err error
		e, err = n.journal.reserve(n.scope, r)
		if err != nil {
			return Result{}, err
		}
		defer n.journal.release(e)
		if e.complete {
			return e.result.Clone(), e.err
		}
	}
	if !e.planned {
		plan, err := n.planner.Plan(ctx, PlanningRequest{n.scope, r.Clone(), route})
		if err != nil {
			return Result{}, safeError(err, ErrInvalidPlan)
		}
		plan = plan.Clone()
		if err := n.validatePlan(r, plan); err != nil {
			return Result{}, err
		}
		e.plan = plan
		e.planned = true
		e.route = route
		e.revisions = map[string]string{}
		for _, d := range plan.Deliveries {
			key := d.Channel + "/" + d.Driver
			e.revisions[key] = n.adapters[key].revision
		}
	}
	for key, revision := range e.revisions {
		if a, ok := n.adapters[key]; !ok || a.revision != revision {
			return Result{}, ErrConflict
		}
	}
	result := Result{RequestID: r.ID, Suppressed: len(e.plan.Suppressions), Deferred: len(e.plan.Deferrals)}
	prior := map[string]DeliveryResult{}
	for _, d := range e.result.Deliveries {
		prior[d.Key] = d
	}
	won := map[RecipientRef]bool{}
	blocked := map[RecipientRef]bool{}
	retry := false
	for _, d := range e.plan.Deliveries {
		key := identity(n.scope.Application, n.scope.Tenant, r.ID, d.Recipient.Type, d.Recipient.ID, r.Route, e.route.Revision, d.Channel, d.Driver, d.Endpoint.Identity)
		delivery := DeliveryResult{Key: key, Channel: d.Channel, Driver: d.Driver, Outcome: Outcome{Status: Skipped}}
		old, exists := prior[key]
		if exists && old.Outcome.Status != Skipped && !canRetry(old) {
			delivery = old
		} else if !(e.route.Policy == FirstSuccess && (won[d.Recipient] || blocked[d.Recipient])) {
			if err := ctx.Err(); err != nil {
				delivery.Error = err
				retry = true
			} else {
				adapter := n.adapters[d.Channel+"/"+d.Driver]
				start := time.Now()
				content := r.Clone().Content
				outcome, err := adapter.send(ctx, RenderRequest{content, d.Locale, d.TimeZone}, d.Endpoint, key)
				delivery.Outcome, delivery.Error = normalize(outcome, err)
				observe(n.observer, Observation{r.Route, d.Channel, d.Driver, delivery.Outcome.Status, time.Since(start)})
			}
		}
		if accepted(delivery.Outcome.Status) {
			won[d.Recipient] = true
		}
		if delivery.Outcome.Status == Unknown {
			blocked[d.Recipient] = true
		}
		if canRetry(delivery) {
			retry = true
			if !e.route.ContinueOnRetryable {
				blocked[d.Recipient] = true
			}
		}
		result.Deliveries = append(result.Deliveries, delivery)
	}
	err := evaluate(r, e.plan, e.route, result)
	result.Succeeded = err == nil
	e.result = result.Clone()
	e.err = err
	e.complete = result.Succeeded || !retry
	return result, err
}

// canRetry prevents ambiguous transport cancellation from repeating unknown work.
func canRetry(d DeliveryResult) bool { return d.Outcome.Status != Unknown && retryable(d.Error) }

// retryable identifies failures that a later explicit Send may resume.
func retryable(err error) bool {
	return errors.Is(err, ErrRetryable) || errors.Is(err, ErrThrottled) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

// normalize bounds transport evidence and strips arbitrary collaborator causes.
func normalize(outcome Outcome, err error) (Outcome, error) {
	if outcome.RetryAfter < 0 {
		outcome.RetryAfter = 0
	}
	if outcome.RetryAfter > 24*time.Hour {
		outcome.RetryAfter = 24 * time.Hour
	}
	switch outcome.Status {
	case Accepted, Delivered:
		return outcome, nil
	case Failed:
		if err == nil {
			err = ErrPermanent
		}
	case Unknown:
		safe := safeError(err, ErrOutcomeUnknown)
		if safe == ErrOutcomeUnknown {
			return outcome, ErrOutcomeUnknown
		}
		return outcome, errors.Join(ErrOutcomeUnknown, safe)
	default:
		outcome.Status = Unknown
		err = ErrOutcomeUnknown
	}
	return outcome, safeError(err, ErrOutcomeUnknown)
}

// validatePlan rejects scope changes, unclassified recipients and duplicate deliveries.
func (n *Notifier) validatePlan(r Request, p Plan) error {
	if p.Scope != n.scope || len(p.Deliveries) > 400 || len(p.Suppressions)+len(p.Deferrals) > len(r.Recipients) {
		return ErrInvalidPlan
	}
	refs := map[RecipientRef]int{}
	for _, ref := range r.Recipients {
		refs[ref] = 0
	}
	seen := map[string]bool{}
	for _, d := range p.Deliveries {
		count, ok := refs[d.Recipient]
		key := identity(d.Recipient.Type, d.Recipient.ID, d.Channel, d.Driver, d.Endpoint.Identity)
		if !ok || count < 0 || seen[key] || !validID(d.Endpoint.Identity) || len(d.Endpoint.Address.value) == 0 || len(d.Endpoint.Address.value) > 4096 || !validID(d.Locale) || n.adapters[d.Channel+"/"+d.Driver].send == nil {
			return ErrInvalidPlan
		}
		if !validID(d.TimeZone) {
			return ErrInvalidPlan
		}
		if _, err := time.LoadLocation(d.TimeZone); err != nil || d.TimeZone == "" || d.TimeZone == "Local" {
			return ErrInvalidPlan
		}
		refs[d.Recipient]++
		seen[key] = true
	}
	for _, group := range [][]Disposition{p.Suppressions, p.Deferrals} {
		for _, d := range group {
			count, ok := refs[d.Recipient]
			if !ok || count != 0 {
				return ErrInvalidPlan
			}
			refs[d.Recipient] = -1
		}
	}
	for _, d := range p.Suppressions {
		if !d.Until.IsZero() {
			return ErrInvalidPlan
		}
	}
	for _, d := range p.Deferrals {
		if d.Until.IsZero() {
			return ErrInvalidPlan
		}
	}
	for _, count := range refs {
		if count == 0 {
			return ErrInvalidPlan
		}
	}
	return nil
}

// evaluate applies success to every recipient instead of stopping global fanout.
func evaluate(r Request, p Plan, route Route, result Result) error {
	var failures []error
	if len(p.Deferrals) > 0 {
		failures = append(failures, ErrDeferred)
	}
	if len(p.Suppressions) > 0 && !route.AllowSuppressed {
		failures = append(failures, ErrSuppressed)
	}
	for _, ref := range r.Recipients {
		total, good := 0, 0
		var causes []error
		for i, d := range p.Deliveries {
			if d.Recipient != ref {
				continue
			}
			total++
			out := result.Deliveries[i]
			if accepted(out.Outcome.Status) {
				good++
			} else if out.Error != nil {
				causes = append(causes, out.Error)
			}
		}
		if total == 0 {
			continue
		}
		success := good > 0
		if route.Policy == AllRequired {
			success = good == total
		}
		if route.Policy == BestEffort {
			success = true
		}
		if !success {
			failures = append(failures, ErrNoDelivery)
			failures = append(failures, causes...)
		}
	}
	return errors.Join(failures...)
}
