// Deprecated: This package has moved into go-libp2p as a sub-package: github.com/libp2p/go-libp2p/p2p/host/resource-manager/obs.
package obs

import (
	"github.com/libp2p/go-libp2p/p2p/host/resource-manager/obs"
)

var (
	// Deprecated: use github.com/libp2p/go-libp2p/p2p/host/resource-manager/obs.ConnView instead
	ConnView = obs.ConnView
	// Deprecated: use github.com/libp2p/go-libp2p/p2p/host/resource-manager/obs.PeerConnsView instead
	PeerConnsView = obs.PeerConnsView
	// Deprecated: use github.com/libp2p/go-libp2p/p2p/host/resource-manager/obs.PeerConnsNegativeView instead
	PeerConnsNegativeView = obs.PeerConnsNegativeView
	// Deprecated: use github.com/libp2p/go-libp2p/p2p/host/resource-manager/obs.StreamView instead
	StreamView = obs.StreamView
	// Deprecated: use github.com/libp2p/go-libp2p/p2p/host/resource-manager/obs.PeerStreamsView instead
	PeerStreamsView = obs.PeerStreamsView
	// Deprecated: use github.com/libp2p/go-libp2p/p2p/host/resource-manager/obs.PeerStreamNegativeView instead
	PeerStreamNegativeView = obs.PeerStreamNegativeView
	// Deprecated: use github.com/libp2p/go-libp2p/p2p/host/resource-manager/obs.MemoryView instead
	MemoryView = obs.MemoryView
	// Deprecated: use github.com/libp2p/go-libp2p/p2p/host/resource-manager/obs.PeerMemoryView instead
	PeerMemoryView = obs.PeerMemoryView
	// Deprecated: use github.com/libp2p/go-libp2p/p2p/host/resource-manager/obs.PeerMemoryNegativeView instead
	PeerMemoryNegativeView = obs.PeerMemoryNegativeView
	// Not setup yet. Memory isn't attached to a given connection.
	// Deprecated: use github.com/libp2p/go-libp2p/p2p/host/resource-manager/obs.ConnMemoryView instead
	ConnMemoryView = obs.ConnMemoryView
	// Deprecated: use github.com/libp2p/go-libp2p/p2p/host/resource-manager/obs.ConnMemoryNegativeView instead
	ConnMemoryNegativeView = obs.ConnMemoryNegativeView
	// Deprecated: use github.com/libp2p/go-libp2p/p2p/host/resource-manager/obs.FDsView instead
	FDsView = obs.FDsView
	// Deprecated: use github.com/libp2p/go-libp2p/p2p/host/resource-manager/obs.BlockedResourcesView instead
	BlockedResourcesView = obs.BlockedResourcesView
)

// Deprecated: use github.com/libp2p/go-libp2p/p2p/host/resource-manager/obs.DefaultViews instead
var DefaultViews = obs.DefaultViews

// StatsTraceReporter reports stats on the resource manager using its traces.
// Deprecated: use github.com/libp2p/go-libp2p/p2p/host/resource-manager/obs.StatsTraceReporter instead
type StatsTraceReporter = obs.StatsTraceReporter

// Deprecated: use github.com/libp2p/go-libp2p/p2p/host/resource-manager/obs.NewStatsTraceReporter instead
func NewStatsTraceReporter() (StatsTraceReporter, error) {
	return obs.NewStatsTraceReporter()
}
