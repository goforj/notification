package notification

import (
	"context"
	"net/http"
	"net/netip"
	"time"
)

// Resolver supplies every DNS answer for connection-time destination policy.
type Resolver interface {
	LookupNetIP(context.Context, string, string) ([]netip.Addr, error)
}

// HTTPConfig configures bounded provider transport without performing network I/O.
// The default timeout is ten seconds. AllowPrivate is a construction-time choice
// for trusted internal services and must never come from notification content.
type HTTPConfig struct {
	Timeout time.Duration
	// AllowPrivate and Resolver configure only the library-owned default transport.
	AllowPrivate bool
	Resolver     Resolver
	// Transport is an optional caller-owned RoundTripper used unchanged.
	// It owns dialing, DNS, TLS, proxy, and retry policy and must support concurrent
	// calls and request cancellation. Drivers never close or mutate it.
	// A nil Transport selects the library-owned destination-checked transport.
	Transport http.RoundTripper
}
