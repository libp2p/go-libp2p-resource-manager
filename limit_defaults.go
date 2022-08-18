package rcmgr

import (
	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
)

// ScalingLimitConfig is a struct for configuring default limits.
// {}BaseLimit is the limits that Apply for a minimal node (128 MB of memory for libp2p) and 256 file descriptors.
// {}LimitIncrease is the additional limit granted for every additional 1 GB of RAM.
// Deprecated: use github.com/libp2p/go-libp2p/p2p/host/resource-manager.ScalingLimitConfig instead
type ScalingLimitConfig = rcmgr.ScalingLimitConfig

// Deprecated: use github.com/libp2p/go-libp2p/p2p/host/resource-manager.LimitConfig instead
type LimitConfig = rcmgr.LimitConfig

// DefaultLimits are the limits used by the default limiter constructors.
// Deprecated: use github.com/libp2p/go-libp2p/p2p/host/resource-manager.DefaultLimits instead
var DefaultLimits = rcmgr.DefaultLimits

// InfiniteLimits are a limiter configuration that uses infinite limits, thus effectively not limiting anything.
// Keep in mind that the operating system limits the number of file descriptors that an application can use.
// Deprecated: use github.com/libp2p/go-libp2p/p2p/host/resource-manager.InfiniteLimits instead
var InfiniteLimits = rcmgr.InfiniteLimits
