package rcmgr

import (
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"

	"github.com/pbnjay/memory"
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

// NewDefaultLimiter creates a limiter with default limits and a system memory cap; if the
// system memory cap is 0, then 1/8th of the available memory is used.
func NewDefaultLimiter(memoryCap int64) *BasicLimiter {
	if memoryCap == 0 {
		memoryCap = int64(memory.TotalMemory() / 8)
	}

	system := &StaticLimit{
		Memory:          memoryCap,
		StreamsInbound:  4096,
		StreamsOutbound: 16384,
		ConnsInbound:    256,
		ConnsOutbound:   512,
		FD:              512,
	}
	transient := &StaticLimit{
		Memory:          memoryCap / 16,
		StreamsInbound:  128,
		StreamsOutbound: 512,
		ConnsInbound:    32,
		ConnsOutbound:   128,
		FD:              128,
	}
	svc := &StaticLimit{
		Memory:          memoryCap / 2,
		StreamsInbound:  2048,
		StreamsOutbound: 8192,
	}
	proto := &StaticLimit{
		Memory:          memoryCap / 4,
		StreamsInbound:  1024,
		StreamsOutbound: 4096,
	}
	peer := &StaticLimit{
		Memory:          memoryCap / 16,
		StreamsInbound:  512,
		StreamsOutbound: 2048,
		ConnsInbound:    8,
		ConnsOutbound:   16,
		FD:              8,
	}
	conn := &StaticLimit{
		Memory:        16 << 20,
		ConnsInbound:  1,
		ConnsOutbound: 1,
		FD:            1,
	}
	stream := &StaticLimit{
		Memory:          16 << 20,
		StreamsInbound:  1,
		StreamsOutbound: 1,
	}

	return &BasicLimiter{
		SystemLimits:          system,
		TransientLimits:       transient,
		DefaultServiceLimits:  svc,
		DefaultProtocolLimits: proto,
		DefaultPeerLimits:     peer,
		ConnLimits:            conn,
		StreamLimits:          stream,
	}
}

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
