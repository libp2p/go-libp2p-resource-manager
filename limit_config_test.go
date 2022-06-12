package rcmgr

import (
	"os"
	"testing"

	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/stretchr/testify/require"
)

func withMemoryLimit(l BaseLimit, m int64) BaseLimit {
	l2 := l
	l2.Memory = m
	return l2
}

func TestLimitConfigParser(t *testing.T) {
	in, err := os.Open("limit_config_test.json")
	require.NoError(t, err)
	defer in.Close()

	DefaultLimits.AddServiceLimit("C", DefaultLimits.ServiceBaseLimit, BaseLimitIncrease{})
	DefaultLimits.AddProtocolPeerLimit("C", DefaultLimits.ServiceBaseLimit, BaseLimitIncrease{})
	defaults := DefaultLimits.AutoScale()
	cfg, err := readLimiterConfigFromJSON(in, defaults)
	require.NoError(t, err)

	require.Equal(t, int64(65536), cfg.System.Memory)
	require.Equal(t, defaults.System.Streams, cfg.System.Streams)
	require.Equal(t, defaults.System.StreamsInbound, cfg.System.StreamsInbound)
	require.Equal(t, defaults.System.StreamsOutbound, cfg.System.StreamsOutbound)
	require.Equal(t, 16, cfg.System.Conns)
	require.Equal(t, 8, cfg.System.ConnsInbound)
	require.Equal(t, 16, cfg.System.ConnsOutbound)
	require.Equal(t, 16, cfg.System.FD)

	require.Equal(t, defaults.Transient, cfg.Transient)
	require.Equal(t, int64(8765), cfg.ServiceDefault.Memory)

	require.Contains(t, cfg.Service, "A")
	require.Equal(t, withMemoryLimit(cfg.ServiceDefault, 8192), cfg.Service["A"])
	require.Contains(t, cfg.Service, "B")
	require.Equal(t, cfg.ServiceDefault, cfg.Service["B"])
	require.Contains(t, cfg.Service, "C")
	require.Equal(t, defaults.Service["C"], cfg.Service["C"])

	require.Equal(t, int64(4096), cfg.PeerDefault.Memory)
	peerID, err := peer.Decode("12D3KooWPFH2Bx2tPfw6RLxN8k2wh47GRXgkt9yrAHU37zFwHWzS")
	require.NoError(t, err)
	require.Contains(t, cfg.Peer, peerID)
	require.Equal(t, int64(4097), cfg.Peer[peerID].Memory)
}
