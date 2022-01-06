package rcmgr

import (
	"github.com/elastic/gosigar"
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
	var mem gosigar.Mem
	if err := mem.Get(); err != nil {
		panic(err)
	}

	limit := int64(float64(mem.ActualFree) * l.MemoryFraction)
	if limit < l.MinMemory {
		limit = l.MinMemory
	} else if limit > l.MaxMemory {
		limit = l.MaxMemory
	}

	return limit
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
		MinMemory:      minMemory / 16,
		MaxMemory:      maxMemory / 16,
		MemoryFraction: memFraction / 16,
		BaseLimit:      DefaultTransientBaseLimit(),
	}
	svc := &DynamicLimit{
		MinMemory:      minMemory / 2,
		MaxMemory:      maxMemory / 2,
		MemoryFraction: memFraction / 2,
		BaseLimit:      DefaultServiceBaseLimit(),
	}
	proto := &DynamicLimit{
		MinMemory:      minMemory / 4,
		MaxMemory:      maxMemory / 4,
		MemoryFraction: memFraction / 4,
		BaseLimit:      DefaultProtocolBaseLimit(),
	}
	peer := &DynamicLimit{
		MinMemory:      minMemory / 16,
		MaxMemory:      maxMemory / 16,
		MemoryFraction: memFraction / 16,
		BaseLimit:      DefaultPeerBaseLimit(),
	}
	conn := &StaticLimit{
		Memory:    16 << 20,
		BaseLimit: ConnBaseLimit(),
	}
	stream := &StaticLimit{
		Memory:    16 << 20,
		BaseLimit: StreamBaseLimit(),
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
