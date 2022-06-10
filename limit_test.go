package rcmgr

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFileDescriptorCounting(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("can't read file descriptors on Windows")
	}
	n := getNumFDs()
	require.NotZero(t, n)
	require.Less(t, n, int(1e6))
}

func TestScaling(t *testing.T) {
	base := BaseLimit{
		Streams:         100,
		StreamsInbound:  200,
		StreamsOutbound: 400,
		Conns:           10,
		ConnsInbound:    20,
		ConnsOutbound:   40,
		FD:              1,
		Memory:          1 << 20,
	}

	t.Run("no scaling if no increase is defined", func(t *testing.T) {
		cfg := ScalingLimitConfig{ServiceBaseLimit: base}
		scaled := cfg.Scale(8<<30, 100)
		require.Equal(t, base, scaled.DefaultServiceLimit)
	})

	t.Run("scaling", func(t *testing.T) {
		cfg := ScalingLimitConfig{
			TransientBaseLimit: base,
			TransientLimitIncrease: BaseLimitIncrease{
				Streams:         1,
				StreamsInbound:  2,
				StreamsOutbound: 3,
				Conns:           4,
				ConnsInbound:    5,
				ConnsOutbound:   6,
				Memory:          7,
				FDFraction:      0.5,
			},
		}
		scaled := cfg.Scale(128<<20+4<<30, 1000)
		require.Equal(t, 500, scaled.TransientLimit.FD)
		require.Equal(t, base.Streams+4, scaled.TransientLimit.Streams)
		require.Equal(t, base.StreamsInbound+4*2, scaled.TransientLimit.StreamsInbound)
		require.Equal(t, base.StreamsOutbound+4*3, scaled.TransientLimit.StreamsOutbound)
		require.Equal(t, base.Conns+4*4, scaled.TransientLimit.Conns)
		require.Equal(t, base.ConnsInbound+4*5, scaled.TransientLimit.ConnsInbound)
		require.Equal(t, base.ConnsOutbound+4*6, scaled.TransientLimit.ConnsOutbound)
		require.Equal(t, base.Memory+4*7, scaled.TransientLimit.Memory)
	})

	t.Run("scaling limits in maps", func(t *testing.T) {
		cfg := ScalingLimitConfig{
			ServiceLimits: map[string]baseLimitConfig{
				"A": {
					BaseLimit: BaseLimit{Streams: 10, Memory: 100, FD: 9},
				},
				"B": {
					BaseLimit:         BaseLimit{Streams: 20, Memory: 200, FD: 10},
					BaseLimitIncrease: BaseLimitIncrease{Streams: 2, Memory: 3, FDFraction: 0.4},
				},
			},
		}
		scaled := cfg.Scale(128<<20+4<<30, 1000)

		require.Len(t, scaled.ServiceLimits, 2)
		require.Contains(t, scaled.ServiceLimits, "A")
		require.Equal(t, 10, scaled.ServiceLimits["A"].Streams)
		require.Equal(t, int64(100), scaled.ServiceLimits["A"].Memory)
		require.Equal(t, 9, scaled.ServiceLimits["A"].FD)

		require.Contains(t, scaled.ServiceLimits, "B")
		require.Equal(t, 20+4*2, scaled.ServiceLimits["B"].Streams)
		require.Equal(t, int64(200+4*3), scaled.ServiceLimits["B"].Memory)
		require.Equal(t, 400, scaled.ServiceLimits["B"].FD)
	})
}
