package rcmgr

import (
	"math"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
)

type baseLimitConfig struct {
	BaseLimit         BaseLimit
	BaseLimitIncrease BaseLimitIncrease
}

// ScalingLimitConfig is a struct for configuring default limits.
// {}BaseLimit is the limits that apply for a minimal node (128 MB of memory for libp2p) and 256 file descriptors.
// {}LimitIncrease is the additional limit granted for every additional 1 GB of RAM.
type ScalingLimitConfig struct {
	SystemBaseLimit     BaseLimit
	SystemLimitIncrease BaseLimitIncrease

	TransientBaseLimit     BaseLimit
	TransientLimitIncrease BaseLimitIncrease

	ServiceBaseLimit     BaseLimit
	ServiceLimitIncrease BaseLimitIncrease
	ServiceLimits        map[string]baseLimitConfig // use AddServiceLimit to modify

	ServicePeerBaseLimit     BaseLimit
	ServicePeerLimitIncrease BaseLimitIncrease
	ServicePeerLimits        map[string]baseLimitConfig // use AddServicePeerLimit to modify

	ProtocolBaseLimit     BaseLimit
	ProtocolLimitIncrease BaseLimitIncrease
	ProtocolLimits        map[protocol.ID]baseLimitConfig // use AddProtocolLimit to modify

	ProtocolPeerBaseLimit     BaseLimit
	ProtocolPeerLimitIncrease BaseLimitIncrease
	ProtocolPeerLimits        map[protocol.ID]baseLimitConfig // use AddProtocolPeerLimit to modify

	PeerBaseLimit     BaseLimit
	PeerLimitIncrease BaseLimitIncrease
	PeerLimits        map[peer.ID]baseLimitConfig // use AddPeerLimit to modify

	ConnBaseLimit     BaseLimit
	ConnLimitIncrease BaseLimitIncrease

	StreamBaseLimit     BaseLimit
	StreamLimitIncrease BaseLimitIncrease
}

func (cfg *ScalingLimitConfig) AddServiceLimit(svc string, base BaseLimit, inc BaseLimitIncrease) {
	if cfg.ServiceLimits == nil {
		cfg.ServiceLimits = make(map[string]baseLimitConfig)
	}
	cfg.ServiceLimits[svc] = baseLimitConfig{
		BaseLimit:         base,
		BaseLimitIncrease: inc,
	}
}

func (cfg *ScalingLimitConfig) AddProtocolLimit(proto protocol.ID, base BaseLimit, inc BaseLimitIncrease) {
	if cfg.ProtocolLimits == nil {
		cfg.ProtocolLimits = make(map[protocol.ID]baseLimitConfig)
	}
	cfg.ProtocolLimits[proto] = baseLimitConfig{
		BaseLimit:         base,
		BaseLimitIncrease: inc,
	}
}

func (cfg *ScalingLimitConfig) AddPeerLimit(p peer.ID, base BaseLimit, inc BaseLimitIncrease) {
	if cfg.PeerLimits == nil {
		cfg.PeerLimits = make(map[peer.ID]baseLimitConfig)
	}
	cfg.PeerLimits[p] = baseLimitConfig{
		BaseLimit:         base,
		BaseLimitIncrease: inc,
	}
}

func (cfg *ScalingLimitConfig) AddServicePeerLimit(svc string, base BaseLimit, inc BaseLimitIncrease) {
	if cfg.ServicePeerLimits == nil {
		cfg.ServicePeerLimits = make(map[string]baseLimitConfig)
	}
	cfg.ServicePeerLimits[svc] = baseLimitConfig{
		BaseLimit:         base,
		BaseLimitIncrease: inc,
	}
}

func (cfg *ScalingLimitConfig) AddProtocolPeerLimit(proto protocol.ID, base BaseLimit, inc BaseLimitIncrease) {
	if cfg.ProtocolPeerLimits == nil {
		cfg.ProtocolPeerLimits = make(map[protocol.ID]baseLimitConfig)
	}
	cfg.ProtocolPeerLimits[proto] = baseLimitConfig{
		BaseLimit:         base,
		BaseLimitIncrease: inc,
	}
}

type LimitConfig struct {
	SystemLimit    BaseLimit
	TransientLimit BaseLimit

	DefaultServiceLimit BaseLimit
	ServiceLimits       map[string]BaseLimit

	DefaultServicePeerLimit BaseLimit
	ServicePeerLimits       map[string]BaseLimit

	DefaultProtocolLimit BaseLimit
	ProtocolLimits       map[protocol.ID]BaseLimit

	DefaultProtocolPeerLimit BaseLimit
	ProtocolPeerLimits       map[protocol.ID]BaseLimit

	DefaultPeerLimit BaseLimit
	PeerLimits       map[peer.ID]BaseLimit

	ConnLimit   BaseLimit
	StreamLimit BaseLimit
}

