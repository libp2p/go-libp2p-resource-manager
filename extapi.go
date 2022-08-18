package rcmgr

import (
	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
)

// ResourceScopeLimiter is a trait interface that allows you to access scope limits.
// Deprecated: use github.com/libp2p/go-libp2p/p2p/host/resource-manager.ResourceScopeLimiter instead
type ResourceScopeLimiter = rcmgr.ResourceScopeLimiter

// ResourceManagerState is a trait that allows you to access resource manager state.
// Deprecated: use github.com/libp2p/go-libp2p/p2p/host/resource-manager.ResourceManagerState instead
type ResourceManagerState = rcmgr.ResourceManagerState

// Deprecated: use github.com/libp2p/go-libp2p/p2p/host/resource-manager.ResourceManagerStat instead
type ResourceManagerStat = rcmgr.ResourceManagerStat
