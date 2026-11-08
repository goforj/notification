package notification

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// TestRequestValidation directly exercises each bounded caller validation branch.
func TestRequestValidation(t *testing.T) {
	tests := []struct {
		name   string
		change func(*Request)
	}{
		{"id", func(r *Request) { r.ID = "" }}, {"id_control", func(r *Request) { r.ID = "one\x01" }}, {"id_utf8", func(r *Request) { r.ID = string([]byte{255}) }}, {"id_space", func(r *Request) { r.ID = " one" }}, {"id_size", func(r *Request) { r.ID = strings.Repeat("x", 257) }},
		{"route", func(r *Request) { r.Route = "Bad" }}, {"key", func(r *Request) { r.Content.Key = "Bad" }}, {"no_recipients", func(r *Request) { r.Recipients = nil }}, {"many_recipients", func(r *Request) { r.Recipients = make([]RecipientRef, 101) }}, {"many_values", func(r *Request) { r.Content.Values = make([]Value, 65) }}, {"many_tags", func(r *Request) { r.Tags = make([]string, 17) }},
		{"recipient_type", func(r *Request) { r.Recipients[0].Type = "" }}, {"recipient_id", func(r *Request) { r.Recipients[0].ID = "" }}, {"duplicate_recipient", func(r *Request) { r.Recipients = append(r.Recipients, r.Recipients[0]) }},
		{"value_name", func(r *Request) { r.Content.Values = []Value{String("Bad", "v")} }}, {"duplicate_value", func(r *Request) { r.Content.Values = append(r.Content.Values, r.Content.Values[0]) }}, {"value_utf8", func(r *Request) { r.Content.Values = []Value{String("a", string([]byte{255}))} }}, {"value_size", func(r *Request) { r.Content.Values = []Value{String("a", strings.Repeat("x", 8193))} }}, {"total_values", func(r *Request) {
			r.Content.Values = []Value{String("a", strings.Repeat("x", 8192)), String("b", strings.Repeat("x", 8192)), String("c", strings.Repeat("x", 8192)), String("d", strings.Repeat("x", 8192)), String("e", "x")}
		}},
		{"zero_value", func(r *Request) { r.Content.Values = []Value{{name: "a"}} }}, {"url_parse", func(r *Request) { r.Content.Values = []Value{URL("a", "http://%xx")} }}, {"url_scheme", func(r *Request) { r.Content.Values = []Value{URL("a", "javascript:alert(1)")} }}, {"url_host", func(r *Request) { r.Content.Values = []Value{URL("a", "https:/path")} }}, {"url_user", func(r *Request) { r.Content.Values = []Value{URL("a", "https://u:p@example.com")} }},
		{"tag", func(r *Request) { r.Tags = []string{"Bad"} }}, {"tag_duplicate", func(r *Request) { r.Tags = []string{"a", "a"} }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := request()
			tt.change(&r)
			if !errors.Is(r.Validate(), ErrInvalidRequest) {
				t.Fatal("invalid request accepted")
			}
		})
	}
	r := request()
	r.Content.Values = []Value{Integer("attempt", 2), Boolean("urgent", true), Time("at", time.Unix(0, 0)), URL("url", "https://example.com"), String("text", "hello")}
	r.Tags = []string{"one"}
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, v := range r.Content.Values {
		if v.Name() == "" || v.Kind() == "" || v.Text() == "" {
			t.Fatal("missing scalar representation")
		}
	}
	clone := r.Clone()
	clone.Content.Values[0] = String("changed", "v")
	clone.Recipients[0].ID = "changed"
	clone.Tags[0] = "changed"
	if r.Content.Values[0].Name() != "attempt" || r.Recipients[0].ID != "one" || r.Tags[0] != "one" {
		t.Fatal("request alias")
	}
	r.Recipients = append(r.Recipients, RecipientRef{Type: "admin", ID: "two"}, RecipientRef{Type: "user", ID: "three"})
	r.Tags = append(r.Tags, "two")
	fp := fingerprint(Scope{Application: "app"}, r)
	r.Recipients[0], r.Recipients[2] = r.Recipients[2], r.Recipients[0]
	r.Content.Values[0], r.Content.Values[4] = r.Content.Values[4], r.Content.Values[0]
	r.Tags[0], r.Tags[1] = r.Tags[1], r.Tags[0]
	if fingerprint(Scope{Application: "app"}, r) != fp {
		t.Fatal("canonical order changed fingerprint")
	}
}

