package notification

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// Scope identifies one application and an optional tenant without trusting context values.
type Scope struct {
	Application string
	Tenant      string
}

// RecipientRef identifies an application-owned recipient, never a raw endpoint.
type RecipientRef struct {
	Type string
	ID   string
}

// Content carries a semantic template key and a closed set of scalar values.
type Content struct {
	Key    string
	Values []Value
}

// Request is bounded application intent, independent of provider payloads.
type Request struct {
	ID         string
	Route      string
	Recipients []RecipientRef
	Content    Content
	Tags       []string
}

// Value stores one named scalar with a canonical representation.
type Value struct {
	name string
	kind string
	text string
}

// String creates a textual template value.
func String(name, value string) Value { return Value{name, "string", value} }

// Integer creates an exact signed integer template value.
func Integer(name string, value int64) Value {
	return Value{name, "integer", strconv.FormatInt(value, 10)}
}

// Boolean creates a Boolean template value.
func Boolean(name string, value bool) Value { return Value{name, "boolean", strconv.FormatBool(value)} }

// Time creates a UTC instant without monotonic-clock state.
func Time(name string, value time.Time) Value {
	return Value{name, "time", value.UTC().Format(time.RFC3339Nano)}
}

// URL creates an HTTP or HTTPS action value validated before planning.
func URL(name, value string) Value { return Value{name, "url", value} }

// Name returns the bounded template variable name.
func (v Value) Name() string { return v.name }

// Kind returns the scalar's stable type identifier.
func (v Value) Kind() string { return v.kind }

// Text returns the scalar representation for a trusted renderer.
func (v Value) Text() string { return v.text }

// String redacts scalar values from ordinary formatting.
func (v Value) String() string { return "[notification value]" }

// GoString redacts scalar values from Go syntax formatting.
func (v Value) GoString() string { return v.String() }

// Clone returns an independent request snapshot.
func (r Request) Clone() Request {
	r.Recipients = append([]RecipientRef(nil), r.Recipients...)
	r.Content.Values = append([]Value(nil), r.Content.Values...)
	r.Tags = append([]string(nil), r.Tags...)
	return r
}

var identifier = regexp.MustCompile(`^[a-z][a-z0-9_.-]{0,127}$`)

// validID rejects controls, ambiguous whitespace, and unbounded identities.
func validID(s string) bool {
	if len(s) == 0 || len(s) > 256 || !utf8.ValidString(s) || strings.TrimSpace(s) != s {
		return false
	}
	for _, r := range s {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

// Validate checks every caller field before any collaborator runs.
func (r Request) Validate() error {
	if !validID(r.ID) || !identifier.MatchString(r.Route) || !identifier.MatchString(r.Content.Key) || len(r.Recipients) == 0 || len(r.Recipients) > 100 || len(r.Content.Values) > 64 || len(r.Tags) > 16 {
		return ErrInvalidRequest
	}
	seen := map[RecipientRef]bool{}
	for _, ref := range r.Recipients {
		if !identifier.MatchString(ref.Type) || !validID(ref.ID) || seen[ref] {
			return ErrInvalidRequest
		}
		seen[ref] = true
	}
	names := map[string]bool{}
	size := 0
	for _, v := range r.Content.Values {
		size += len(v.text)
		if !identifier.MatchString(v.name) || names[v.name] || !utf8.ValidString(v.text) || len(v.text) > 8192 || size > 32768 {
			return ErrInvalidRequest
		}
		names[v.name] = true
		switch v.kind {
		case "string", "integer", "boolean", "time":
		case "url":
			u, err := url.Parse(v.text)
			if err != nil || u.Host == "" || u.User != nil || (u.Scheme != "http" && u.Scheme != "https") {
				return ErrInvalidRequest
			}
		default:
			return ErrInvalidRequest
		}
	}
	tags := map[string]bool{}
	for _, tag := range r.Tags {
		if !identifier.MatchString(tag) || tags[tag] {
			return ErrInvalidRequest
		}
		tags[tag] = true
	}
	return nil
}

// fingerprint separates caller intent from execution-assigned route revisions.
func fingerprint(scope Scope, r Request) string {
	r = r.Clone()
	sort.Slice(r.Recipients, func(i, j int) bool {
		if r.Recipients[i].Type != r.Recipients[j].Type {
			return r.Recipients[i].Type < r.Recipients[j].Type
		}
		return r.Recipients[i].ID < r.Recipients[j].ID
	})
	sort.Strings(r.Tags)
	sort.Slice(r.Content.Values, func(i, j int) bool { return r.Content.Values[i].name < r.Content.Values[j].name })
	values := make([][3]string, len(r.Content.Values))
	for i, v := range r.Content.Values {
		values[i] = [3]string{v.name, v.kind, v.text}
	}
	data, _ := json.Marshal(struct {
		Scope          Scope
		ID, Route, Key string
		Recipients     []RecipientRef
		Values         [][3]string
		Tags           []string
	}{scope, r.ID, r.Route, r.Content.Key, r.Recipients, values, r.Tags})
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// identity hashes an unambiguous tuple without exposing destination data.
func identity(parts ...string) string {
	data, _ := json.Marshal(parts)
	hash := sha256.Sum256(data)
	return "v1-" + hex.EncodeToString(hash[:])
}
