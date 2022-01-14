package rcmgr

import (
	"runtime"

	"github.com/pbnjay/memory"
)

// DynamicLimit is a limit with dynamic memory values, based on available (free) memory
type DynamicLimit struct {
	BaseLimit

	// MinMemory is the minimum memory for this limit
	MinMemory int64
	// MaxMemory is the maximum memory for this limit
	MaxMemory int64
	// MemoryFraction is the fraction of available memory allowed for this limit,
	// bounded by [MinMemory, MaxMemory]
	MemoryFraction float64
}

var _ Limit = (*DynamicLimit)(nil)

func (l *DynamicLimit) GetMemoryLimit() int64 {
	freemem := memory.FreeMemory()

	// account for memory retained by the runtime that is actually free
	// HeapInuse - HeapAlloc is the memory available in allocator spans
	// HeapIdle - HeapReleased is memory held by the runtime that could be returned to the OS
	var memstat runtime.MemStats
	runtime.ReadMemStats(&memstat)

	freemem += (memstat.HeapInuse - memstat.HeapAlloc) + (memstat.HeapIdle - memstat.HeapReleased)

	limit := int64(float64(freemem) * l.MemoryFraction)
	return memoryLimit(limit, l.MinMemory, l.MaxMemory)
}

func (l *DynamicLimit) WithMemoryLimit(memFraction float64, minMemory, maxMemory int64) Limit {
	r := new(DynamicLimit)
	*r = *l

	r.MemoryFraction *= memFraction
	r.MinMemory = minMemory
	r.MaxMemory = maxMemory

	return r
}

func (l *DynamicLimit) WithStreamLimit(numStreamsIn, numStreamsOut int) Limit {
	r := new(DynamicLimit)
	*r = *l

	r.BaseLimit.StreamsInbound = numStreamsIn
	r.BaseLimit.StreamsOutbound = numStreamsOut

	return r
}

func (l *DynamicLimit) WithConnLimit(numConnsIn, numConnsOut int) Limit {
	r := new(DynamicLimit)
	*r = *l

	r.BaseLimit.ConnsInbound = numConnsIn
	r.BaseLimit.ConnsOutbound = numConnsOut

	return r
}

func (l *DynamicLimit) WithFDLimit(numFD int) Limit {
	r := new(DynamicLimit)
	*r = *l

	r.BaseLimit.FD = numFD

	return r
}

// NewDynamicLimiter creates a limiter with default limits and a memory cap dynamically computed
// based on available memory. minMemory and maxMemory specify the system memory bounds,
// while memFraction specifies the fraction of available memory available for the system, within
// the specified bounds.
func NewDynamicLimiter(memFraction float64, minMemory, maxMemory int64) *BasicLimiter {
	system := &DynamicLimit{
		MinMemory:      minMemory,
		MaxMemory:      maxMemory,
		MemoryFraction: memFraction,
		BaseLimit:      DefaultSystemBaseLimit(),
	}
	transient := &DynamicLimit{
		MinMemory:      64 << 20,
		MaxMemory:      128 << 20,
		MemoryFraction: memFraction / 16,
		BaseLimit:      DefaultTransientBaseLimit(),
	}
	svc := &DynamicLimit{
		MinMemory:      64 << 20,
		MaxMemory:      512 << 20,
		MemoryFraction: memFraction / 4,
		BaseLimit:      DefaultServiceBaseLimit(),
	}
	svcPeer := &DynamicLimit{
		MinMemory:      16 << 20,
		MaxMemory:      64 << 20,
		MemoryFraction: memFraction / 16,
		BaseLimit:      DefaultServicePeerBaseLimit(),
	}
	proto := &DynamicLimit{
		MinMemory:      64 << 20,
		MaxMemory:      128 << 20,
		MemoryFraction: memFraction / 16,
		BaseLimit:      DefaultProtocolBaseLimit(),
	}
	protoPeer := &DynamicLimit{
		MinMemory:      16 << 20,
		MaxMemory:      64 << 20,
		MemoryFraction: memFraction / 16,
		BaseLimit:      DefaultProtocolPeerBaseLimit(),
	}

	peer := &DynamicLimit{
		MinMemory:      64 << 20,
		MaxMemory:      128 << 20,
		MemoryFraction: memFraction / 16,
		BaseLimit:      DefaultPeerBaseLimit(),
	}
	conn := &StaticLimit{
		Memory:    1 << 20,
		BaseLimit: ConnBaseLimit(),
	}
	stream := &StaticLimit{
		Memory:    16 << 20,
		BaseLimit: StreamBaseLimit(),
	}

	return &BasicLimiter{
		SystemLimits:              system,
		TransientLimits:           transient,
		DefaultServiceLimits:      svc,
		DefaultServicePeerLimits:  svcPeer,
		DefaultProtocolLimits:     proto,
		DefaultProtocolPeerLimits: protoPeer,
		DefaultPeerLimits:         peer,
		ConnLimits:                conn,
		StreamLimits:              stream,
	}
}
