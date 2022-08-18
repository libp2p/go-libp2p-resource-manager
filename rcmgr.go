// Deprecated: This package has moved into go-libp2p as a sub-package: github.com/libp2p/go-libp2p/p2p/host/resource-manager.
package rcmgr

import (
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
)

// Deprecated: use github.com/libp2p/go-libp2p/p2p/host/resource-manager.Option instead
type Option = rcmgr.Option

// Deprecated: use github.com/libp2p/go-libp2p/p2p/host/resource-manager.NewResourceManager instead
func NewResourceManager(limits Limiter, opts ...Option) (network.ResourceManager, error) {
	return rcmgr.NewResourceManager(limits, opts...)
}

// GetAllowlist tries to get the allowlist from the given resourcemanager
// interface by checking to see if its concrete type is a resourceManager.
// Returns nil if it fails to get the allowlist.
// Deprecated: use github.com/libp2p/go-libp2p/p2p/host/resource-manager.GetAllowlist instead
func GetAllowlist(mgr network.ResourceManager) *Allowlist {
	return rcmgr.GetAllowlist(mgr)
}

// Deprecated: use github.com/libp2p/go-libp2p/p2p/host/resource-manager.IsSystemScope instead
func IsSystemScope(name string) bool {
	return rcmgr.IsSystemScope(name)
}

// Deprecated: use github.com/libp2p/go-libp2p/p2p/host/resource-manager.IsTransientScope instead
func IsTransientScope(name string) bool {
	return rcmgr.IsTransientScope(name)
}

// Deprecated: use github.com/libp2p/go-libp2p/p2p/host/resource-manager.IsConnScope instead
func IsConnScope(name string) bool {
	return rcmgr.IsConnScope(name)
}

// ParsePeerScopeName returns "" if name is not a peerScopeName
// Deprecated: use github.com/libp2p/go-libp2p/p2p/host/resource-manager.ParsePeerScopeName instead
func ParsePeerScopeName(name string) peer.ID {
	return rcmgr.ParsePeerScopeName(name)
}

// ParseServiceScopeName returns the service name if name is a serviceScopeName.
// Otherwise returns ""
// Deprecated: use github.com/libp2p/go-libp2p/p2p/host/resource-manager.ParseServiceScopeName instead
func ParseServiceScopeName(name string) string {
	return rcmgr.ParseServiceScopeName(name)
}

// ParseProtocolScopeName returns the service name if name is a serviceScopeName.
// Otherwise returns ""
// Deprecated: use github.com/libp2p/go-libp2p/p2p/host/resource-manager.ParseProtocolScopeName instead
func ParseProtocolScopeName(name string) string {
	return rcmgr.ParseProtocolScopeName(name)
}
