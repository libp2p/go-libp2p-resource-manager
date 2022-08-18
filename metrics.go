package rcmgr

import (
	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
)

// MetricsReporter is an interface for collecting metrics from resource manager actions
// Deprecated: use github.com/libp2p/go-libp2p/p2p/host/resource-manager.MetricsReporter instead
type MetricsReporter = rcmgr.MetricsReporter

type metrics struct {
	reporter MetricsReporter
}

// WithMetrics is a resource manager option to enable metrics collection
// Deprecated: use github.com/libp2p/go-libp2p/p2p/host/resource-manager.WithMetrics instead
func WithMetrics(reporter MetricsReporter) Option {
	return rcmgr.WithMetrics(reporter)
}
