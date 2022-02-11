package rcmgr

import (
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
)

// MetricsReporter is an interface for collecting metrics from resource manager actions
type MetricsReporter interface {
	// BlockOpenConn is invoked when opening a connection is blocked
	BlockOpenConn(dir network.Direction, usefd bool)
	// BlockOpenStream is invoked when opening a stream is blocked
	BlockOpenStream(p peer.ID, dir network.Direction)
	// BlockSetPeer is invoked when attaching ac onnection to a peer is blocked
	BlockSetPeer(p peer.ID)
	// BlockSetProtocol is invoked when setting the protocol for a stream is blocked
	BlockSetProtocol(proto protocol.ID)
	// BlockedSetProtocolPeer is invoekd when setting the protocol for a stream is blocked at the per protocol peer scope
	BlockSetProtocolPeer(proto protocol.ID, p peer.ID)
	// BlockSetPService is invoked when setting the protocol for a stream is blocked
	BlockSetService(svc string)
	// BlockedSetServicePeer is invoekd when setting the service for a stream is blocked at the per service peer scope
	BlockSetServicePeer(svc string, p peer.ID)
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

func (m *metrics) BlockOpenConn(dir network.Direction, usefd bool) {
	if m == nil {
		return
	}

	m.reporter.BlockOpenConn(dir, usefd)
}

func (m *metrics) BlockOpenStream(p peer.ID, dir network.Direction) {
	if m == nil {
		return
	}

	m.reporter.BlockOpenStream(p, dir)
}

func (m *metrics) BlockSetPeer(p peer.ID) {
	if m == nil {
		return
	}

	m.reporter.BlockSetPeer(p)
}

func (m *metrics) BlockSetProtocol(proto protocol.ID) {
	if m == nil {
		return
	}

	m.reporter.BlockSetProtocol(proto)
}

func (m *metrics) BlockSetProtocolPeer(proto protocol.ID, p peer.ID) {
	if m == nil {
		return
	}

	m.reporter.BlockSetProtocolPeer(proto, p)
}

func (m *metrics) BlockSetService(svc string) {
	if m == nil {
		return
	}

	m.reporter.BlockSetService(svc)
}

func (m *metrics) BlockSetServicePeer(svc string, p peer.ID) {
	if m == nil {
		return
	}

	m.reporter.BlockSetServicePeer(svc, p)
}
