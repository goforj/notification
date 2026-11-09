package notification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// request supplies safe deterministic application intent.
func request() Request {
	return Request{ID: "operation-1", Route: "ticket.updated", Recipients: []RecipientRef{{Type: "user", ID: "one"}}, Content: Content{Key: "ticket.changed", Values: []Value{String("ticket", "T-42")}}}
}

// delivery supplies one trusted resolved destination.
func delivery(ref RecipientRef, driver string) PlannedDelivery {
	return PlannedDelivery{Recipient: ref, Channel: "chat", Driver: driver, Endpoint: Endpoint{Identity: driver, Address: Address("https://example.com/secret")}, Locale: "en", TimeZone: "UTC"}
}

// setup composes the real dispatcher with controlled collaborators.
func setup(t *testing.T, policy Policy, plan Plan, drivers map[string]DriverFunc[TextPayload], options ...DispatchOption) *Dispatcher {
	t.Helper()
	adapters := []Adapter{}
	for name, driver := range drivers {
		adapters = append(adapters, NewAdapter("chat", name, "v1", Literal("Ticket", "Changed"), driver))
	}
	n, err := NewDispatcher(Scope{Application: "test"}, NewMemoryPlanner(map[string]Plan{"ticket.updated": plan}), []Route{{Name: "ticket.updated", Revision: "v1", Policy: policy}}, adapters, options...)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

// success is a typed transport fixture that acknowledges acceptance.
func success(context.Context, Delivery[TextPayload]) (Outcome, error) {
	return Outcome{Status: Accepted}, nil
}

// TestPolicies exercises recipient-local fallback and the complete success matrix.
func TestPolicies(t *testing.T) {
	statuses := []Status{Accepted, Delivered, Failed, Unknown}
	for _, policy := range []Policy{AllRequired, Any, FirstSuccess, BestEffort} {
		for _, a := range statuses {
			for _, b := range statuses {
				t.Run(string(policy)+"/"+string(a)+"/"+string(b), func(t *testing.T) {
					r := request()
					n := setup(t, policy, Plan{Deliveries: []PlannedDelivery{delivery(r.Recipients[0], "a"), delivery(r.Recipients[0], "b")}}, map[string]DriverFunc[TextPayload]{"a": func(context.Context, Delivery[TextPayload]) (Outcome, error) { return Outcome{Status: a}, nil }, "b": func(context.Context, Delivery[TextPayload]) (Outcome, error) { return Outcome{Status: b}, nil }})
					out, err := n.Send(context.Background(), r)
					want := accepted(a) || accepted(b)
					if policy == FirstSuccess && a == Unknown {
						want = false
					}
					if policy == AllRequired {
						want = accepted(a) && accepted(b)
					}
					if policy == BestEffort {
						want = true
					}
					if out.Succeeded != want || (err == nil) != want || len(out.Deliveries) != 2 {
						t.Fatalf("policy result: %+v %v", out, err)
					}
				})
			}
		}
	}
	r := request()
	r.Recipients = append(r.Recipients, RecipientRef{Type: "user", ID: "two"})
	var calls int
	n := setup(t, FirstSuccess, Plan{Deliveries: []PlannedDelivery{delivery(r.Recipients[0], "a"), delivery(r.Recipients[0], "b"), delivery(r.Recipients[1], "a"), delivery(r.Recipients[1], "b")}}, map[string]DriverFunc[TextPayload]{"a": func(context.Context, Delivery[TextPayload]) (Outcome, error) {
		calls++
		return Outcome{Status: Accepted}, nil
	}, "b": success})
	out, err := n.Send(nil, r)
	if err != nil || calls != 2 || out.Deliveries[1].Outcome.Status != Skipped || out.Deliveries[3].Outcome.Status != Skipped {
		t.Fatalf("recipient-local fallback: %+v %v", out, err)
	}
}

// TestJournal proves conflicts, partial retry, immutable results and explicit retention.
func TestJournal(t *testing.T) {
	r := request()
	journal := NewMemoryJournal()
	var a, b int
	n := setup(t, AllRequired, Plan{Deliveries: []PlannedDelivery{delivery(r.Recipients[0], "a"), delivery(r.Recipients[0], "b")}}, map[string]DriverFunc[TextPayload]{"a": func(context.Context, Delivery[TextPayload]) (Outcome, error) {
		a++
		return Outcome{Status: Accepted}, nil
	}, "b": func(context.Context, Delivery[TextPayload]) (Outcome, error) {
		b++
		if b == 1 {
			return Outcome{Status: Failed}, ErrRetryable
		}
		return Outcome{Status: Accepted}, nil
	}}, WithJournal(journal))
	out, err := n.Send(nil, r)
	if !errors.Is(err, ErrRetryable) {
		t.Fatal(err)
	}
	out.Deliveries[0].Outcome.Status = Failed
	out, err = n.Send(nil, r)
	if err != nil || a != 1 || b != 2 {
		t.Fatalf("partial retry: %d %d %v", a, b, err)
	}
	out.Deliveries[0].Channel = "mutated"
	out, err = n.Send(nil, r)
	if err != nil || out.Deliveries[0].Channel != "chat" || a != 1 {
		t.Fatal("completion alias")
	}
	changed := r.Clone()
	changed.Content.Values[0] = String("ticket", "different")
	if _, err := n.Send(nil, changed); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
	if err := journal.Forget(n.scope, r.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := n.Send(nil, changed); err != nil || a != 2 {
		t.Fatal("forget did not end deduplication")
	}
}

// TestConcurrentJournal arbitrates active submissions and protects active retention.
func TestConcurrentJournal(t *testing.T) {
	r := request()
	j := NewMemoryJournal()
	entered := make(chan struct{})
	release := make(chan struct{})
	n := setup(t, Any, Plan{Deliveries: []PlannedDelivery{delivery(r.Recipients[0], "a")}}, map[string]DriverFunc[TextPayload]{"a": func(context.Context, Delivery[TextPayload]) (Outcome, error) {
		close(entered)
		<-release
		return Outcome{Status: Accepted}, nil
	}}, WithJournal(j))
	done := make(chan struct{})
	go func() { defer close(done); _, _ = n.Send(nil, r) }()
	<-entered
	if _, err := n.Send(nil, r); !errors.Is(err, ErrInProgress) {
		t.Fatal(err)
	}
	if err := j.Forget(n.scope, r.ID); !errors.Is(err, ErrInProgress) {
		t.Fatal(err)
	}
	close(release)
	<-done
}

// TestRetryFallback verifies explicit advancement after a transient failure.
func TestRetryFallback(t *testing.T) {
	for _, advance := range []bool{false, true} {
		t.Run(fmt.Sprint(advance), func(t *testing.T) {
			r := request()
			n := setup(t, FirstSuccess, Plan{Deliveries: []PlannedDelivery{delivery(r.Recipients[0], "a"), delivery(r.Recipients[0], "b")}}, map[string]DriverFunc[TextPayload]{"a": func(context.Context, Delivery[TextPayload]) (Outcome, error) {
				return Outcome{Status: Failed}, ErrThrottled
			}, "b": success})
			route := n.routes[r.Route]
			route.ContinueOnRetryable = advance
			n.routes[r.Route] = route
			out, err := n.Send(nil, r)
			if (err == nil) != advance {
				t.Fatal(err)
			}
			if (out.Deliveries[1].Outcome.Status == Accepted) != advance {
				t.Fatal(out)
			}
		})
	}
}

// TestScopeAndRevisions separates caller fingerprints from execution snapshots.
func TestScopeAndRevisions(t *testing.T) {
	r := request()
	j := NewMemoryJournal()
	calls := 0
	driver := DriverFunc[TextPayload](func(context.Context, Delivery[TextPayload]) (Outcome, error) {
		calls++
		return Outcome{Status: Failed}, ErrRetryable
	})
	plan := Plan{Deliveries: []PlannedDelivery{delivery(r.Recipients[0], "a")}}
	n := setup(t, Any, plan, map[string]DriverFunc[TextPayload]{"a": driver}, WithJournal(j))
	_, _ = n.Send(nil, r)
	route := n.routes[r.Route]
	route.Revision = "v2"
	n.routes[r.Route] = route
	_, err := n.Send(nil, r)
	if !errors.Is(err, ErrRetryable) || calls != 2 {
		t.Fatal("route revision changed caller intent")
	}
	a := n.adapters["chat/a"]
	a.revision = "v2"
	n.adapters["chat/a"] = a
	if _, err := n.Send(nil, r); !errors.Is(err, ErrConflict) || calls != 2 {
		t.Fatal("unavailable pinned adapter executed")
	}
	n.scope.Tenant = "other"
	if _, err := n.Send(nil, r); !errors.Is(err, ErrRetryable) || calls != 3 {
		t.Fatal("tenant collision")
	}
	n.scope.Application = "other-app"
	_, _ = n.Send(nil, r)
	if calls != 4 {
		t.Fatal("application collision")
	}
}

// TestCancellation verifies preflight, partial and accepted outcomes preserve evidence.
func TestCancellation(t *testing.T) {
	r := request()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	n := setup(t, Any, Plan{}, map[string]DriverFunc[TextPayload]{"a": success})
	if _, err := n.Send(ctx, r); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	ctx, cancel = context.WithCancel(context.Background())
	n = setup(t, AllRequired, Plan{Deliveries: []PlannedDelivery{delivery(r.Recipients[0], "a"), delivery(r.Recipients[0], "b")}}, map[string]DriverFunc[TextPayload]{"a": func(context.Context, Delivery[TextPayload]) (Outcome, error) {
		cancel()
		return Outcome{Status: Accepted}, context.Canceled
	}, "b": success})
	out, err := n.Send(ctx, r)
	if !errors.Is(err, context.Canceled) || out.Deliveries[0].Outcome.Status != Accepted || out.Deliveries[1].Outcome.Status != Skipped {
		t.Fatalf("%+v %v", out, err)
	}
}

// TestCollaboratorErrors verifies redaction and panic-isolated observations.
func TestCollaboratorErrors(t *testing.T) {
	r := request()
	plan := Plan{Deliveries: []PlannedDelivery{delivery(r.Recipients[0], "a")}}
	n := setup(t, Any, plan, map[string]DriverFunc[TextPayload]{"a": func(context.Context, Delivery[TextPayload]) (Outcome, error) {
		return Outcome{Status: Failed}, errors.New("secret@example.com")
	}}, WithObserver(ObserverFunc(func(Observation) { panic("observer") })))
	_, err := n.Send(nil, r)
	if strings.Contains(err.Error(), "secret") {
		t.Fatal("provider prose leaked")
	}
	n.planner = PlannerFunc(func(context.Context, PlanningRequest) (Plan, error) { return Plan{}, errors.New("secret") })
	if _, err := n.Send(nil, r); !errors.Is(err, ErrInvalidPlan) {
		t.Fatal(err)
	}
	n.planner = PlannerFunc(func(context.Context, PlanningRequest) (Plan, error) { return Plan{}, context.DeadlineExceeded })
	if _, err := n.Send(nil, r); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
	var observations atomic.Int64
	n = setup(t, Any, plan, map[string]DriverFunc[TextPayload]{"a": success}, WithObserver(ObserverFunc(func(o Observation) {
		if o.Route != "ticket.updated" || o.Channel != "chat" || o.Driver != "a" || o.Status != Accepted {
			t.Error("observation labels")
		}
		observations.Add(1)
	})))
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, _ = n.Send(nil, r) }()
	}
	wg.Wait()
	if observations.Load() != 16 {
		t.Fatal("missing observations")
	}
}

