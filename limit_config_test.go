package rcmgr

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLimitConfigParser(t *testing.T) {
	in, err := os.Open("limit_config_test.json")
	require.NoError(t, err)
	defer in.Close()

	limiter, err := NewDefaultLimiterFromJSON(in)
	require.NoError(t, err)

	require.Equal(t,
		&DynamicLimit{
			MemoryLimit: MemoryLimit{
				MinMemory:      16384,
				MaxMemory:      65536,
				MemoryFraction: 0.125,
			},
			BaseLimit: BaseLimit{
				Streams:         64,
				StreamsInbound:  32,
				StreamsOutbound: 48,
				Conns:           16,
				ConnsInbound:    8,
				ConnsOutbound:   16,
				FD:              16,
			},
		},
		limiter.SystemLimits)

	require.Equal(t,
		&StaticLimit{
			Memory:    4096,
			BaseLimit: DefaultLimits.TransientBaseLimit,
		},
		limiter.TransientLimits)

	require.Equal(t,
		&StaticLimit{
			Memory:    8192,
			BaseLimit: DefaultLimits.ServiceBaseLimit,
		},
		limiter.DefaultServiceLimits)

	require.Equal(t,
		&StaticLimit{
			Memory:    2048,
			BaseLimit: DefaultLimits.ServicePeerBaseLimit,
		},
		limiter.DefaultServicePeerLimits)

	require.Equal(t, 1, len(limiter.ServiceLimits))
	require.Equal(t,
		&StaticLimit{
			Memory:    8192,
			BaseLimit: DefaultLimits.ServiceBaseLimit,
		},
		limiter.ServiceLimits["A"])

	require.Equal(t, 1, len(limiter.ServicePeerLimits))
	require.Equal(t,
		&StaticLimit{
			Memory:    4096,
			BaseLimit: DefaultLimits.ServicePeerBaseLimit,
		},
		limiter.ServicePeerLimits["A"])

	require.Equal(t,
		&StaticLimit{
			Memory:    2048,
			BaseLimit: DefaultLimits.ProtocolBaseLimit,
		},
		limiter.DefaultProtocolLimits)
	require.Equal(t,
		&StaticLimit{
			Memory:    1024,
			BaseLimit: DefaultLimits.ProtocolPeerBaseLimit,
		},
		limiter.DefaultProtocolPeerLimits)

	require.Equal(t, 1, len(limiter.ProtocolLimits))
	require.Equal(t,
		&StaticLimit{
			Memory:    8192,
			BaseLimit: DefaultLimits.ProtocolBaseLimit,
		},
		limiter.ProtocolLimits["/A"])

	require.Equal(t, 1, len(limiter.ProtocolPeerLimits))
	require.Equal(t,
		&StaticLimit{
			Memory:    4096,
			BaseLimit: DefaultLimits.ProtocolPeerBaseLimit,
		},
		limiter.ProtocolPeerLimits["/A"])

	require.Equal(t,
		&StaticLimit{
			Memory:    4096,
			BaseLimit: DefaultLimits.PeerBaseLimit,
		},
		limiter.DefaultPeerLimits)

	require.Equal(t,
		&StaticLimit{
			Memory:    1 << 20,
			BaseLimit: DefaultLimits.ConnBaseLimit,
		},
		limiter.ConnLimits)

	require.Equal(t,
		&StaticLimit{
			Memory:    16 << 20,
			BaseLimit: DefaultLimits.StreamBaseLimit,
		},
		limiter.StreamLimits)

}
