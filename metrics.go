package rcmgr

import (
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
)

// MetricsReporter is an interface for collecting metrics from resource manager actions
type MetricsReporter interface {
	// AllowConn is invoked when opening a connection is allowed
	AllowConn(dir network.Direction, usefd bool)
	// BlockConn is invoked when opening a connection is blocked
	BlockConn(dir network.Direction, usefd bool)

	// AllowStream is invoked when opening a stream is allowed
	AllowStream(p peer.ID, dir network.Direction)
	// BlockStream is invoked when opening a stream is blocked
	BlockStream(p peer.ID, dir network.Direction)

	// AllowPeer is invoked when attaching ac onnection to a peer is allowed
	AllowPeer(p peer.ID)
	// BlockPeer is invoked when attaching ac onnection to a peer is blocked
	BlockPeer(p peer.ID)

	// AllowProtocol is invoked when setting the protocol for a stream is allowed
	AllowProtocol(proto protocol.ID)
	// BlockProtocol is invoked when setting the protocol for a stream is blocked
	BlockProtocol(proto protocol.ID)
	// BlockProtocolPeer is invoked when setting the protocol for a stream is blocked at the per protocol peer scope
	BlockProtocolPeer(proto protocol.ID, p peer.ID)

	// AllowService is invoked when setting the protocol for a stream is allowed
	AllowService(svc string)
	// BlockService is invoked when setting the protocol for a stream is blocked
	BlockService(svc string)
	// BlockServicePeer is invoked when setting the service for a stream is blocked at the per service peer scope
	BlockServicePeer(svc string, p peer.ID)

	// AllowMemory is invoked when a memory reservation is allowed
	AllowMemory(size int)
	// BlockMemory is invoked when a memory reservation is blocked
	BlockMemory(size int)
}

type metrics struct {
	reporter MetricsReporter
}

// WithMetrics is a resource manager option to enable metrics collection. Can be
// called multiple times to add multiple reporters.
func WithMetrics(reporter MetricsReporter) Option {
	return func(r *resourceManager) error {
		if r.metrics == nil {
			r.metrics = &metrics{reporter: reporter}
		} else if multimetrics, ok := r.metrics.reporter.(*MultiMetricsReporter); ok {
			multimetrics.reporters = append(multimetrics.reporters, reporter)
		} else {
			// This was a single reporter. Lets convert it to a multimetrics reporter
			r.metrics = &metrics{
				reporter: &MultiMetricsReporter{reporters: []MetricsReporter{r.metrics.reporter, reporter}},
			}
		}
		return nil
	}
}

func (m *metrics) AllowConn(dir network.Direction, usefd bool) {
	if m == nil {
		return
	}

	m.reporter.AllowConn(dir, usefd)
}

func (m *metrics) BlockConn(dir network.Direction, usefd bool) {
	if m == nil {
		return
	}

	m.reporter.BlockConn(dir, usefd)
}

func (m *metrics) AllowStream(p peer.ID, dir network.Direction) {
	if m == nil {
		return
	}

	m.reporter.AllowStream(p, dir)
}

func (m *metrics) BlockStream(p peer.ID, dir network.Direction) {
	if m == nil {
		return
	}

	m.reporter.BlockStream(p, dir)
}

func (m *metrics) AllowPeer(p peer.ID) {
	if m == nil {
		return
	}

	m.reporter.AllowPeer(p)
}

func (m *metrics) BlockPeer(p peer.ID) {
	if m == nil {
		return
	}

	m.reporter.BlockPeer(p)
}

func (m *metrics) AllowProtocol(proto protocol.ID) {
	if m == nil {
		return
	}

	m.reporter.AllowProtocol(proto)
}

func (m *metrics) BlockProtocol(proto protocol.ID) {
	if m == nil {
		return
	}

	m.reporter.BlockProtocol(proto)
}

func (m *metrics) BlockProtocolPeer(proto protocol.ID, p peer.ID) {
	if m == nil {
		return
	}

	m.reporter.BlockProtocolPeer(proto, p)
}

func (m *metrics) AllowService(svc string) {
	if m == nil {
		return
	}

	m.reporter.AllowService(svc)
}