// TestDispositions distinguishes suppression and deferral from provider acceptance.
func TestDispositions(t *testing.T) {
	r := request()
	for _, allow := range []bool{false, true} {
		n := setup(t, Any, Plan{Suppressions: []Disposition{{Recipient: r.Recipients[0]}}}, map[string]DriverFunc[TextPayload]{"a": success})
		route := n.routes[r.Route]
		route.AllowSuppressed = allow
		n.routes[r.Route] = route
		out, err := n.Send(nil, r)
		if out.Suppressed != 1 || (err == nil) != allow {
			t.Fatal(out, err)
		}
	}
	n := setup(t, Any, Plan{Deferrals: []Disposition{{Recipient: r.Recipients[0], Until: time.Now().Add(time.Hour)}}}, map[string]DriverFunc[TextPayload]{"a": success})
	out, err := n.Send(nil, r)
	if out.Deferred != 1 || !errors.Is(err, ErrDeferred) {
		t.Fatal(out, err)
	}
}

// TestRedactedFormatting checks all ordinary diagnostic representations.
func TestRedactedFormatting(t *testing.T) {
	a := Address("secret")
	for _, s := range []string{a.String(), fmt.Sprintf("%v %#v", a, a), String("key", "secret").String(), fmt.Sprintf("%#v", String("key", "secret"))} {
		if strings.Contains(s, "secret") {
			t.Fatal("format leak")
		}
	}
	b, err := json.Marshal(a)
	if err != nil || string(b) != `"[redacted]"` || a.Reveal() != "secret" {
		t.Fatal(string(b), err)
	}
}