// TestConstruction checks invalid catalogs without invoking required collaborators.
func TestConstruction(t *testing.T) {
	p := NewMemoryPlanner(nil)
	a := NewAdapter("chat", "a", "v1", Literal("", "text"), DriverFunc[TextPayload](success))
	route := Route{Name: "route", Revision: "v1", Policy: Any}
	cases := []struct {
		scope    Scope
		routes   []Route
		adapters []Adapter
	}{{Scope{}, []Route{route}, []Adapter{a}}, {Scope{Application: "app", Tenant: " bad"}, []Route{route}, []Adapter{a}}, {Scope{Application: "app"}, nil, []Adapter{a}}, {Scope{Application: "app"}, []Route{route}, nil}, {Scope{Application: "app"}, []Route{route, route}, []Adapter{a}}, {Scope{Application: "app"}, []Route{{Name: "bad", Revision: "v1", Policy: "bad"}}, []Adapter{a}}, {Scope{Application: "app"}, []Route{{Name: "Bad", Revision: "v1", Policy: Any}}, []Adapter{a}}, {Scope{Application: "app"}, []Route{{Name: "route", Policy: Any}}, []Adapter{a}}, {Scope{Application: "app"}, []Route{route}, []Adapter{a, a}}, {Scope{Application: "app"}, []Route{route}, []Adapter{{}}}}
	for i, c := range cases {
		if _, err := New(c.scope, p, c.routes, c.adapters); !errors.Is(err, ErrInvalidRequest) {
			t.Fatalf("case %d", i)
		}
	}
}

// TestPlanValidation rejects every invalid planner boundary before rendering.
func TestPlanValidation(t *testing.T) {
	r := request()
	valid := Plan{Scope: Scope{Application: "test"}, Deliveries: []PlannedDelivery{delivery(r.Recipients[0], "a")}}
	n := setup(t, Any, valid, map[string]DriverFunc[TextPayload]{"a": success})
	tests := []func(*Plan){func(p *Plan) { p.Scope = Scope{} }, func(p *Plan) { p.Deliveries = make([]PlannedDelivery, 401) }, func(p *Plan) { p.Suppressions = make([]Disposition, 2) }, func(p *Plan) { p.Deliveries[0].Recipient.ID = "other" }, func(p *Plan) { p.Deliveries = append(p.Deliveries, p.Deliveries[0]) }, func(p *Plan) { p.Deliveries[0].Endpoint.Identity = "" }, func(p *Plan) { p.Deliveries[0].Endpoint.Address = Address("") }, func(p *Plan) { p.Deliveries[0].Endpoint.Address = Address(strings.Repeat("x", 4097)) }, func(p *Plan) { p.Deliveries[0].Locale = "" }, func(p *Plan) { p.Deliveries[0].Driver = "other" }, func(p *Plan) { p.Deliveries[0].TimeZone = "" }, func(p *Plan) { p.Deliveries[0].TimeZone = "Local" }, func(p *Plan) { p.Deliveries[0].TimeZone = "invalid/zone" }, func(p *Plan) { p.Deliveries[0].TimeZone = strings.Repeat("x", 257) }, func(p *Plan) { p.Suppressions = []Disposition{{Recipient: r.Recipients[0]}} }, func(p *Plan) { p.Deliveries = nil }, func(p *Plan) {
		p.Deliveries = nil
		p.Suppressions = []Disposition{{Recipient: RecipientRef{Type: "user", ID: "other"}}}
	}, func(p *Plan) {
		p.Deliveries = nil
		p.Suppressions = []Disposition{{Recipient: r.Recipients[0], Until: time.Now()}}
	}, func(p *Plan) { p.Deliveries = nil; p.Deferrals = []Disposition{{Recipient: r.Recipients[0]}} }}
	for i, change := range tests {
		p := valid.Clone()
		change(&p)
		if err := n.validatePlan(r, p); !errors.Is(err, ErrInvalidPlan) {
			t.Fatalf("case %d: %v", i, err)
		}
	}
	p := valid.Clone()
	p.Deliveries = nil
	p.Deferrals = []Disposition{{Recipient: r.Recipients[0], Until: time.Now()}}
	cloned := p.Clone()
	cloned.Deferrals[0].Recipient.ID = "changed"
	if p.Deferrals[0].Recipient.ID != "one" {
		t.Fatal("deferral alias")
	}
	p.Deferrals = nil
	p.Suppressions = []Disposition{{Recipient: r.Recipients[0]}}
	cloned = p.Clone()
	cloned.Suppressions[0].Recipient.ID = "changed"
	if p.Suppressions[0].Recipient.ID != "one" {
		t.Fatal("suppression alias")
	}
	n.planner = PlannerFunc(func(context.Context, PlanningRequest) (Plan, error) { return Plan{}, nil })
	if _, err := n.Send(nil, r); !errors.Is(err, ErrInvalidPlan) {
		t.Fatal(err)
	}
}

