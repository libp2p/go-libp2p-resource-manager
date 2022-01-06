package rcmgr

import (
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
)

// Limit is an object that specifies basic resource limits.
type Limit interface {
	GetMemoryLimit() int64
	GetStreamLimit(network.Direction) int
	GetConnLimit(network.Direction) int
	GetFDLimit() int
}

// Limiter is the interface for providing limits to the resource manager.
type Limiter interface {
	GetSystemLimits() Limit
	GetTransientLimits() Limit
	GetServiceLimits(svc string) Limit
	GetProtocolLimits(proto protocol.ID) Limit
	GetPeerLimits(p peer.ID) Limit
	GetStreamLimits(p peer.ID) Limit
	GetConnLimits() Limit
}

// BasicLimiter is a limiter with fixed limits.
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

// BaseLimit is a mixin type for basic resource limits.
type BaseLimit struct {
	StreamsInbound  int
	StreamsOutbound int
	ConnsInbound    int
	ConnsOutbound   int
	FD              int
}

func (l *BaseLimit) GetStreamLimit(dir network.Direction) int {
	if dir == network.DirInbound {
		return l.StreamsInbound
	} else {
		return l.StreamsOutbound
	}
}

func (l *BaseLimit) GetConnLimit(dir network.Direction) int {
	if dir == network.DirInbound {
		return l.ConnsInbound
	} else {
		return l.ConnsOutbound
	}
}

func (l *BaseLimit) GetFDLimit() int {
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

// DefaultSystemBaseLimit returns the default BaseLimit for the System Scope.
func DefaultSystemBaseLimit() BaseLimit {
	return BaseLimit{
		StreamsInbound:  4096,
		StreamsOutbound: 16384,
		ConnsInbound:    256,
		ConnsOutbound:   512,
		FD:              512,
	}
}

// DefaultTransientBaseLimit returns the default BaseLimit for the Transient Scope.
func DefaultTransientBaseLimit() BaseLimit {
	return BaseLimit{
		StreamsInbound:  128,
		StreamsOutbound: 512,
		ConnsInbound:    32,
		ConnsOutbound:   128,
		FD:              128,
	}
}

// DefaultServiceBaseLimit returns the default BaseLimit for Service Scopes.
func DefaultServiceBaseLimit() BaseLimit {
	return BaseLimit{
		StreamsInbound:  2048,
		StreamsOutbound: 8192,
	}
}

// DefaultProtocolBaseLimit returns the default BaseLimit for Protocol Scopes.
func DefaultProtocolBaseLimit() BaseLimit {
	return BaseLimit{
		StreamsInbound:  1024,
		StreamsOutbound: 4096,
	}
}

// DefaultPeerBaseLimit returns the default BaseLimit for Peer Scopes.
func DefaultPeerBaseLimit() BaseLimit {
	return BaseLimit{
		StreamsInbound:  512,
		StreamsOutbound: 2048,
		ConnsInbound:    8,
		ConnsOutbound:   16,
		FD:              8,
	}
}

// ConnBaseLimit returns the BaseLimit for Connection Scopes.
func ConnBaseLimit() BaseLimit {
	return BaseLimit{
		ConnsInbound:  1,
		ConnsOutbound: 1,
		FD:            1,
	}
}

// StreamBaseLimit returns the BaseLimit for Stream Scopes.
func StreamBaseLimit() BaseLimit {
	return BaseLimit{
		StreamsInbound:  1,
		StreamsOutbound: 1,
	}
}
