package rcmgr

import (
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
)

type Limit interface {
	GetMemoryLimit() int64
	GetStreamLimit(network.Direction) int
	GetConnLimit(network.Direction) int
	GetFDLimit() int
}

type Limiter interface {
	GetSystemLimits() Limit
	GetTransientLimits() Limit
	GetServiceLimits(svc string) Limit
	GetProtocolLimits(proto protocol.ID) Limit
	GetPeerLimits(p peer.ID) Limit
	GetStreamLimits(p peer.ID) Limit
	GetConnLimits() Limit
}

// static limits
type StaticLimit struct {
	Memory          int64
	StreamsInbound  int
	StreamsOutbound int
	ConnsInbound    int
	ConnsOutbound   int
	FD              int
}

var _ Limit = (*StaticLimit)(nil)

// basic limiter with fixed limits
type BasicLimiter struct {
	SystemLimits          Limit
	TransientLimits       Limit
	DefaultServiceLimits  Limit
	ServiceLimits         map[string]Limit
	DefaultProtocolLimits Limit
	ProtocolLimits        map[protocol.ID]Limit
	DefaultPeerLimits     Limit
	PeerLimits            map[peer.ID]Limit
	ConnLimits            Limit
	StreamLimits          Limit
}

var _ Limiter = (*BasicLimiter)(nil)

func (l *StaticLimit) GetMemoryLimit() int64 {
	return l.Memory
}

func (l *StaticLimit) GetStreamLimit(dir network.Direction) int {
	if dir == network.DirInbound {
		return l.StreamsInbound
	} else {
		return l.StreamsOutbound
	}
}

func (l *StaticLimit) GetConnLimit(dir network.Direction) int {
	if dir == network.DirInbound {
		return l.ConnsInbound
	} else {
		return l.ConnsOutbound
	}
}

func (l *StaticLimit) GetFDLimit() int {
	return l.FD
}

func (l *BasicLimiter) GetSystemLimits() Limit {
	return l.SystemLimits
}

func (l *BasicLimiter) GetTransientLimits() Limit {
	return l.TransientLimits
}

func (l *BasicLimiter) GetServiceLimits(svc string) Limit {
	sl, ok := l.ServiceLimits[svc]
	if !ok {
		return l.DefaultServiceLimits
	}
	return sl
}

func (l *BasicLimiter) GetProtocolLimits(proto protocol.ID) Limit {
	pl, ok := l.ProtocolLimits[proto]
	if !ok {
		return l.DefaultProtocolLimits
	}
	return pl
}

func (l *BasicLimiter) GetPeerLimits(p peer.ID) Limit {
	pl, ok := l.PeerLimits[p]
	if !ok {
		return l.DefaultPeerLimits
	}
	return pl
}

func (l *BasicLimiter) GetStreamLimits(p peer.ID) Limit {
	return l.StreamLimits
}

func (l *BasicLimiter) GetConnLimits() Limit {
	return l.ConnLimits
}