// TestUnknownCancellationNeverRetries protects ambiguous provider acceptance.
func TestUnknownCancellationNeverRetries(t *testing.T) {
	r := request()
	var calls int
	n := setup(t, FirstSuccess, Plan{Deliveries: []PlannedDelivery{delivery(r.Recipients[0], "a"), delivery(r.Recipients[0], "b")}}, map[string]DriverFunc[TextPayload]{"a": func(context.Context, Delivery[TextPayload]) (Outcome, error) {
		calls++
		return Outcome{Status: Unknown}, context.Canceled
	}, "b": success}, WithJournal(NewMemoryJournal()))
	for i := 0; i < 2; i++ {
		out, err := n.Send(nil, r)
		if !errors.Is(err, context.Canceled) || !errors.Is(err, ErrOutcomeUnknown) || out.Deliveries[1].Outcome.Status != Skipped {
			t.Fatal(out, err)
		}
	}
	if calls != 1 {
		t.Fatal("unknown delivery repeated")
	}
}

// TestSuccessfulFallbackIsTerminal prevents duplicate submissions from returning
// to a preferred channel after a fallback already satisfied the route.
func TestSuccessfulFallbackIsTerminal(t *testing.T) {
	for _, policy := range []Policy{Any, FirstSuccess, BestEffort} {
		t.Run(string(policy), func(t *testing.T) {
			r := request()
			calls := 0
			n := setup(t, policy, Plan{Deliveries: []PlannedDelivery{delivery(r.Recipients[0], "a"), delivery(r.Recipients[0], "b")}}, map[string]DriverFunc[TextPayload]{"a": func(context.Context, Delivery[TextPayload]) (Outcome, error) {
				calls++
				return Outcome{Status: Failed}, ErrRetryable
			}, "b": success}, WithJournal(NewMemoryJournal()))
			route := n.routes[r.Route]
			route.ContinueOnRetryable = true
			n.routes[r.Route] = route
			for i := 0; i < 2; i++ {
				if _, err := n.Send(nil, r); err != nil {
					t.Fatal(err)
				}
			}
			if calls != 1 {
				t.Fatal("completed policy repeated primary delivery")
			}
		})
	}
}

