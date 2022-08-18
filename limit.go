/*
Package rcmgr is the resource manager for go-libp2p. This allows you to track
resources being used throughout your go-libp2p process. As well as making sure
that the process doesn't use more resources than what you define as your
limits. The resource manager only knows about things it is told about, so it's
the responsibility of the user of this library (either go-libp2p or a go-libp2p
user) to make sure they check with the resource manager before actually
allocating the resource.
*/
package rcmgr

import (
	"io"

	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
)

// Limit is an object that specifies basic resource limits.
// Deprecated: use github.com/libp2p/go-libp2p/p2p/host/resource-manager.Limit instead
type Limit = rcmgr.Limit

// Limiter is the interface for providing limits to the resource manager.
// Deprecated: use github.com/libp2p/go-libp2p/p2p/host/resource-manager.Limiter instead
type Limiter = rcmgr.Limiter

// NewDefaultLimiterFromJSON creates a new limiter by parsing a json configuration,
// using the default limits for fallback.
// Deprecated: use github.com/libp2p/go-libp2p/p2p/host/resource-manager.NewDefaultLimiterFromJSON instead
func NewDefaultLimiterFromJSON(in io.Reader) (Limiter, error) {
	return rcmgr.NewDefaultLimiterFromJSON(in)
}

// NewLimiterFromJSON creates a new limiter by parsing a json configuration.
// Deprecated: use github.com/libp2p/go-libp2p/p2p/host/resource-manager.NewLimiterFromJSON instead
func NewLimiterFromJSON(in io.Reader, defaults LimitConfig) (Limiter, error) {
	return rcmgr.NewLimiterFromJSON(in, defaults)
}

// Deprecated: use github.com/libp2p/go-libp2p/p2p/host/resource-manager.NewFixedLimiter instead
func NewFixedLimiter(conf LimitConfig) Limiter {
	return rcmgr.NewFixedLimiter(conf)
}

// BaseLimit is a mixin type for basic resource limits.
// Deprecated: use github.com/libp2p/go-libp2p/p2p/host/resource-manager.BaseLimit instead
type BaseLimit = rcmgr.BaseLimit

// BaseLimitIncrease is the increase per GB of system memory.
// Deprecated: use github.com/libp2p/go-libp2p/p2p/host/resource-manager.BaseLimitIncrease instead
type BaseLimitIncrease = rcmgr.BaseLimitIncrease