// TestAdapterFailures verifies renderer and payload rejection are redacted.
func TestAdapterFailures(t *testing.T) {
	r := request()
	n := setup(t, Any, Plan{Deliveries: []PlannedDelivery{delivery(r.Recipients[0], "a")}}, map[string]DriverFunc[TextPayload]{"a": success})
	for _, renderer := range []TextRenderer{func(context.Context, RenderRequest) (TextPayload, error) { return TextPayload{}, errors.New("secret") }, Literal("", "")} {
		n.adapters["chat/a"] = NewAdapter("chat", "a", "v1", renderer, DriverFunc[TextPayload](success))
		if _, err := n.Send(nil, r); !errors.Is(err, ErrRender) {
			t.Fatal(err)
		}
	}
	for _, p := range []TextPayload{{Title: string([]byte{255}), Body: "text"}, {Title: strings.Repeat("x", 257), Body: "text"}, {Body: string([]byte{255})}, {Body: " "}, {Body: strings.Repeat("x", 16385)}} {
		if p.Validate() == nil {
			t.Fatal("invalid text accepted")
		}
	}
	if (TextPayload{Title: strings.Repeat("x", 256), Body: strings.Repeat("x", 16384)}).Validate() != nil {
		t.Fatal("boundary rejected")
	}
}

// TestNormalize preserves typed evidence and bounds server retry advice.
func TestNormalize(t *testing.T) {
	for _, tt := range []struct {
		out    Outcome
		err    error
		status Status
		want   error
	}{{Outcome{Status: Failed}, nil, Failed, ErrPermanent}, {Outcome{Status: "invalid"}, nil, Unknown, ErrOutcomeUnknown}, {Outcome{Status: Unknown}, context.Canceled, Unknown, context.Canceled}, {Outcome{Status: Accepted}, context.Canceled, Accepted, nil}, {Outcome{Status: Failed}, ErrUnauthorized, Failed, ErrUnauthorized}, {Outcome{Status: Failed}, context.DeadlineExceeded, Failed, context.DeadlineExceeded}} {
		out, err := normalize(tt.out, tt.err)
		if out.Status != tt.status || !errors.Is(err, tt.want) {
			t.Fatal(out, err)
		}
	}
	for _, wait := range []time.Duration{-1, 48 * time.Hour} {
		out, _ := normalize(Outcome{Status: Accepted, RetryAfter: wait}, nil)
		if out.RetryAfter < 0 || out.RetryAfter > 24*time.Hour {
			t.Fatal(out)
		}
	}
}

// TestPreflightAndMemoryPlanner verifies no collaborator runs for invalid intent.
func TestPreflightAndMemoryPlanner(t *testing.T) {
	n := setup(t, Any, Plan{}, map[string]DriverFunc[TextPayload]{"a": success})
	r := request()
	r.ID = ""
	if _, err := n.Send(nil, r); !errors.Is(err, ErrInvalidRequest) {
		t.Fatal(err)
	}
	r = request()
	r.Route = "other"
	if _, err := n.Send(nil, r); !errors.Is(err, ErrUnknownRoute) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	p := NewMemoryPlanner(nil)
	if _, err := p.Plan(ctx, PlanningRequest{}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := p.Plan(context.Background(), PlanningRequest{}); !errors.Is(err, ErrInvalidPlan) {
		t.Fatal(err)
	}
}
