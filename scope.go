package rcmgr

import (
	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
)

// IsSpan will return true if this name was created by newResourceScopeSpan
// Deprecated: use github.com/libp2p/go-libp2p/p2p/host/resource-manager.IsSpan instead
func IsSpan(name string) bool {
	return rcmgr.IsSpan(name)
}