func (m *metrics) BlockService(svc string) {
	if m == nil {
		return
	}

	m.reporter.BlockService(svc)
}

func (m *metrics) BlockServicePeer(svc string, p peer.ID) {
	if m == nil {
		return
	}

	m.reporter.BlockServicePeer(svc, p)
}

func (m *metrics) AllowMemory(size int) {
	if m == nil {
		return
	}

	m.reporter.AllowMemory(size)
}

func (m *metrics) BlockMemory(size int) {
	if m == nil {
		return
	}

	m.reporter.BlockMemory(size)
}

// MultiMetricsReporter is a helper that allows you to report to multiple metrics reporters.
type MultiMetricsReporter struct {
	reporters []MetricsReporter
}

// AllowConn is invoked when opening a connection is allowed
func (mmr *MultiMetricsReporter) AllowConn(dir network.Direction, usefd bool) {
	for _, r := range mmr.reporters {
		r.AllowConn(dir, usefd)
	}
}

// BlockConn is invoked when opening a connection is blocked
func (mmr *MultiMetricsReporter) BlockConn(dir network.Direction, usefd bool) {
	for _, r := range mmr.reporters {
		r.BlockConn(dir, usefd)
	}
}

// AllowStream is invoked when opening a stream is allowed
func (mmr *MultiMetricsReporter) AllowStream(p peer.ID, dir network.Direction) {
	for _, r := range mmr.reporters {
		r.AllowStream(p, dir)
	}
}

// BlockStream is invoked when opening a stream is blocked
func (mmr *MultiMetricsReporter) BlockStream(p peer.ID, dir network.Direction) {
	for _, r := range mmr.reporters {
		r.BlockStream(p, dir)
	}
}

// AllowPeer is invoked when attaching ac onnection to a peer is allowed
func (mmr *MultiMetricsReporter) AllowPeer(p peer.ID) {
	for _, r := range mmr.reporters {
		r.AllowPeer(p)
	}
}

// BlockPeer is invoked when attaching ac onnection to a peer is blocked
func (mmr *MultiMetricsReporter) BlockPeer(p peer.ID) {
	for _, r := range mmr.reporters {
		r.BlockPeer(p)
	}
}

// AllowProtocol is invoked when setting the protocol for a stream is allowed
func (mmr *MultiMetricsReporter) AllowProtocol(proto protocol.ID) {
	for _, r := range mmr.reporters {
		r.AllowProtocol(proto)
	}
}

// BlockProtocol is invoked when setting the protocol for a stream is blocked
func (mmr *MultiMetricsReporter) BlockProtocol(proto protocol.ID) {
	for _, r := range mmr.reporters {
		r.BlockProtocol(proto)
	}
}

// BlockedProtocolPeer is invoekd when setting the protocol for a stream is blocked at the per protocol peer scope
func (mmr *MultiMetricsReporter) BlockProtocolPeer(proto protocol.ID, p peer.ID) {
	for _, r := range mmr.reporters {
		r.BlockProtocolPeer(proto, p)
	}
}

// AllowPService is invoked when setting the protocol for a stream is allowed
func (mmr *MultiMetricsReporter) AllowService(svc string) {
	for _, r := range mmr.reporters {
		r.AllowService(svc)
	}
}

// BlockPService is invoked when setting the protocol for a stream is blocked
func (mmr *MultiMetricsReporter) BlockService(svc string) {
	for _, r := range mmr.reporters {
		r.BlockService(svc)
	}
}

// BlockedServicePeer is invoked when setting the service for a stream is blocked at the per service peer scope
func (mmr *MultiMetricsReporter) BlockServicePeer(svc string, p peer.ID) {
	for _, r := range mmr.reporters {
		r.BlockServicePeer(svc, p)
	}
}

// AllowMemory is invoked when a memory reservation is allowed
func (mmr *MultiMetricsReporter) AllowMemory(size int) {
	for _, r := range mmr.reporters {
		r.AllowMemory(size)
	}
}

// BlockMemory is invoked when a memory reservation is blocked
func (mmr *MultiMetricsReporter) BlockMemory(size int) {
	for _, r := range mmr.reporters {
		r.BlockMemory(size)
	}
}
