package rcmgr

import (
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
)

// MetricsReporter is an interface for collecting metrics from resource manager actions
type MetricsReporter interface {
	// BlockConn is invoked when opening a connection is blocked
	BlockConn(dir network.Direction, usefd bool)
	// BlockStream is invoked when opening a stream is blocked
	BlockStream(p peer.ID, dir network.Direction)
	// BlockPeer is invoked when attaching ac onnection to a peer is blocked
	BlockPeer(p peer.ID)
	// BlockProtocol is invoked when setting the protocol for a stream is blocked
	BlockProtocol(proto protocol.ID)
	// BlockedProtocolPeer is invoekd when setting the protocol for a stream is blocked at the per protocol peer scope
	BlockProtocolPeer(proto protocol.ID, p peer.ID)
	// BlockPService is invoked when setting the protocol for a stream is blocked
	BlockService(svc string)
	// BlockedServicePeer is invoekd when setting the service for a stream is blocked at the per service peer scope
	BlockServicePeer(svc string, p peer.ID)
	// BlockMemory is invoked when a memory reservation fails
	BlockMemory(size int)
}

type metrics struct {
	reporter MetricsReporter
}

// WithMetrics is a resource manager option to enable metrics collection
func WithMetrics(reporter MetricsReporter) Option {
	return func(r *resourceManager) error {
		r.metrics = &metrics{reporter: reporter}
		return nil
	}
}

func (m *metrics) BlockConn(dir network.Direction, usefd bool) {
	if m == nil {
		return
	}

	m.reporter.BlockConn(dir, usefd)
}

func (m *metrics) BlockStream(p peer.ID, dir network.Direction) {
	if m == nil {
		return
	}

	m.reporter.BlockStream(p, dir)
}

func (m *metrics) BlockPeer(p peer.ID) {
	if m == nil {
		return
	}

	m.reporter.BlockPeer(p)
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

func (m *metrics) BlockMemory(size int) {
	if m == nil {
		return
	}

	m.reporter.BlockMemory(size)
}
