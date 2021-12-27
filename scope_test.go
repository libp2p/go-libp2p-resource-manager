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

func TestResourceScopeBasic(t *testing.T) {
	s := NewResourceScope(
		&StaticLimit{
			Memory:          4096,
			StreamsInbound:  1,
			StreamsOutbound: 1,
			ConnsInbound:    1,
			ConnsOutbound:   1,
			FD:              1,
		},
		nil,
	)

	s.IncRef()
	if s.refCnt != 1 {
		t.Fatal("expected refcnt of 1")
	}
	s.DecRef()
	if s.refCnt != 0 {
		t.Fatal("expected refcnt of 0")
	}

	if err := s.ReserveMemory(2048); err != nil {
		t.Fatal(err)
	}
	checkResources(t, s.rc, network.ScopeStat{Memory: 2048})

	if err := s.ReserveMemory(2048); err != nil {
		t.Fatal(err)
	}
	checkResources(t, s.rc, network.ScopeStat{Memory: 4096})

	if err := s.ReserveMemory(1024); err == nil {
		t.Fatal("expected ReserveMemory to fail")
	}
	checkResources(t, s.rc, network.ScopeStat{Memory: 4096})

	s.ReleaseMemory(4096)
	checkResources(t, s.rc, network.ScopeStat{})

	if err := s.AddStream(network.DirInbound); err != nil {
		t.Fatal(err)
	}
	checkResources(t, s.rc, network.ScopeStat{NumStreamsInbound: 1})

	if err := s.AddStream(network.DirInbound); err == nil {
		t.Fatal("expected AddStream to fail")
	}
	checkResources(t, s.rc, network.ScopeStat{NumStreamsInbound: 1})

	s.RemoveStream(network.DirInbound)
	checkResources(t, s.rc, network.ScopeStat{})

	if err := s.AddStream(network.DirOutbound); err != nil {
		t.Fatal(err)
	}
	checkResources(t, s.rc, network.ScopeStat{NumStreamsOutbound: 1})

	if err := s.AddStream(network.DirOutbound); err == nil {
		t.Fatal("expected AddStream to fail")
	}
	checkResources(t, s.rc, network.ScopeStat{NumStreamsOutbound: 1})

	s.RemoveStream(network.DirOutbound)
	checkResources(t, s.rc, network.ScopeStat{})

	if err := s.AddConn(network.DirInbound); err != nil {
		t.Fatal(err)
	}
	checkResources(t, s.rc, network.ScopeStat{NumConnsInbound: 1})

	if err := s.AddConn(network.DirInbound); err == nil {
		t.Fatal("expected AddConn to fail")
	}
	checkResources(t, s.rc, network.ScopeStat{NumConnsInbound: 1})

	s.RemoveConn(network.DirInbound)
	checkResources(t, s.rc, network.ScopeStat{})

	if err := s.AddConn(network.DirOutbound); err != nil {
		t.Fatal(err)
	}
	checkResources(t, s.rc, network.ScopeStat{NumConnsOutbound: 1})

	if err := s.AddConn(network.DirOutbound); err == nil {
		t.Fatal("expected AddConn to fail")
	}
	checkResources(t, s.rc, network.ScopeStat{NumConnsOutbound: 1})

	s.RemoveConn(network.DirOutbound)
	checkResources(t, s.rc, network.ScopeStat{})

	if err := s.AddFD(1); err != nil {
		t.Fatal(err)
	}
	checkResources(t, s.rc, network.ScopeStat{NumFD: 1})

	if err := s.AddFD(1); err == nil {
		t.Fatal("expected AddFD to fail")
	}
	checkResources(t, s.rc, network.ScopeStat{NumFD: 1})

	s.RemoveFD(1)
	checkResources(t, s.rc, network.ScopeStat{})
}

func TestResourceScopeBuffer(t *testing.T) {
	s := NewResourceScope(
		&StaticLimit{
			Memory:          4096,
			StreamsInbound:  1,
			StreamsOutbound: 1,
			ConnsInbound:    1,
			ConnsOutbound:   1,
			FD:              1,
		},
		nil,
	)

	buf, err := s.GetBuffer(2048)
	if err != nil {
		t.Fatal(err)
	}
	if len(buf.Data()) != 2048 {
		t.Fatalf("expected buffer of length %d but got %d", 2048, len(buf.Data()))
	}
	if len(s.rc.buffers) != 1 {
		t.Fatalf("expected %d buffers to be tracked but got %d", 1, len(s.rc.buffers))
	}

	if err = buf.Grow(4096); err != nil {
		t.Fatal(err)
	}
	if len(buf.Data()) != 4096 {
		t.Fatalf("expected buffer of length %d but got %d", 4096, len(buf.Data()))
	}
	if len(s.rc.buffers) != 1 {
		t.Fatalf("expected %d buffers to be tracked but got %d", 1, len(s.rc.buffers))
	}

	if err = buf.Grow(8192); err == nil {
		t.Fatal("expected grow to fail")
	}
	if len(buf.Data()) != 4096 {
		t.Fatalf("expected buffer of length %d but got %d", 4096, len(buf.Data()))
	}
	if len(s.rc.buffers) != 1 {
		t.Fatalf("expected %d buffers to be tracked but got %d", 1, len(s.rc.buffers))
	}

	buf.Release()
	if len(buf.Data()) != 0 {
		t.Fatalf("expected buffer of length %d but got %d", 0, len(buf.Data()))
	}
	if len(s.rc.buffers) != 0 {
		t.Fatalf("expected %d buffers to be tracked but got %d", 0, len(s.rc.buffers))
	}

	buf1, err := s.GetBuffer(2048)
	if err != nil {
		t.Fatal(err)
	}
	buf2, err := s.GetBuffer(2048)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetBuffer(2048); err == nil {
		t.Fatal("expected GetBuffer to fail")
	}
	if len(s.rc.buffers) != 2 {
		t.Fatalf("expected %d buffers to be tracked but got %d", 2, len(s.rc.buffers))
	}

	buf1.Release()
	buf3, err := s.GetBuffer(2048)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.rc.buffers) != 2 {
		t.Fatalf("expected %d buffers to be tracked but got %d", 2, len(s.rc.buffers))
	}

	buf2.Release()
	buf3.Release()
	if len(s.rc.buffers) != 0 {
		t.Fatalf("expected %d buffers to be tracked but got %d", 0, len(s.rc.buffers))
	}
}
