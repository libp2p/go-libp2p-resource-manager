package rcmgr

import (
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"

	"github.com/elastic/gosigar"
	"github.com/pbnjay/memory"
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

type BaseLimit struct {
	StreamsInbound  int
	StreamsOutbound int
	ConnsInbound    int
	ConnsOutbound   int
	FD              int
}

// StaticLimit is a limit with static values.
type StaticLimit struct {
	BaseLimit
	Memory int64
}

var _ Limit = (*StaticLimit)(nil)

// DynamicLimit is a limit with dynamic memory values, based on available memory
type DynamicLimit struct {
	BaseLimit

	// MinMemory is the minimum memory for this limit
	MinMemory int64
	// MaxMemory is the maximum memory for this limit
	MaxMemory int64
	// MemoryFraction is the fraction of available memory allowed for this limit,
	// bounded by [MinMemory, MaxMemory]
	MemoryFraction int
}

var _ Limit = (*DynamicLimit)(nil)

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

// NewStaticLimiter creates a limiter with default limits and a system memory cap; if the
// system memory cap is 0, then 1/8th of the total memory is used.
func NewStaticLimiter(memoryCap int64) *BasicLimiter {
	if memoryCap == 0 {
		memoryCap = int64(memory.TotalMemory() / 8)
	}

	system := &StaticLimit{
		Memory: memoryCap,
		BaseLimit: BaseLimit{
			StreamsInbound:  4096,
			StreamsOutbound: 16384,
			ConnsInbound:    256,
			ConnsOutbound:   512,
			FD:              512,
		},
	}
	transient := &StaticLimit{
		Memory: memoryCap / 16,
		BaseLimit: BaseLimit{
			StreamsInbound:  128,
			StreamsOutbound: 512,
			ConnsInbound:    32,
			ConnsOutbound:   128,
			FD:              128,
		},
	}
	svc := &StaticLimit{
		Memory: memoryCap / 2,
		BaseLimit: BaseLimit{
			StreamsInbound:  2048,
			StreamsOutbound: 8192,
		},
	}
	proto := &StaticLimit{
		Memory: memoryCap / 4,
		BaseLimit: BaseLimit{
			StreamsInbound:  1024,
			StreamsOutbound: 4096,
		},
	}
	peer := &StaticLimit{
		Memory: memoryCap / 16,
		BaseLimit: BaseLimit{
			StreamsInbound:  512,
			StreamsOutbound: 2048,
			ConnsInbound:    8,
			ConnsOutbound:   16,
			FD:              8,
		},
	}
	conn := &StaticLimit{
		Memory: 16 << 20,
		BaseLimit: BaseLimit{
			ConnsInbound:  1,
			ConnsOutbound: 1,
			FD:            1,
		},
	}
	stream := &StaticLimit{
		Memory: 16 << 20,
		BaseLimit: BaseLimit{
			StreamsInbound:  1,
			StreamsOutbound: 1,
		},
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

// NewDynamicLimiter creates a limiter with default limits and a memory cap dynamically computed
// based on available memory. minMemory and maxMemory specify the system memory bounds,
// while memFraction specifies the fraction of available memory available for the system, within
// the specified bounds.
func NewDynamicLimiter(minMemory, maxMemory int64, memFraction int) *BasicLimiter {
	system := &DynamicLimit{
		MinMemory:      minMemory,
		MaxMemory:      maxMemory,
		MemoryFraction: memFraction,
		BaseLimit: BaseLimit{
			StreamsInbound:  4096,
			StreamsOutbound: 16384,
			ConnsInbound:    256,
			ConnsOutbound:   512,
			FD:              512,
		},
	}
	transient := &DynamicLimit{
		MinMemory:      minMemory / 16,
		MaxMemory:      maxMemory / 16,
		MemoryFraction: memFraction * 16,
		BaseLimit: BaseLimit{
			StreamsInbound:  128,
			StreamsOutbound: 512,
			ConnsInbound:    32,
			ConnsOutbound:   128,
			FD:              128,
		},
	}
	svc := &DynamicLimit{
		MinMemory:      minMemory / 2,
		MaxMemory:      maxMemory / 2,
		MemoryFraction: memFraction * 2,
		BaseLimit: BaseLimit{
			StreamsInbound:  2048,
			StreamsOutbound: 8192,
		},
	}
	proto := &DynamicLimit{
		MinMemory:      minMemory / 4,
		MaxMemory:      maxMemory / 4,
		MemoryFraction: memFraction * 4,
		BaseLimit: BaseLimit{
			StreamsInbound:  1024,
			StreamsOutbound: 4096,
		},
	}
	peer := &DynamicLimit{
		MinMemory:      minMemory / 16,
		MaxMemory:      maxMemory / 16,
		MemoryFraction: memFraction * 16,
		BaseLimit: BaseLimit{
			StreamsInbound:  512,
			StreamsOutbound: 2048,
			ConnsInbound:    8,
			ConnsOutbound:   16,
			FD:              8,
		},
	}
	conn := &StaticLimit{
		Memory: 16 << 20,
		BaseLimit: BaseLimit{
			ConnsInbound:  1,
			ConnsOutbound: 1,
			FD:            1,
		},
	}
	stream := &StaticLimit{
		Memory: 16 << 20,
		BaseLimit: BaseLimit{
			StreamsInbound:  1,
			StreamsOutbound: 1,
		},
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

func (l *DynamicLimit) GetMemoryLimit() int64 {
	var mem gosigar.Mem
	if err := mem.Get(); err != nil {
		panic(err)
	}

	limit := int64(mem.ActualFree) / int64(l.MemoryFraction)
	if limit < l.MinMemory {
		limit = l.MinMemory
	} else if limit > l.MaxMemory {
		limit = l.MaxMemory
	}

	return limit
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
