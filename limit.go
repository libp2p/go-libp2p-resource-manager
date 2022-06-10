package rcmgr

import (
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
)

// Limit is an object that specifies basic resource limits.
type Limit interface {
	// GetMemoryLimit returns the (current) memory limit.
	GetMemoryLimit() int64
	// GetStreamLimit returns the stream limit, for inbound or outbound streams.
	GetStreamLimit(network.Direction) int
	// GetStreamTotalLimit returns the total stream limit
	GetStreamTotalLimit() int
	// GetConnLimit returns the connection limit, for inbound or outbound connections.
	GetConnLimit(network.Direction) int
	// GetConnTotalLimit returns the total connection limit
	GetConnTotalLimit() int
	// GetFDLimit returns the file descriptor limit.
	GetFDLimit() int
}

// Limiter is the interface for providing limits to the resource manager.
type Limiter interface {
	GetSystemLimits() Limit
	GetTransientLimits() Limit
	GetServiceLimits(svc string) Limit
	GetServicePeerLimits(svc string) Limit
	GetProtocolLimits(proto protocol.ID) Limit
	GetProtocolPeerLimits(proto protocol.ID) Limit
	GetPeerLimits(p peer.ID) Limit
	GetStreamLimits(p peer.ID) Limit
	GetConnLimits() Limit
}

// fixedLimiter is a limiter with fixed limits.
type fixedLimiter struct {
	LimitConfig
}

var _ Limiter = (*fixedLimiter)(nil)

func NewFixedLimiter(conf LimitConfig) Limiter {
	return &fixedLimiter{LimitConfig: conf}
}

// BaseLimit is a mixin type for basic resource limits.
type BaseLimit struct {
	Streams         int
	StreamsInbound  int
	StreamsOutbound int
	Conns           int
	ConnsInbound    int
	ConnsOutbound   int
	FD              int
	Memory          int64
}

// BaseLimitIncrease is the increase per GB of system memory.
type BaseLimitIncrease struct {
	Streams         int
	StreamsInbound  int
	StreamsOutbound int
	Conns           int
	ConnsInbound    int
	ConnsOutbound   int
	Memory          int64
	FDFraction      float64
}

func (l *BaseLimit) GetStreamLimit(dir network.Direction) int {
	if dir == network.DirInbound {
		return l.StreamsInbound
	} else {
		return l.StreamsOutbound
	}
}

func (l *BaseLimit) GetStreamTotalLimit() int {
	return l.Streams
}

func (l *BaseLimit) GetConnLimit(dir network.Direction) int {
	if dir == network.DirInbound {
		return l.ConnsInbound
	} else {
		return l.ConnsOutbound
	}
}

func (l *BaseLimit) GetConnTotalLimit() int {
	return l.Conns
}

func (l *BaseLimit) GetFDLimit() int {
	return l.FD
}

func (l *BaseLimit) GetMemoryLimit() int64 {
	return l.Memory
}

func (l *fixedLimiter) GetSystemLimits() Limit {
	return &l.SystemLimit
}

func (l *fixedLimiter) GetTransientLimits() Limit {
	return &l.TransientLimit
}

func (l *fixedLimiter) GetServiceLimits(svc string) Limit {
	sl, ok := l.ServiceLimits[svc]
	if !ok {
		return &l.DefaultServiceLimit
	}
	return &sl
}

func (l *fixedLimiter) GetServicePeerLimits(svc string) Limit {
	pl, ok := l.ServicePeerLimits[svc]
	if !ok {
		return &l.DefaultServicePeerLimit
	}
	return &pl
}

func (l *fixedLimiter) GetProtocolLimits(proto protocol.ID) Limit {
	pl, ok := l.ProtocolLimits[proto]
	if !ok {
		return &l.DefaultProtocolLimit
	}
	return &pl
}

func (l *fixedLimiter) GetProtocolPeerLimits(proto protocol.ID) Limit {
	pl, ok := l.ProtocolPeerLimits[proto]
	if !ok {
		return &l.DefaultProtocolPeerLimit
	}
	return &pl
}

func (l *fixedLimiter) GetPeerLimits(p peer.ID) Limit {
	pl, ok := l.PeerLimits[p]
	if !ok {
		return &l.DefaultPeerLimit
	}
	return &pl
}

func (l *fixedLimiter) GetStreamLimits(_ peer.ID) Limit {
	return &l.StreamLimit
}

func (l *fixedLimiter) GetConnLimits() Limit {
	return &l.ConnLimit
}
