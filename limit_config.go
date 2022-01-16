package rcmgr

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"

	"github.com/pbnjay/memory"
)

type limitConfig struct {
	// if true, then a dynamic limit is used
	Dynamic bool
	// either Memory is set for fixed memory limit
	Memory int64
	// or the following 3 fields for computed memory limits
	MinMemory      int64
	MaxMemory      int64
	MemoryFraction float64

	StreamsInbound  int
	StreamsOutbound int
	Streams         int

	ConnsInbound  int
	ConnsOutbound int
	Conns         int

	FD int
}

func (cfg *limitConfig) toLimit(base BaseLimit, memFraction float64, minMemory, maxMemory int64) (Limit, error) {
	if cfg == nil {
		mem := memoryLimit(int64(float64(memory.TotalMemory())*memFraction), minMemory, maxMemory)
		return &StaticLimit{
			Memory:    mem,
			BaseLimit: base,
		}, nil
	}

	if cfg.Streams > 0 {
		base.Streams = cfg.Streams
	}
	if cfg.StreamsInbound > 0 {
		base.StreamsInbound = cfg.StreamsInbound
	}
	if cfg.StreamsOutbound > 0 {
		base.StreamsOutbound = cfg.StreamsOutbound
	}
	if cfg.Conns > 0 {
		base.Conns = cfg.Conns
	}
	if cfg.ConnsInbound > 0 {
		base.ConnsInbound = cfg.ConnsInbound
	}
	if cfg.ConnsOutbound > 0 {
		base.ConnsOutbound = cfg.ConnsOutbound
	}
	if cfg.FD > 0 {
		base.FD = cfg.FD
	}

	switch {
	case cfg.Memory > 0:
		return &StaticLimit{
			Memory:    cfg.Memory,
			BaseLimit: base,
		}, nil

	case cfg.Dynamic:
		if cfg.MemoryFraction < 0 {
			return nil, fmt.Errorf("negative memory fraction: %f", cfg.MemoryFraction)
		}
		if cfg.MemoryFraction > 0 {
			memFraction = cfg.MemoryFraction
		}
		if cfg.MinMemory > 0 {
			minMemory = cfg.MinMemory
		}
		if cfg.MaxMemory > 0 {
			maxMemory = cfg.MaxMemory
		}

		return &DynamicLimit{
			MinMemory:      minMemory,
			MaxMemory:      maxMemory,
			MemoryFraction: memFraction,
			BaseLimit:      base,
		}, nil

	default:
		if cfg.MemoryFraction < 0 {
			return nil, fmt.Errorf("negative memory fraction: %f", cfg.MemoryFraction)
		}
		if cfg.MemoryFraction > 0 {
			memFraction = cfg.MemoryFraction
		}
		if cfg.MinMemory > 0 {
			minMemory = cfg.MinMemory
		}
		if cfg.MaxMemory > 0 {
			maxMemory = cfg.MaxMemory
		}

		mem := memoryLimit(int64(float64(memory.TotalMemory())*memFraction), minMemory, maxMemory)
		return &StaticLimit{
			Memory:    mem,
			BaseLimit: base,
		}, nil
	}
}

type limiterConfig struct {
	System    *limitConfig
	Transient *limitConfig

	ServiceDefault     *limitConfig
	ServicePeerDefault *limitConfig
	Service            map[string]limitConfig
	ServicePeer        map[string]limitConfig

	ProtocolDefault     *limitConfig
	ProtocolPeerDefault *limitConfig
	Protocol            map[string]limitConfig
	ProtocolPeer        map[string]limitConfig

	PeerDefault *limitConfig
	Peer        map[string]limitConfig

	Conn   *limitConfig
	Stream *limitConfig
}

