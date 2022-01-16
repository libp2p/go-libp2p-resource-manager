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

	limiter, err := NewLimiterFromJSON(in)
	require.NoError(t, err)

	require.Equal(t,
		&DynamicLimit{
			MinMemory:      16384,
			MaxMemory:      65536,
			MemoryFraction: 0.125,
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
			BaseLimit: DefaultTransientBaseLimit(),
		},
		limiter.TransientLimits)

	require.Equal(t,
		&StaticLimit{
			Memory:    8192,
			BaseLimit: DefaultServiceBaseLimit(),
		},
		limiter.DefaultServiceLimits)

	require.Equal(t,
		&StaticLimit{
			Memory:    2048,
			BaseLimit: DefaultServicePeerBaseLimit(),
		},
		limiter.DefaultServicePeerLimits)

	require.Equal(t, 1, len(limiter.ServiceLimits))
	require.Equal(t,
		&StaticLimit{
			Memory:    8192,
			BaseLimit: DefaultServiceBaseLimit(),
		},
		limiter.ServiceLimits["A"])

	require.Equal(t, 1, len(limiter.ServicePeerLimits))
	require.Equal(t,
		&StaticLimit{
			Memory:    4096,
			BaseLimit: DefaultServicePeerBaseLimit(),
		},
		limiter.ServicePeerLimits["A"])

	require.Equal(t,
		&StaticLimit{
			Memory:    2048,
			BaseLimit: DefaultProtocolBaseLimit(),
		},
		limiter.DefaultProtocolLimits)
	require.Equal(t,
		&StaticLimit{
			Memory:    1024,
			BaseLimit: DefaultProtocolPeerBaseLimit(),
		},
		limiter.DefaultProtocolPeerLimits)

	require.Equal(t, 1, len(limiter.ProtocolLimits))
	require.Equal(t,
		&StaticLimit{
			Memory:    8192,
			BaseLimit: DefaultProtocolBaseLimit(),
		},
		limiter.ProtocolLimits["/A"])

	require.Equal(t, 1, len(limiter.ProtocolPeerLimits))
	require.Equal(t,
		&StaticLimit{
			Memory:    4096,
			BaseLimit: DefaultProtocolPeerBaseLimit(),
		},
		limiter.ProtocolPeerLimits["/A"])

	require.Equal(t,
		&StaticLimit{
			Memory:    4096,
			BaseLimit: DefaultPeerBaseLimit(),
		},
		limiter.DefaultPeerLimits)

	require.Equal(t,
		&StaticLimit{
			Memory:    1 << 20,
			BaseLimit: ConnBaseLimit(),
		},
		limiter.ConnLimits)

	require.Equal(t,
		&StaticLimit{
			Memory:    16 << 20,
			BaseLimit: StreamBaseLimit(),
		},
		limiter.StreamLimits)

}