// Scale scales up a limit configuration.
// memory is the amount of memory that the stack is allowed to consume,
// for a full it's recommended to use 1/8 of the installed system memory.
// If memory is smaller than 128 MB, the base configuration will be used.
//
func (cfg *ScalingLimitConfig) Scale(memory int64, numFD int) LimitConfig {
	var scaleFactor int
	if memory > 128<<20 {
		scaleFactor = int((memory - 128<<20) >> 20)
	}
	lc := LimitConfig{
		SystemLimit:              scale(cfg.SystemBaseLimit, cfg.SystemLimitIncrease, scaleFactor, numFD),
		TransientLimit:           scale(cfg.TransientBaseLimit, cfg.TransientLimitIncrease, scaleFactor, numFD),
		DefaultServiceLimit:      scale(cfg.ServiceBaseLimit, cfg.ServiceLimitIncrease, scaleFactor, numFD),
		DefaultServicePeerLimit:  scale(cfg.ServicePeerBaseLimit, cfg.ServicePeerLimitIncrease, scaleFactor, numFD),
		DefaultProtocolLimit:     scale(cfg.ProtocolBaseLimit, cfg.ProtocolLimitIncrease, scaleFactor, numFD),
		DefaultProtocolPeerLimit: scale(cfg.ProtocolPeerBaseLimit, cfg.ProtocolPeerLimitIncrease, scaleFactor, numFD),
		DefaultPeerLimit:         scale(cfg.PeerBaseLimit, cfg.PeerLimitIncrease, scaleFactor, numFD),
		ConnLimit:                scale(cfg.ConnBaseLimit, cfg.ConnLimitIncrease, scaleFactor, numFD),
		StreamLimit:              scale(cfg.StreamBaseLimit, cfg.ConnLimitIncrease, scaleFactor, numFD),
	}
	if cfg.ServiceLimits != nil {
		lc.ServiceLimits = make(map[string]BaseLimit)
		for svc, l := range cfg.ServiceLimits {
			lc.ServiceLimits[svc] = scale(l.BaseLimit, l.BaseLimitIncrease, scaleFactor, numFD)
		}
	}
	if cfg.ProtocolLimits != nil {
		lc.ProtocolLimits = make(map[protocol.ID]BaseLimit)
		for proto, l := range cfg.ProtocolLimits {
			lc.ProtocolLimits[proto] = scale(l.BaseLimit, l.BaseLimitIncrease, scaleFactor, numFD)
		}
	}
	if cfg.PeerLimits != nil {
		lc.PeerLimits = make(map[peer.ID]BaseLimit)
		for p, l := range cfg.PeerLimits {
			lc.PeerLimits[p] = scale(l.BaseLimit, l.BaseLimitIncrease, scaleFactor, numFD)
		}
	}
	if cfg.ServicePeerLimits != nil {
		lc.ServicePeerLimits = make(map[string]BaseLimit)
		for svc, l := range cfg.ServicePeerLimits {
			lc.ServicePeerLimits[svc] = scale(l.BaseLimit, l.BaseLimitIncrease, scaleFactor, numFD)
		}
	}
	if cfg.ProtocolPeerLimits != nil {
		lc.ProtocolPeerLimits = make(map[protocol.ID]BaseLimit)
		for p, l := range cfg.ProtocolPeerLimits {
			lc.ProtocolPeerLimits[p] = scale(l.BaseLimit, l.BaseLimitIncrease, scaleFactor, numFD)
		}
	}
	return lc
}

// factor is the number of MBs above the minimum (128 MB)
func scale(base BaseLimit, inc BaseLimitIncrease, factor int, numFD int) BaseLimit {
	l := BaseLimit{
		StreamsInbound:  base.StreamsInbound + (inc.StreamsInbound*factor)>>10,
		StreamsOutbound: base.StreamsOutbound + (inc.StreamsOutbound*factor)>>10,
		Streams:         base.Streams + (inc.Streams*factor)>>10,
		ConnsInbound:    base.ConnsInbound + (inc.ConnsInbound*factor)>>10,
		ConnsOutbound:   base.ConnsOutbound + (inc.ConnsOutbound*factor)>>10,
		Conns:           base.Conns + (inc.Conns*factor)>>10,
		Memory:          base.Memory + (inc.Memory*int64(factor))>>10,
		FD:              base.FD,
	}
	if inc.FDFraction > 0 {
		l.FD = int(inc.FDFraction * float64(numFD))
	}
	return l
}