// NewLimiterFromJSON creates a new limiter by parsing a json configuration.
func NewLimiterFromJSON(in io.Reader) (*BasicLimiter, error) {
	jin := json.NewDecoder(in)

	var cfg limiterConfig

	if err := jin.Decode(&cfg); err != nil {
		return nil, err
	}

	limiter := new(BasicLimiter)
	var err error

	limiter.SystemLimits, err = cfg.System.toLimit(DefaultSystemBaseLimit(), 0.125, 128<<20, 1<<30)
	if err != nil {
		return nil, fmt.Errorf("invalid system limit: %w", err)
	}

	limiter.TransientLimits, err = cfg.Transient.toLimit(DefaultTransientBaseLimit(), 0.0078125, 64<<20, 128<<20)
	if err != nil {
		return nil, fmt.Errorf("invalid transient limit: %w", err)
	}

	limiter.DefaultServiceLimits, err = cfg.ServiceDefault.toLimit(DefaultServiceBaseLimit(), 0.03125, 64<<20, 512<<20)
	if err != nil {
		return nil, fmt.Errorf("invlaid default service limit: %w", err)
	}

	limiter.DefaultServicePeerLimits, err = cfg.ServicePeerDefault.toLimit(DefaultServicePeerBaseLimit(), 0.0078125, 16<<20, 64<<20)
	if err != nil {
		return nil, fmt.Errorf("invlaid default service peer limit: %w", err)
	}

	if len(cfg.Service) > 0 {
		limiter.ServiceLimits = make(map[string]Limit, len(cfg.Service))
		for svc, cfgLimit := range cfg.Service {
			limiter.ServiceLimits[svc], err = cfgLimit.toLimit(DefaultServiceBaseLimit(), 0.03125, 64<<20, 512<<20)
			if err != nil {
				return nil, fmt.Errorf("invalid service limit for %s: %w", svc, err)
			}
		}
	}

	if len(cfg.ServicePeer) > 0 {
		limiter.ServicePeerLimits = make(map[string]Limit, len(cfg.ServicePeer))
		for svc, cfgLimit := range cfg.ServicePeer {
			limiter.ServicePeerLimits[svc], err = cfgLimit.toLimit(DefaultServicePeerBaseLimit(), 0.0078125, 16<<20, 64<<20)
			if err != nil {
				return nil, fmt.Errorf("invalid service peer limit for %s: %w", svc, err)
			}
		}
	}

	limiter.DefaultProtocolLimits, err = cfg.ProtocolDefault.toLimit(DefaultProtocolBaseLimit(), 0.0078125, 64<<20, 128<<20)
	if err != nil {
		return nil, fmt.Errorf("invlaid default protocol limit: %w", err)
	}

	limiter.DefaultProtocolPeerLimits, err = cfg.ProtocolPeerDefault.toLimit(DefaultProtocolPeerBaseLimit(), 0.0078125, 16<<20, 64<<20)
	if err != nil {
		return nil, fmt.Errorf("invlaid default protocol peer limit: %w", err)
	}

	if len(cfg.Protocol) > 0 {
		limiter.ProtocolLimits = make(map[protocol.ID]Limit, len(cfg.Protocol))
		for p, cfgLimit := range cfg.Protocol {
			limiter.ProtocolLimits[protocol.ID(p)], err = cfgLimit.toLimit(DefaultProtocolBaseLimit(), 0.0078125, 64<<20, 128<<20)
			if err != nil {
				return nil, fmt.Errorf("invalid service limit for %s: %w", p, err)
			}
		}
	}

	if len(cfg.ProtocolPeer) > 0 {
		limiter.ProtocolPeerLimits = make(map[protocol.ID]Limit, len(cfg.ProtocolPeer))
		for p, cfgLimit := range cfg.ProtocolPeer {
			limiter.ProtocolPeerLimits[protocol.ID(p)], err = cfgLimit.toLimit(DefaultProtocolPeerBaseLimit(), 0.0078125, 16<<20, 64<<20)
			if err != nil {
				return nil, fmt.Errorf("invalid service peer limit for %s: %w", p, err)
			}
		}
	}

	limiter.DefaultPeerLimits, err = cfg.PeerDefault.toLimit(DefaultPeerBaseLimit(), 0.0078125, 64<<20, 1288<<20)
	if err != nil {
		return nil, fmt.Errorf("invalid peer limit: %w", err)
	}

	if len(cfg.Peer) > 0 {
		limiter.PeerLimits = make(map[peer.ID]Limit, len(cfg.Peer))
		for p, cfgLimit := range cfg.Peer {
			pid, err := peer.IDFromString(p)
			if err != nil {
				return nil, fmt.Errorf("invalid peer ID %s: %w", p, err)
			}
			limiter.PeerLimits[pid], err = cfgLimit.toLimit(DefaultPeerBaseLimit(), 0.0078125, 64<<20, 1288<<20)
			if err != nil {
				return nil, fmt.Errorf("invalid peer limit for %s: %w", p, err)
			}
		}
	}

	limiter.ConnLimits, err = cfg.Conn.toLimit(ConnBaseLimit(), 1, 1<<20, 1<<20)
	if err != nil {
		return nil, fmt.Errorf("invalid conn limit: %w", err)
	}

	limiter.StreamLimits, err = cfg.Stream.toLimit(StreamBaseLimit(), 1, 16<<20, 16<<20)
	if err != nil {
		return nil, fmt.Errorf("invalid stream limit: %w", err)
	}

	return limiter, nil
}
