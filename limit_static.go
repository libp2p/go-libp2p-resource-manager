package rcmgr

import (
	"github.com/pbnjay/memory"
)

// StaticLimit is a limit with static values.
type StaticLimit struct {
	BaseLimit
	Memory int64
}

var _ Limit = (*StaticLimit)(nil)

func (l *StaticLimit) GetMemoryLimit() int64 {
	return l.Memory
}

func (l *StaticLimit) WithMemoryLimit(memFraction float64, minMemory, maxMemory int64) Limit {
	r := new(StaticLimit)
	*r = *l

	r.Memory = int64(memFraction * float64(r.Memory))
	if r.Memory < minMemory {
		r.Memory = minMemory
	} else if r.Memory > maxMemory {
		r.Memory = maxMemory
	}

	return r
}

func (l *StaticLimit) WithStreamLimit(numStreamsIn, numStreamsOut int) Limit {
	r := new(StaticLimit)
	*r = *l

	r.BaseLimit.StreamsInbound = numStreamsIn
	r.BaseLimit.StreamsOutbound = numStreamsOut

	return r
}

func (l *StaticLimit) WithConnLimit(numConnsIn, numConnsOut int) Limit {
	r := new(StaticLimit)
	*r = *l

	r.BaseLimit.ConnsInbound = numConnsIn
	r.BaseLimit.ConnsOutbound = numConnsOut

	return r
}

func (l *StaticLimit) WithFDLimit(numFD int) Limit {
	r := new(StaticLimit)
	*r = *l

	r.BaseLimit.FD = numFD

	return r
}

// NewStaticLimiter creates a limiter with default base limits and a system memory cap specified as
// a fraction of total system memory. The assigned memory will not be less than minMemory or more
// than maxMemory.
func NewStaticLimiter(memFraction float64, minMemory, maxMemory int64) *BasicLimiter {
	memoryCap := int64(float64(memory.TotalMemory()) * memFraction)
	switch {
	case memoryCap < minMemory:
		memoryCap = minMemory
	case memoryCap > maxMemory:
		memoryCap = maxMemory
	}
	return newDefaultStaticLimiter(memoryCap)
}

// NewFixedLimiter creates a limiter with default base limits and a specified system memory cap.
func NewFixedLimiter(memoryCap int64) *BasicLimiter {
	return newDefaultStaticLimiter(memoryCap)
}

func newDefaultStaticLimiter(memoryCap int64) *BasicLimiter {
	system := &StaticLimit{
		Memory:    memoryCap,
		BaseLimit: DefaultSystemBaseLimit(),
	}
	transient := &StaticLimit{
		Memory:    memoryCap / 4,
		BaseLimit: DefaultTransientBaseLimit(),
	}
	svc := &StaticLimit{
		Memory:    memoryCap / 4,
		BaseLimit: DefaultServiceBaseLimit(),
	}
	proto := &StaticLimit{
		Memory:    memoryCap / 4,
		BaseLimit: DefaultProtocolBaseLimit(),
	}
	peer := &StaticLimit{
		Memory:    memoryCap / 4,
		BaseLimit: DefaultPeerBaseLimit(),
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