// DefaultLimits are the limits used by the default limiter constructors.
var DefaultLimits = ScalingLimitConfig{
	SystemBaseLimit: BaseLimit{
		ConnsInbound:    64,
		ConnsOutbound:   128,
		Conns:           128,
		StreamsInbound:  64 * 16,
		StreamsOutbound: 128 * 16,
		Streams:         128 * 16,
		Memory:          128 << 20,
		FD:              256,
	},

	SystemLimitIncrease: BaseLimitIncrease{
		ConnsInbound:    64,
		ConnsOutbound:   128,
		Conns:           128,
		StreamsInbound:  64 * 16,
		StreamsOutbound: 128 * 16,
		Streams:         128 * 16,
		Memory:          1 << 30,
		FDFraction:      1,
	},

	TransientBaseLimit: BaseLimit{
		ConnsInbound:    32,
		ConnsOutbound:   64,
		Conns:           64,
		StreamsInbound:  128,
		StreamsOutbound: 256,
		Streams:         256,
		Memory:          32 << 20,
		FD:              64,
	},

	TransientLimitIncrease: BaseLimitIncrease{
		ConnsInbound:    16,
		ConnsOutbound:   32,
		Conns:           32,
		StreamsInbound:  128,
		StreamsOutbound: 256,
		Streams:         256,
		Memory:          128 << 20,
		FDFraction:      0.25,
	},

	ServiceBaseLimit: BaseLimit{
		StreamsInbound:  1024,
		StreamsOutbound: 4096,
		Streams:         4096,
		Memory:          64 << 20,
	},

	ServiceLimitIncrease: BaseLimitIncrease{
		StreamsInbound:  512,
		StreamsOutbound: 2048,
		Streams:         2048,
		Memory:          128 << 20,
	},

	ServicePeerBaseLimit: BaseLimit{
		StreamsInbound:  128,
		StreamsOutbound: 256,
		Streams:         256,
		Memory:          16 << 20,
	},

	ServicePeerLimitIncrease: BaseLimitIncrease{
		StreamsInbound:  4,
		StreamsOutbound: 8,
		Streams:         8,
		Memory:          4 << 20,
	},

	ProtocolBaseLimit: BaseLimit{
		StreamsInbound:  512,
		StreamsOutbound: 2048,
		Streams:         2048,
		Memory:          64 << 20,
	},

	ProtocolLimitIncrease: BaseLimitIncrease{
		StreamsInbound:  256,
		StreamsOutbound: 512,
		Streams:         512,
		Memory:          164 << 20,
	},

	ProtocolPeerBaseLimit: BaseLimit{
		StreamsInbound:  64,
		StreamsOutbound: 128,
		Streams:         256,
		Memory:          16 << 20,
	},

	ProtocolPeerLimitIncrease: BaseLimitIncrease{
		StreamsInbound:  4,
		StreamsOutbound: 8,
		Streams:         16,
		Memory:          4,
	},

	PeerBaseLimit: BaseLimit{
		ConnsInbound:    4,
		ConnsOutbound:   8,
		Conns:           8,
		StreamsInbound:  256,
		StreamsOutbound: 512,
		Streams:         512,
		Memory:          64 << 20,
		FD:              4,
	},

	PeerLimitIncrease: BaseLimitIncrease{
		StreamsInbound:  128,
		StreamsOutbound: 256,
		Streams:         256,
		Memory:          128 << 20,
		FDFraction:      1.0 / 64,
	},

	ConnBaseLimit: BaseLimit{
		ConnsInbound:  1,
		ConnsOutbound: 1,
		Conns:         1,
		FD:            1,
		Memory:        1 << 20,
	},

	StreamBaseLimit: BaseLimit{
		StreamsInbound:  1,
		StreamsOutbound: 1,
		Streams:         1,
		Memory:          16 << 20,
	},
}

var infiniteBaseLimit = BaseLimit{
	Streams:         math.MaxInt,
	StreamsInbound:  math.MaxInt,
	StreamsOutbound: math.MaxInt,
	Conns:           math.MaxInt,
	ConnsInbound:    math.MaxInt,
	ConnsOutbound:   math.MaxInt,
	FD:              math.MaxInt,
	Memory:          math.MaxInt64,
}

// InfiniteLimits are a limiter configuration that uses infinite limits, thus effectively not limiting anything.
// Keep in mind that the operating system limits the number of file descriptors that an application can use.
var InfiniteLimits = LimitConfig{
	SystemLimit:              infiniteBaseLimit,
	TransientLimit:           infiniteBaseLimit,
	DefaultServiceLimit:      infiniteBaseLimit,
	DefaultServicePeerLimit:  infiniteBaseLimit,
	DefaultProtocolLimit:     infiniteBaseLimit,
	DefaultProtocolPeerLimit: infiniteBaseLimit,
	DefaultPeerLimit:         infiniteBaseLimit,
	ConnLimit:                infiniteBaseLimit,
	StreamLimit:              infiniteBaseLimit,
}
