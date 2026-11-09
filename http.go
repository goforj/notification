package notification

import (
	"context"
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
	Timeout      time.Duration
	AllowPrivate bool
	Resolver     Resolver
}