// TestConflictingFailureEvidence prevents ambiguous custom transports from enabling duplicates.
func TestConflictingFailureEvidence(t *testing.T) {
	for _, cause := range []error{
		ErrOutcomeUnknown,
		errors.Join(ErrOutcomeUnknown, ErrRetryable),
		errors.Join(ErrOutcomeUnknown, context.Canceled),
		errors.New("private unclassified transport failure"),
	} {
		for _, advance := range []bool{false, true} {
			r := request()
			primary, backup := 0, 0
			n := setup(t, FirstSuccess, Plan{Deliveries: []PlannedDelivery{
				delivery(r.Recipients[0], "a"),
				delivery(r.Recipients[0], "b"),
			}}, map[string]DriverFunc[TextPayload]{
				"a": func(context.Context, Delivery[TextPayload]) (Outcome, error) {
					primary++
					return Outcome{Status: Failed}, cause
				},
				"b": func(context.Context, Delivery[TextPayload]) (Outcome, error) {
					backup++
					return Outcome{Status: Accepted}, nil
				},
			}, WithJournal(NewMemoryJournal()))
			route := n.routes[r.Route]
			route.ContinueOnRetryable = advance
			n.routes[r.Route] = route
			for attempt := 0; attempt < 2; attempt++ {
				result, err := n.Send(context.Background(), r)
				if !errors.Is(err, ErrOutcomeUnknown) || result.Succeeded || result.Deliveries[0].Outcome.Status != Unknown || result.Deliveries[1].Outcome.Status != Skipped {
					t.Fatalf("conflicting evidence allowed delivery: %+v, %v", result, err)
				}
			}
			if primary != 1 || backup != 0 {
				t.Fatalf("ambiguous work repeated: primary=%d backup=%d", primary, backup)
			}
		}
	}
}
