package rcmgr

import (
	"testing"

	"github.com/libp2p/go-libp2p-core/network"
)

func checkResources(t *testing.T, rc *Resources, st network.ScopeStat) {
	t.Helper()

	if rc.nconnsIn != st.NumConnsInbound {
		t.Fatalf("expected %d inbound conns, got %d", st.NumConnsInbound, rc.nconnsIn)
	}
	if rc.nconnsOut != st.NumConnsOutbound {
		t.Fatalf("expected %d outbound conns, got %d", st.NumConnsOutbound, rc.nconnsOut)
	}
	if rc.nstreamsIn != st.NumStreamsInbound {
		t.Fatalf("expected %d inbound streams, got %d", st.NumStreamsInbound, rc.nstreamsIn)
	}
	if rc.nstreamsOut != st.NumStreamsOutbound {
		t.Fatalf("expected %d outbound streams, got %d", st.NumStreamsOutbound, rc.nstreamsOut)
	}
	if rc.nfd != st.NumFD {
		t.Fatalf("expected %d file descriptors, got %d", st.NumFD, rc.nfd)
	}
	if rc.memory != st.Memory {
		t.Fatalf("expected %d reserved bytes of memory, got %d", st.Memory, rc.memory)
	}
}

func TestResources(t *testing.T) {
	rc := NewResources(&StaticLimit{
		Memory:          4096,
		StreamsInbound:  1,
		StreamsOutbound: 1,
		ConnsInbound:    1,
		ConnsOutbound:   1,
		FD:              1,
	})

	checkResources(t, rc, network.ScopeStat{})

	if err := rc.checkMemory(1024); err != nil {
		t.Fatal(err)
	}
	if err := rc.checkMemory(4096); err != nil {
		t.Fatal(err)
	}
	if err := rc.checkMemory(8192); err == nil {
		t.Fatal("expected memory check to fail")
	}

	if err := rc.reserveMemory(1024); err != nil {
		t.Fatal(err)
	}
	checkResources(t, rc, network.ScopeStat{Memory: 1024})

	if err := rc.reserveMemory(1024); err != nil {
		t.Fatal(err)
	}
	checkResources(t, rc, network.ScopeStat{Memory: 2048})

	if err := rc.reserveMemory(4096); err == nil {
		t.Fatal("expected memory reservation to fail")
	}
	checkResources(t, rc, network.ScopeStat{Memory: 2048})

	rc.releaseMemory(1024)
	checkResources(t, rc, network.ScopeStat{Memory: 1024})

	if err := rc.reserveMemory(2048); err != nil {
		t.Fatal(err)
	}
	checkResources(t, rc, network.ScopeStat{Memory: 3072})

	buf, key, err := rc.getBuffer(1024)
	if err != nil {
		t.Fatal(err)
	}
	if len(buf) != 1024 {
		t.Fatalf("expected buffer of length %d, got %d", 1024, len(buf))
	}
	if len(rc.buffers) != 1 {
		t.Fatal("expected buffer map to have one buffer")
	}

	checkResources(t, rc, network.ScopeStat{Memory: 4096})

	rc.releaseMemory(3072)
	checkResources(t, rc, network.ScopeStat{Memory: 1024})

	buf[0] = 1
	buf[1] = 2
	buf[2] = 3
	buf, err = rc.growBuffer(key, 2048)
	if err != nil {
		t.Fatal(err)
	}
	if len(buf) != 2048 {
		t.Fatalf("expected buffer of length %d, got %d", 2048, len(buf))
	}
	if buf[0] != 1 || buf[1] != 2 || buf[2] != 3 {
		t.Fatal("buffer was not properly copied")
	}
	if len(rc.buffers) != 1 {
		t.Fatal("expected buffer map to have one buffer")
	}
	checkResources(t, rc, network.ScopeStat{Memory: 2048})

	rc.releaseBuffer(key)
	if len(rc.buffers) != 0 {
		t.Fatal("expected buffer map to be empty")
	}
	checkResources(t, rc, network.ScopeStat{})

	if err := rc.addStream(network.DirInbound); err != nil {
		t.Fatal(err)
	}
	checkResources(t, rc, network.ScopeStat{NumStreamsInbound: 1})

	if err := rc.addStream(network.DirInbound); err == nil {
		t.Fatal("expected addStream to fail")
	}
	checkResources(t, rc, network.ScopeStat{NumStreamsInbound: 1})

	rc.removeStream(network.DirInbound)
	checkResources(t, rc, network.ScopeStat{})

	if err := rc.addStream(network.DirOutbound); err != nil {
		t.Fatal(err)
	}
	checkResources(t, rc, network.ScopeStat{NumStreamsOutbound: 1})

	if err := rc.addStream(network.DirOutbound); err == nil {
		t.Fatal("expected addStream to fail")
	}
	checkResources(t, rc, network.ScopeStat{NumStreamsOutbound: 1})

	rc.removeStream(network.DirOutbound)
	checkResources(t, rc, network.ScopeStat{})

	if err := rc.addConn(network.DirInbound); err != nil {
		t.Fatal(err)
	}
	checkResources(t, rc, network.ScopeStat{NumConnsInbound: 1})

	if err := rc.addConn(network.DirInbound); err == nil {
		t.Fatal("expected addConn to fail")
	}
	checkResources(t, rc, network.ScopeStat{NumConnsInbound: 1})

	rc.removeConn(network.DirInbound)
	checkResources(t, rc, network.ScopeStat{})

	if err := rc.addConn(network.DirOutbound); err != nil {
		t.Fatal(err)
	}
	checkResources(t, rc, network.ScopeStat{NumConnsOutbound: 1})

	if err := rc.addConn(network.DirOutbound); err == nil {
		t.Fatal("expected addConn to fail")
	}
	checkResources(t, rc, network.ScopeStat{NumConnsOutbound: 1})

	rc.removeConn(network.DirOutbound)
	checkResources(t, rc, network.ScopeStat{})

	if err := rc.addFD(1); err != nil {
		t.Fatal(err)
	}
	checkResources(t, rc, network.ScopeStat{NumFD: 1})

	if err := rc.addFD(1); err == nil {
		t.Fatal("expected addFD to fail")
	}
	checkResources(t, rc, network.ScopeStat{NumFD: 1})

	rc.removeFD(1)
	checkResources(t, rc, network.ScopeStat{})
}
