package rcmgr

import (
	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"

	"github.com/multiformats/go-multiaddr"
)

// Deprecated: use github.com/libp2p/go-libp2p/p2p/host/resource-manager.Allowlist instead
type Allowlist = rcmgr.Allowlist

// WithAllowlistedMultiaddrs sets the multiaddrs to be in the allowlist
// Deprecated: use github.com/libp2p/go-libp2p/p2p/host/resource-manager.WithAllowlistedMultiaddrs instead
func WithAllowlistedMultiaddrs(mas []multiaddr.Multiaddr) Option {
	return rcmgr.WithAllowlistedMultiaddrs(mas)
}
