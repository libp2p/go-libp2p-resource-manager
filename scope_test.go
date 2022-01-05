package rcmgr

import (
	"testing"

	"github.com/libp2p/go-libp2p-core/network"
)

func checkResources(t *testing.T, rc *resources, st network.ScopeStat) {
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

func checkStatus(t *testing.T, expected, status network.MemoryStatus) {
	t.Helper()
	if status != expected {
		t.Fatalf("expected memory status %d but got %d", expected, status)
	}
}

func TestResources(t *testing.T) {
	rc := resources{limit: &StaticLimit{
		Memory:          4096,
		StreamsInbound:  1,
		StreamsOutbound: 1,
		ConnsInbound:    1,
		ConnsOutbound:   1,
		FD:              1,
	}}

	checkResources(t, &rc, network.ScopeStat{})

	var status network.MemoryStatus
	var err error

	if status, err = rc.checkMemory(1024); err != nil {
		t.Fatal(err)
	}
	checkStatus(t, network.MemoryStatusOK, status)

	if status, err = rc.checkMemory(2048); err != nil {
		t.Fatal(err)
	}
	checkStatus(t, network.MemoryStatusOK, status)

	if status, err = rc.checkMemory(3072); err != nil {
		t.Fatal(err)
	}
	checkStatus(t, network.MemoryStatusCaution, status)

	if status, err = rc.checkMemory(4096); err != nil {
		t.Fatal(err)
	}
	checkStatus(t, network.MemoryStatusCritical, status)

	if _, err := rc.checkMemory(8192); err == nil {
		t.Fatal("expected memory check to fail")
	}

	if status, err = rc.reserveMemory(1024); err != nil {
		t.Fatal(err)
	}
	checkResources(t, &rc, network.ScopeStat{Memory: 1024})
	checkStatus(t, network.MemoryStatusOK, status)

	if status, err = rc.reserveMemory(1024); err != nil {
		t.Fatal(err)
	}
	checkResources(t, &rc, network.ScopeStat{Memory: 2048})
	checkStatus(t, network.MemoryStatusOK, status)

	if status, err = rc.reserveMemory(1024); err != nil {
		t.Fatal(err)
	}
	checkResources(t, &rc, network.ScopeStat{Memory: 3072})
	checkStatus(t, network.MemoryStatusCaution, status)

	if status, err = rc.reserveMemory(512); err != nil {
		t.Fatal(err)
	}
	checkResources(t, &rc, network.ScopeStat{Memory: 3584})
	checkStatus(t, network.MemoryStatusCritical, status)

	if _, err := rc.reserveMemory(4096); err == nil {
		t.Fatal("expected memory reservation to fail")
	}
	checkResources(t, &rc, network.ScopeStat{Memory: 3584})

	rc.releaseMemory(2560)
	checkResources(t, &rc, network.ScopeStat{Memory: 1024})

	if status, err = rc.reserveMemory(2048); err != nil {
		t.Fatal(err)
	}
	checkResources(t, &rc, network.ScopeStat{Memory: 3072})
	checkStatus(t, network.MemoryStatusCaution, status)

	rc.releaseMemory(3072)
	checkResources(t, &rc, network.ScopeStat{})

	if err := rc.addStream(network.DirInbound); err != nil {
		t.Fatal(err)
	}
	checkResources(t, &rc, network.ScopeStat{NumStreamsInbound: 1})

	if err := rc.addStream(network.DirInbound); err == nil {
		t.Fatal("expected addStream to fail")
	}
	checkResources(t, &rc, network.ScopeStat{NumStreamsInbound: 1})

	rc.removeStream(network.DirInbound)
	checkResources(t, &rc, network.ScopeStat{})

	if err := rc.addStream(network.DirOutbound); err != nil {
		t.Fatal(err)
	}
	checkResources(t, &rc, network.ScopeStat{NumStreamsOutbound: 1})

	if err := rc.addStream(network.DirOutbound); err == nil {
		t.Fatal("expected addStream to fail")
	}
	checkResources(t, &rc, network.ScopeStat{NumStreamsOutbound: 1})

	rc.removeStream(network.DirOutbound)
	checkResources(t, &rc, network.ScopeStat{})

	if err := rc.addConn(network.DirInbound, false); err != nil {
		t.Fatal(err)
	}
	checkResources(t, &rc, network.ScopeStat{NumConnsInbound: 1})

	if err := rc.addConn(network.DirInbound, false); err == nil {
		t.Fatal("expected addConn to fail")
	}
	checkResources(t, &rc, network.ScopeStat{NumConnsInbound: 1})

	rc.removeConn(network.DirInbound, false)
	checkResources(t, &rc, network.ScopeStat{})

	if err := rc.addConn(network.DirOutbound, false); err != nil {
		t.Fatal(err)
	}
	checkResources(t, &rc, network.ScopeStat{NumConnsOutbound: 1})

	if err := rc.addConn(network.DirOutbound, false); err == nil {
		t.Fatal("expected addConn to fail")
	}
	checkResources(t, &rc, network.ScopeStat{NumConnsOutbound: 1})

	rc.removeConn(network.DirOutbound, false)
	checkResources(t, &rc, network.ScopeStat{})

	if err := rc.addConn(network.DirInbound, true); err != nil {
		t.Fatal(err)
	}
	checkResources(t, &rc, network.ScopeStat{NumConnsInbound: 1, NumFD: 1})

	if err := rc.addConn(network.DirOutbound, true); err == nil {
		t.Fatal("expected addConn to fail")
	}
	checkResources(t, &rc, network.ScopeStat{NumConnsInbound: 1, NumFD: 1})

	rc.removeConn(network.DirInbound, true)
	checkResources(t, &rc, network.ScopeStat{})
}

func TestResourceScopeSimple(t *testing.T) {
	s := newResourceScope(
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

	testResourceScopeBasic(t, s)
}

func testResourceScopeBasic(t *testing.T, s *resourceScope) {
	var status network.MemoryStatus
	var err error

	if status, err = s.ReserveMemory(2048); err != nil {
		t.Fatal(err)
	}
	checkResources(t, &s.rc, network.ScopeStat{Memory: 2048})
	checkStatus(t, network.MemoryStatusOK, status)

	if status, err = s.ReserveMemory(1024); err != nil {
		t.Fatal(err)
	}
	checkResources(t, &s.rc, network.ScopeStat{Memory: 3072})
	checkStatus(t, network.MemoryStatusCaution, status)

	if status, err = s.ReserveMemory(512); err != nil {
		t.Fatal(err)
	}
	checkResources(t, &s.rc, network.ScopeStat{Memory: 3584})
	checkStatus(t, network.MemoryStatusCritical, status)

	if status, err = s.ReserveMemory(512); err != nil {
		t.Fatal(err)
	}
	checkResources(t, &s.rc, network.ScopeStat{Memory: 4096})
	checkStatus(t, network.MemoryStatusCritical, status)

	if _, err := s.ReserveMemory(1024); err == nil {
		t.Fatal("expected ReserveMemory to fail")
	}
	checkResources(t, &s.rc, network.ScopeStat{Memory: 4096})

	s.ReleaseMemory(4096)
	checkResources(t, &s.rc, network.ScopeStat{})

	if err := s.AddStream(network.DirInbound); err != nil {
		t.Fatal(err)
	}
	checkResources(t, &s.rc, network.ScopeStat{NumStreamsInbound: 1})

	if err := s.AddStream(network.DirInbound); err == nil {
		t.Fatal("expected AddStream to fail")
	}
	checkResources(t, &s.rc, network.ScopeStat{NumStreamsInbound: 1})

	s.RemoveStream(network.DirInbound)
	checkResources(t, &s.rc, network.ScopeStat{})

	if err := s.AddStream(network.DirOutbound); err != nil {
		t.Fatal(err)
	}
	checkResources(t, &s.rc, network.ScopeStat{NumStreamsOutbound: 1})

	if err := s.AddStream(network.DirOutbound); err == nil {
		t.Fatal("expected AddStream to fail")
	}
	checkResources(t, &s.rc, network.ScopeStat{NumStreamsOutbound: 1})

	s.RemoveStream(network.DirOutbound)
	checkResources(t, &s.rc, network.ScopeStat{})

	if err := s.AddConn(network.DirInbound, false); err != nil {
		t.Fatal(err)
	}
	checkResources(t, &s.rc, network.ScopeStat{NumConnsInbound: 1})

	if err := s.AddConn(network.DirInbound, false); err == nil {
		t.Fatal("expected AddConn to fail")
	}
	checkResources(t, &s.rc, network.ScopeStat{NumConnsInbound: 1})

	s.RemoveConn(network.DirInbound, false)
	checkResources(t, &s.rc, network.ScopeStat{})

	if err := s.AddConn(network.DirOutbound, false); err != nil {
		t.Fatal(err)
	}
	checkResources(t, &s.rc, network.ScopeStat{NumConnsOutbound: 1})

	if err := s.AddConn(network.DirOutbound, false); err == nil {
		t.Fatal("expected AddConn to fail")
	}
	checkResources(t, &s.rc, network.ScopeStat{NumConnsOutbound: 1})

	s.RemoveConn(network.DirOutbound, false)
	checkResources(t, &s.rc, network.ScopeStat{})

	if err := s.AddConn(network.DirInbound, true); err != nil {
		t.Fatal(err)
	}
	checkResources(t, &s.rc, network.ScopeStat{NumConnsInbound: 1, NumFD: 1})

	if err := s.AddConn(network.DirOutbound, true); err == nil {
		t.Fatal("expected AddConn to fail")
	}
	checkResources(t, &s.rc, network.ScopeStat{NumConnsInbound: 1, NumFD: 1})

	s.RemoveConn(network.DirInbound, true)
	checkResources(t, &s.rc, network.ScopeStat{})
}

func TestResourceScopeTxnBasic(t *testing.T) {
	s := newResourceScope(
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

	txn, err := s.BeginTransaction()
	if err != nil {
		t.Fatal(err)
	}

	testResourceScopeBasic(t, txn.(*resourceScope))
	checkResources(t, &s.rc, network.ScopeStat{})

	// check constraint propagation
	if _, err := txn.ReserveMemory(4096); err != nil {
		t.Fatal(err)
	}
	checkResources(t, &txn.(*resourceScope).rc, network.ScopeStat{Memory: 4096})
	checkResources(t, &s.rc, network.ScopeStat{Memory: 4096})
	txn.Done()
	checkResources(t, &s.rc, network.ScopeStat{})
	txn.Done() // idempotent
	checkResources(t, &s.rc, network.ScopeStat{})
}

func TestResourceScopeTxnZombie(t *testing.T) {
	s := newResourceScope(
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

	txn1, err := s.BeginTransaction()
	if err != nil {
		t.Fatal(err)
	}

	txn2, err := txn1.BeginTransaction()
	if err != nil {
		t.Fatal(err)
	}

	if _, err := txn2.ReserveMemory(4096); err != nil {
		t.Fatal(err)
	}
	checkResources(t, &txn2.(*resourceScope).rc, network.ScopeStat{Memory: 4096})
	checkResources(t, &txn1.(*resourceScope).rc, network.ScopeStat{Memory: 4096})
	checkResources(t, &s.rc, network.ScopeStat{Memory: 4096})

	txn1.Done()
	checkResources(t, &s.rc, network.ScopeStat{})
	if _, err := txn2.ReserveMemory(4096); err == nil {
		t.Fatal("expected ReserveMemory to fail")
	}

	txn2.Done()
	checkResources(t, &s.rc, network.ScopeStat{})
}

func TestResourceScopeTxnTree(t *testing.T) {
	s := newResourceScope(
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

	txn1, err := s.BeginTransaction()
	if err != nil {
		t.Fatal(err)
	}

	txn2, err := txn1.BeginTransaction()
	if err != nil {
		t.Fatal(err)
	}

	txn3, err := txn1.BeginTransaction()
	if err != nil {
		t.Fatal(err)
	}

	txn4, err := txn2.BeginTransaction()
	if err != nil {
		t.Fatal(err)
	}

	txn5, err := txn2.BeginTransaction()
	if err != nil {
		t.Fatal(err)
	}

	if _, err := txn3.ReserveMemory(1024); err != nil {
		t.Fatal(err)
	}
	checkResources(t, &txn3.(*resourceScope).rc, network.ScopeStat{Memory: 1024})
	checkResources(t, &txn1.(*resourceScope).rc, network.ScopeStat{Memory: 1024})
	checkResources(t, &s.rc, network.ScopeStat{Memory: 1024})

	if _, err := txn4.ReserveMemory(1024); err != nil {
		t.Fatal(err)
	}
	checkResources(t, &txn4.(*resourceScope).rc, network.ScopeStat{Memory: 1024})
	checkResources(t, &txn3.(*resourceScope).rc, network.ScopeStat{Memory: 1024})
	checkResources(t, &txn2.(*resourceScope).rc, network.ScopeStat{Memory: 1024})
	checkResources(t, &txn1.(*resourceScope).rc, network.ScopeStat{Memory: 2048})
	checkResources(t, &s.rc, network.ScopeStat{Memory: 2048})

	if _, err := txn5.ReserveMemory(1024); err != nil {
		t.Fatal(err)
	}
	checkResources(t, &txn5.(*resourceScope).rc, network.ScopeStat{Memory: 1024})
	checkResources(t, &txn4.(*resourceScope).rc, network.ScopeStat{Memory: 1024})
	checkResources(t, &txn3.(*resourceScope).rc, network.ScopeStat{Memory: 1024})
	checkResources(t, &txn2.(*resourceScope).rc, network.ScopeStat{Memory: 2048})
	checkResources(t, &txn1.(*resourceScope).rc, network.ScopeStat{Memory: 3072})
	checkResources(t, &s.rc, network.ScopeStat{Memory: 3072})

	if _, err := txn1.ReserveMemory(1024); err != nil {
		t.Fatal(err)
	}
	checkResources(t, &txn5.(*resourceScope).rc, network.ScopeStat{Memory: 1024})
	checkResources(t, &txn4.(*resourceScope).rc, network.ScopeStat{Memory: 1024})
	checkResources(t, &txn3.(*resourceScope).rc, network.ScopeStat{Memory: 1024})
	checkResources(t, &txn2.(*resourceScope).rc, network.ScopeStat{Memory: 2048})
	checkResources(t, &txn1.(*resourceScope).rc, network.ScopeStat{Memory: 4096})
	checkResources(t, &s.rc, network.ScopeStat{Memory: 4096})

	if _, err := txn5.ReserveMemory(1024); err == nil {
		t.Fatal("expected ReserveMemory to fail")
	}
	if _, err := txn4.ReserveMemory(1024); err == nil {
		t.Fatal("expected ReserveMemory to fail")
	}
	if _, err := txn3.ReserveMemory(1024); err == nil {
		t.Fatal("expected ReserveMemory to fail")
	}
	if _, err := txn2.ReserveMemory(1024); err == nil {
		t.Fatal("expected ReserveMemory to fail")
	}
	checkResources(t, &txn5.(*resourceScope).rc, network.ScopeStat{Memory: 1024})
	checkResources(t, &txn4.(*resourceScope).rc, network.ScopeStat{Memory: 1024})
	checkResources(t, &txn3.(*resourceScope).rc, network.ScopeStat{Memory: 1024})
	checkResources(t, &txn2.(*resourceScope).rc, network.ScopeStat{Memory: 2048})
	checkResources(t, &txn1.(*resourceScope).rc, network.ScopeStat{Memory: 4096})
	checkResources(t, &s.rc, network.ScopeStat{Memory: 4096})

	txn1.Done()
	checkResources(t, &s.rc, network.ScopeStat{})
}

func TestResourceScopeDAG(t *testing.T) {
	// A small DAG of scopes
	// s1
	// +---> s2
	//        +------------> s5
	//        +----
	// +---> s3 +.  \
	//          | \  -----+-> s4 (a diamond!)
	//          |  ------/
	//          \
	//           ------> s6
	s1 := newResourceScope(
		&StaticLimit{
			Memory:          4096,
			StreamsInbound:  4,
			StreamsOutbound: 4,
			ConnsInbound:    4,
			ConnsOutbound:   4,
			FD:              4,
		},
		nil,
	)
	s2 := newResourceScope(
		&StaticLimit{
			Memory:          2048,
			StreamsInbound:  2,
			StreamsOutbound: 2,
			ConnsInbound:    2,
			ConnsOutbound:   2,
			FD:              2,
		},
		[]*resourceScope{s1},
	)
	s3 := newResourceScope(
		&StaticLimit{
			Memory:          2048,
			StreamsInbound:  2,
			StreamsOutbound: 2,
			ConnsInbound:    2,
			ConnsOutbound:   2,
			FD:              2,
		},
		[]*resourceScope{s1},
	)
	s4 := newResourceScope(
		&StaticLimit{
			Memory:          2048,
			StreamsInbound:  2,
			StreamsOutbound: 2,
			ConnsInbound:    2,
			ConnsOutbound:   2,
			FD:              2,
		},
		[]*resourceScope{s2, s3, s1},
	)
	s5 := newResourceScope(
		&StaticLimit{
			Memory:          2048,
			StreamsInbound:  2,
			StreamsOutbound: 2,
			ConnsInbound:    2,
			ConnsOutbound:   2,
			FD:              2,
		},
		[]*resourceScope{s2, s1},
	)
	s6 := newResourceScope(
		&StaticLimit{
			Memory:          2048,
			StreamsInbound:  2,
			StreamsOutbound: 2,
			ConnsInbound:    2,
			ConnsOutbound:   2,
			FD:              2,
		},
		[]*resourceScope{s3, s1},
	)

	if _, err := s4.ReserveMemory(1024); err != nil {
		t.Fatal(err)
	}
	checkResources(t, &s6.rc, network.ScopeStat{})
	checkResources(t, &s5.rc, network.ScopeStat{})
	checkResources(t, &s4.rc, network.ScopeStat{Memory: 1024})
	checkResources(t, &s3.rc, network.ScopeStat{Memory: 1024})
	checkResources(t, &s2.rc, network.ScopeStat{Memory: 1024})
	checkResources(t, &s1.rc, network.ScopeStat{Memory: 1024})

	if _, err := s5.ReserveMemory(1024); err != nil {
		t.Fatal(err)
	}
	checkResources(t, &s6.rc, network.ScopeStat{})
	checkResources(t, &s5.rc, network.ScopeStat{Memory: 1024})
	checkResources(t, &s4.rc, network.ScopeStat{Memory: 1024})
	checkResources(t, &s3.rc, network.ScopeStat{Memory: 1024})
	checkResources(t, &s2.rc, network.ScopeStat{Memory: 2048})
	checkResources(t, &s1.rc, network.ScopeStat{Memory: 2048})

	if _, err := s6.ReserveMemory(1024); err != nil {
		t.Fatal(err)
	}
	checkResources(t, &s6.rc, network.ScopeStat{Memory: 1024})
	checkResources(t, &s5.rc, network.ScopeStat{Memory: 1024})
	checkResources(t, &s4.rc, network.ScopeStat{Memory: 1024})
	checkResources(t, &s3.rc, network.ScopeStat{Memory: 2048})
	checkResources(t, &s2.rc, network.ScopeStat{Memory: 2048})
	checkResources(t, &s1.rc, network.ScopeStat{Memory: 3072})

	if _, err := s4.ReserveMemory(1024); err == nil {
		t.Fatal("expcted ReserveMemory to fail")
	}
	if _, err := s5.ReserveMemory(1024); err == nil {
		t.Fatal("expcted ReserveMemory to fail")
	}
	if _, err := s6.ReserveMemory(1024); err == nil {
		t.Fatal("expcted ReserveMemory to fail")
	}

	checkResources(t, &s6.rc, network.ScopeStat{Memory: 1024})
	checkResources(t, &s5.rc, network.ScopeStat{Memory: 1024})
	checkResources(t, &s4.rc, network.ScopeStat{Memory: 1024})
	checkResources(t, &s3.rc, network.ScopeStat{Memory: 2048})
	checkResources(t, &s2.rc, network.ScopeStat{Memory: 2048})
	checkResources(t, &s1.rc, network.ScopeStat{Memory: 3072})

	s4.ReleaseMemory(1024)
	checkResources(t, &s6.rc, network.ScopeStat{Memory: 1024})
	checkResources(t, &s5.rc, network.ScopeStat{Memory: 1024})
	checkResources(t, &s4.rc, network.ScopeStat{})
	checkResources(t, &s3.rc, network.ScopeStat{Memory: 1024})
	checkResources(t, &s2.rc, network.ScopeStat{Memory: 1024})
	checkResources(t, &s1.rc, network.ScopeStat{Memory: 2048})

	s5.ReleaseMemory(1024)
	checkResources(t, &s6.rc, network.ScopeStat{Memory: 1024})
	checkResources(t, &s5.rc, network.ScopeStat{})
	checkResources(t, &s4.rc, network.ScopeStat{})
	checkResources(t, &s3.rc, network.ScopeStat{Memory: 1024})
	checkResources(t, &s2.rc, network.ScopeStat{})
	checkResources(t, &s1.rc, network.ScopeStat{Memory: 1024})

	s6.ReleaseMemory(1024)
	checkResources(t, &s6.rc, network.ScopeStat{})
	checkResources(t, &s5.rc, network.ScopeStat{})
	checkResources(t, &s4.rc, network.ScopeStat{})
	checkResources(t, &s3.rc, network.ScopeStat{})
	checkResources(t, &s2.rc, network.ScopeStat{})
	checkResources(t, &s1.rc, network.ScopeStat{})

	if err := s4.AddStream(network.DirInbound); err != nil {
		t.Fatal(err)
	}
	checkResources(t, &s6.rc, network.ScopeStat{})
	checkResources(t, &s5.rc, network.ScopeStat{})
	checkResources(t, &s4.rc, network.ScopeStat{NumStreamsInbound: 1})
	checkResources(t, &s3.rc, network.ScopeStat{NumStreamsInbound: 1})
	checkResources(t, &s2.rc, network.ScopeStat{NumStreamsInbound: 1})
	checkResources(t, &s1.rc, network.ScopeStat{NumStreamsInbound: 1})

	if err := s5.AddStream(network.DirInbound); err != nil {
		t.Fatal(err)
	}
	checkResources(t, &s6.rc, network.ScopeStat{})
	checkResources(t, &s5.rc, network.ScopeStat{NumStreamsInbound: 1})
	checkResources(t, &s4.rc, network.ScopeStat{NumStreamsInbound: 1})
	checkResources(t, &s3.rc, network.ScopeStat{NumStreamsInbound: 1})
	checkResources(t, &s2.rc, network.ScopeStat{NumStreamsInbound: 2})
	checkResources(t, &s1.rc, network.ScopeStat{NumStreamsInbound: 2})

	if err := s6.AddStream(network.DirInbound); err != nil {
		t.Fatal(err)
	}
	checkResources(t, &s6.rc, network.ScopeStat{NumStreamsInbound: 1})
	checkResources(t, &s5.rc, network.ScopeStat{NumStreamsInbound: 1})
	checkResources(t, &s4.rc, network.ScopeStat{NumStreamsInbound: 1})
	checkResources(t, &s3.rc, network.ScopeStat{NumStreamsInbound: 2})
	checkResources(t, &s2.rc, network.ScopeStat{NumStreamsInbound: 2})
	checkResources(t, &s1.rc, network.ScopeStat{NumStreamsInbound: 3})

	if err := s4.AddStream(network.DirInbound); err == nil {
		t.Fatal("expected AddStream to fail")
	}
	if err := s5.AddStream(network.DirInbound); err == nil {
		t.Fatal("expected AddStream to fail")
	}
	if err := s6.AddStream(network.DirInbound); err == nil {
		t.Fatal("expected AddStream to fail")
	}
	checkResources(t, &s6.rc, network.ScopeStat{NumStreamsInbound: 1})
	checkResources(t, &s5.rc, network.ScopeStat{NumStreamsInbound: 1})
	checkResources(t, &s4.rc, network.ScopeStat{NumStreamsInbound: 1})
	checkResources(t, &s3.rc, network.ScopeStat{NumStreamsInbound: 2})
	checkResources(t, &s2.rc, network.ScopeStat{NumStreamsInbound: 2})
	checkResources(t, &s1.rc, network.ScopeStat{NumStreamsInbound: 3})

	s4.RemoveStream(network.DirInbound)
	checkResources(t, &s6.rc, network.ScopeStat{NumStreamsInbound: 1})
	checkResources(t, &s5.rc, network.ScopeStat{NumStreamsInbound: 1})
	checkResources(t, &s4.rc, network.ScopeStat{})
	checkResources(t, &s3.rc, network.ScopeStat{NumStreamsInbound: 1})
	checkResources(t, &s2.rc, network.ScopeStat{NumStreamsInbound: 1})
	checkResources(t, &s1.rc, network.ScopeStat{NumStreamsInbound: 2})

	s5.RemoveStream(network.DirInbound)
	checkResources(t, &s6.rc, network.ScopeStat{NumStreamsInbound: 1})
	checkResources(t, &s5.rc, network.ScopeStat{})
	checkResources(t, &s4.rc, network.ScopeStat{})
	checkResources(t, &s3.rc, network.ScopeStat{NumStreamsInbound: 1})
	checkResources(t, &s2.rc, network.ScopeStat{})
	checkResources(t, &s1.rc, network.ScopeStat{NumStreamsInbound: 1})

	s6.RemoveStream(network.DirInbound)
	checkResources(t, &s6.rc, network.ScopeStat{})
	checkResources(t, &s5.rc, network.ScopeStat{})
	checkResources(t, &s4.rc, network.ScopeStat{})
	checkResources(t, &s3.rc, network.ScopeStat{})
	checkResources(t, &s2.rc, network.ScopeStat{})
	checkResources(t, &s1.rc, network.ScopeStat{})

	if err := s4.AddStream(network.DirOutbound); err != nil {
		t.Fatal(err)
	}
	checkResources(t, &s6.rc, network.ScopeStat{})
	checkResources(t, &s5.rc, network.ScopeStat{})
	checkResources(t, &s4.rc, network.ScopeStat{NumStreamsOutbound: 1})
	checkResources(t, &s3.rc, network.ScopeStat{NumStreamsOutbound: 1})
	checkResources(t, &s2.rc, network.ScopeStat{NumStreamsOutbound: 1})
	checkResources(t, &s1.rc, network.ScopeStat{NumStreamsOutbound: 1})

	if err := s5.AddStream(network.DirOutbound); err != nil {
		t.Fatal(err)
	}
	checkResources(t, &s6.rc, network.ScopeStat{})
	checkResources(t, &s5.rc, network.ScopeStat{NumStreamsOutbound: 1})
	checkResources(t, &s4.rc, network.ScopeStat{NumStreamsOutbound: 1})
	checkResources(t, &s3.rc, network.ScopeStat{NumStreamsOutbound: 1})
	checkResources(t, &s2.rc, network.ScopeStat{NumStreamsOutbound: 2})
	checkResources(t, &s1.rc, network.ScopeStat{NumStreamsOutbound: 2})

	if err := s6.AddStream(network.DirOutbound); err != nil {
		t.Fatal(err)
	}
	checkResources(t, &s6.rc, network.ScopeStat{NumStreamsOutbound: 1})
	checkResources(t, &s5.rc, network.ScopeStat{NumStreamsOutbound: 1})
	checkResources(t, &s4.rc, network.ScopeStat{NumStreamsOutbound: 1})
	checkResources(t, &s3.rc, network.ScopeStat{NumStreamsOutbound: 2})
	checkResources(t, &s2.rc, network.ScopeStat{NumStreamsOutbound: 2})
	checkResources(t, &s1.rc, network.ScopeStat{NumStreamsOutbound: 3})

	if err := s4.AddStream(network.DirOutbound); err == nil {
		t.Fatal("expected AddStream to fail")
	}
	if err := s5.AddStream(network.DirOutbound); err == nil {
		t.Fatal("expected AddStream to fail")
	}
	if err := s6.AddStream(network.DirOutbound); err == nil {
		t.Fatal("expected AddStream to fail")
	}
	checkResources(t, &s6.rc, network.ScopeStat{NumStreamsOutbound: 1})
	checkResources(t, &s5.rc, network.ScopeStat{NumStreamsOutbound: 1})
	checkResources(t, &s4.rc, network.ScopeStat{NumStreamsOutbound: 1})
	checkResources(t, &s3.rc, network.ScopeStat{NumStreamsOutbound: 2})
	checkResources(t, &s2.rc, network.ScopeStat{NumStreamsOutbound: 2})
	checkResources(t, &s1.rc, network.ScopeStat{NumStreamsOutbound: 3})

	s4.RemoveStream(network.DirOutbound)
	checkResources(t, &s6.rc, network.ScopeStat{NumStreamsOutbound: 1})
	checkResources(t, &s5.rc, network.ScopeStat{NumStreamsOutbound: 1})
	checkResources(t, &s4.rc, network.ScopeStat{})
	checkResources(t, &s3.rc, network.ScopeStat{NumStreamsOutbound: 1})
	checkResources(t, &s2.rc, network.ScopeStat{NumStreamsOutbound: 1})
	checkResources(t, &s1.rc, network.ScopeStat{NumStreamsOutbound: 2})

	s5.RemoveStream(network.DirOutbound)
	checkResources(t, &s6.rc, network.ScopeStat{NumStreamsOutbound: 1})
	checkResources(t, &s5.rc, network.ScopeStat{})
	checkResources(t, &s4.rc, network.ScopeStat{})
	checkResources(t, &s3.rc, network.ScopeStat{NumStreamsOutbound: 1})
	checkResources(t, &s2.rc, network.ScopeStat{})
	checkResources(t, &s1.rc, network.ScopeStat{NumStreamsOutbound: 1})

	s6.RemoveStream(network.DirOutbound)
	checkResources(t, &s6.rc, network.ScopeStat{})
	checkResources(t, &s5.rc, network.ScopeStat{})
	checkResources(t, &s4.rc, network.ScopeStat{})
	checkResources(t, &s3.rc, network.ScopeStat{})
	checkResources(t, &s2.rc, network.ScopeStat{})
	checkResources(t, &s1.rc, network.ScopeStat{})

	if err := s4.AddConn(network.DirInbound, false); err != nil {
		t.Fatal(err)
	}
	checkResources(t, &s6.rc, network.ScopeStat{})
	checkResources(t, &s5.rc, network.ScopeStat{})
	checkResources(t, &s4.rc, network.ScopeStat{NumConnsInbound: 1})
	checkResources(t, &s3.rc, network.ScopeStat{NumConnsInbound: 1})
	checkResources(t, &s2.rc, network.ScopeStat{NumConnsInbound: 1})
	checkResources(t, &s1.rc, network.ScopeStat{NumConnsInbound: 1})

	if err := s5.AddConn(network.DirInbound, false); err != nil {
		t.Fatal(err)
	}
	checkResources(t, &s6.rc, network.ScopeStat{})
	checkResources(t, &s5.rc, network.ScopeStat{NumConnsInbound: 1})
	checkResources(t, &s4.rc, network.ScopeStat{NumConnsInbound: 1})
	checkResources(t, &s3.rc, network.ScopeStat{NumConnsInbound: 1})
	checkResources(t, &s2.rc, network.ScopeStat{NumConnsInbound: 2})
	checkResources(t, &s1.rc, network.ScopeStat{NumConnsInbound: 2})

	if err := s6.AddConn(network.DirInbound, false); err != nil {
		t.Fatal(err)
	}
	checkResources(t, &s6.rc, network.ScopeStat{NumConnsInbound: 1})
	checkResources(t, &s5.rc, network.ScopeStat{NumConnsInbound: 1})
	checkResources(t, &s4.rc, network.ScopeStat{NumConnsInbound: 1})
	checkResources(t, &s3.rc, network.ScopeStat{NumConnsInbound: 2})
	checkResources(t, &s2.rc, network.ScopeStat{NumConnsInbound: 2})
	checkResources(t, &s1.rc, network.ScopeStat{NumConnsInbound: 3})

	if err := s4.AddConn(network.DirInbound, false); err == nil {
		t.Fatal("expected AddConn to fail")
	}
	if err := s5.AddConn(network.DirInbound, false); err == nil {
		t.Fatal("expected AddConn to fail")
	}
	if err := s6.AddConn(network.DirInbound, false); err == nil {
		t.Fatal("expected AddConn to fail")
	}
	checkResources(t, &s6.rc, network.ScopeStat{NumConnsInbound: 1})
	checkResources(t, &s5.rc, network.ScopeStat{NumConnsInbound: 1})
	checkResources(t, &s4.rc, network.ScopeStat{NumConnsInbound: 1})
	checkResources(t, &s3.rc, network.ScopeStat{NumConnsInbound: 2})
	checkResources(t, &s2.rc, network.ScopeStat{NumConnsInbound: 2})
	checkResources(t, &s1.rc, network.ScopeStat{NumConnsInbound: 3})

	s4.RemoveConn(network.DirInbound, false)
	checkResources(t, &s6.rc, network.ScopeStat{NumConnsInbound: 1})
	checkResources(t, &s5.rc, network.ScopeStat{NumConnsInbound: 1})
	checkResources(t, &s4.rc, network.ScopeStat{})
	checkResources(t, &s3.rc, network.ScopeStat{NumConnsInbound: 1})
	checkResources(t, &s2.rc, network.ScopeStat{NumConnsInbound: 1})
	checkResources(t, &s1.rc, network.ScopeStat{NumConnsInbound: 2})

	s5.RemoveConn(network.DirInbound, false)
	checkResources(t, &s6.rc, network.ScopeStat{NumConnsInbound: 1})
	checkResources(t, &s5.rc, network.ScopeStat{})
	checkResources(t, &s4.rc, network.ScopeStat{})
	checkResources(t, &s3.rc, network.ScopeStat{NumConnsInbound: 1})
	checkResources(t, &s2.rc, network.ScopeStat{})
	checkResources(t, &s1.rc, network.ScopeStat{NumConnsInbound: 1})

	s6.RemoveConn(network.DirInbound, false)
	checkResources(t, &s6.rc, network.ScopeStat{})
	checkResources(t, &s5.rc, network.ScopeStat{})
	checkResources(t, &s4.rc, network.ScopeStat{})
	checkResources(t, &s3.rc, network.ScopeStat{})
	checkResources(t, &s2.rc, network.ScopeStat{})
	checkResources(t, &s1.rc, network.ScopeStat{})

	if err := s4.AddConn(network.DirOutbound, false); err != nil {
		t.Fatal(err)
	}
	checkResources(t, &s6.rc, network.ScopeStat{})
	checkResources(t, &s5.rc, network.ScopeStat{})
	checkResources(t, &s4.rc, network.ScopeStat{NumConnsOutbound: 1})
	checkResources(t, &s3.rc, network.ScopeStat{NumConnsOutbound: 1})
	checkResources(t, &s2.rc, network.ScopeStat{NumConnsOutbound: 1})
	checkResources(t, &s1.rc, network.ScopeStat{NumConnsOutbound: 1})

	if err := s5.AddConn(network.DirOutbound, false); err != nil {
		t.Fatal(err)
	}
	checkResources(t, &s6.rc, network.ScopeStat{})
	checkResources(t, &s5.rc, network.ScopeStat{NumConnsOutbound: 1})
	checkResources(t, &s4.rc, network.ScopeStat{NumConnsOutbound: 1})
	checkResources(t, &s3.rc, network.ScopeStat{NumConnsOutbound: 1})
	checkResources(t, &s2.rc, network.ScopeStat{NumConnsOutbound: 2})
	checkResources(t, &s1.rc, network.ScopeStat{NumConnsOutbound: 2})

	if err := s6.AddConn(network.DirOutbound, false); err != nil {
		t.Fatal(err)
	}
	checkResources(t, &s6.rc, network.ScopeStat{NumConnsOutbound: 1})
	checkResources(t, &s5.rc, network.ScopeStat{NumConnsOutbound: 1})
	checkResources(t, &s4.rc, network.ScopeStat{NumConnsOutbound: 1})
	checkResources(t, &s3.rc, network.ScopeStat{NumConnsOutbound: 2})
	checkResources(t, &s2.rc, network.ScopeStat{NumConnsOutbound: 2})
	checkResources(t, &s1.rc, network.ScopeStat{NumConnsOutbound: 3})

	if err := s4.AddConn(network.DirOutbound, false); err == nil {
		t.Fatal("expected AddConn to fail")
	}
	if err := s5.AddConn(network.DirOutbound, false); err == nil {
		t.Fatal("expected AddConn to fail")
	}
	if err := s6.AddConn(network.DirOutbound, false); err == nil {
		t.Fatal("expected AddConn to fail")
	}
	checkResources(t, &s6.rc, network.ScopeStat{NumConnsOutbound: 1})
	checkResources(t, &s5.rc, network.ScopeStat{NumConnsOutbound: 1})
	checkResources(t, &s4.rc, network.ScopeStat{NumConnsOutbound: 1})
	checkResources(t, &s3.rc, network.ScopeStat{NumConnsOutbound: 2})
	checkResources(t, &s2.rc, network.ScopeStat{NumConnsOutbound: 2})
	checkResources(t, &s1.rc, network.ScopeStat{NumConnsOutbound: 3})

	s4.RemoveConn(network.DirOutbound, false)
	checkResources(t, &s6.rc, network.ScopeStat{NumConnsOutbound: 1})
	checkResources(t, &s5.rc, network.ScopeStat{NumConnsOutbound: 1})
	checkResources(t, &s4.rc, network.ScopeStat{})
	checkResources(t, &s3.rc, network.ScopeStat{NumConnsOutbound: 1})
	checkResources(t, &s2.rc, network.ScopeStat{NumConnsOutbound: 1})
	checkResources(t, &s1.rc, network.ScopeStat{NumConnsOutbound: 2})

	s5.RemoveConn(network.DirOutbound, false)
	checkResources(t, &s6.rc, network.ScopeStat{NumConnsOutbound: 1})
	checkResources(t, &s5.rc, network.ScopeStat{})
	checkResources(t, &s4.rc, network.ScopeStat{})
	checkResources(t, &s3.rc, network.ScopeStat{NumConnsOutbound: 1})
	checkResources(t, &s2.rc, network.ScopeStat{})
	checkResources(t, &s1.rc, network.ScopeStat{NumConnsOutbound: 1})

	s6.RemoveConn(network.DirOutbound, false)
	checkResources(t, &s6.rc, network.ScopeStat{})
	checkResources(t, &s5.rc, network.ScopeStat{})
	checkResources(t, &s4.rc, network.ScopeStat{})
	checkResources(t, &s3.rc, network.ScopeStat{})
	checkResources(t, &s2.rc, network.ScopeStat{})
	checkResources(t, &s1.rc, network.ScopeStat{})

	if err := s4.AddConn(network.DirInbound, true); err != nil {
		t.Fatal(err)
	}
	checkResources(t, &s6.rc, network.ScopeStat{})
	checkResources(t, &s5.rc, network.ScopeStat{})
	checkResources(t, &s4.rc, network.ScopeStat{NumConnsInbound: 1, NumFD: 1})
	checkResources(t, &s3.rc, network.ScopeStat{NumConnsInbound: 1, NumFD: 1})
	checkResources(t, &s2.rc, network.ScopeStat{NumConnsInbound: 1, NumFD: 1})
	checkResources(t, &s1.rc, network.ScopeStat{NumConnsInbound: 1, NumFD: 1})

	if err := s5.AddConn(network.DirInbound, true); err != nil {
		t.Fatal(err)
	}
	checkResources(t, &s6.rc, network.ScopeStat{})
	checkResources(t, &s5.rc, network.ScopeStat{NumConnsInbound: 1, NumFD: 1})
	checkResources(t, &s4.rc, network.ScopeStat{NumConnsInbound: 1, NumFD: 1})
	checkResources(t, &s3.rc, network.ScopeStat{NumConnsInbound: 1, NumFD: 1})
	checkResources(t, &s2.rc, network.ScopeStat{NumConnsInbound: 2, NumFD: 2})
	checkResources(t, &s1.rc, network.ScopeStat{NumConnsInbound: 2, NumFD: 2})

	if err := s6.AddConn(network.DirInbound, true); err != nil {
		t.Fatal(err)
	}
	checkResources(t, &s6.rc, network.ScopeStat{NumConnsInbound: 1, NumFD: 1})
	checkResources(t, &s5.rc, network.ScopeStat{NumConnsInbound: 1, NumFD: 1})
	checkResources(t, &s4.rc, network.ScopeStat{NumConnsInbound: 1, NumFD: 1})
	checkResources(t, &s3.rc, network.ScopeStat{NumConnsInbound: 2, NumFD: 2})
	checkResources(t, &s2.rc, network.ScopeStat{NumConnsInbound: 2, NumFD: 2})
	checkResources(t, &s1.rc, network.ScopeStat{NumConnsInbound: 3, NumFD: 3})

	if err := s4.AddConn(network.DirOutbound, true); err == nil {
		t.Fatal("expected AddConn to fail")
	}
	if err := s5.AddConn(network.DirOutbound, true); err == nil {
		t.Fatal("expected AddConn to fail")
	}
	if err := s6.AddConn(network.DirOutbound, true); err == nil {
		t.Fatal("expected AddConn to fail")
	}
	checkResources(t, &s6.rc, network.ScopeStat{NumConnsInbound: 1, NumFD: 1})
	checkResources(t, &s5.rc, network.ScopeStat{NumConnsInbound: 1, NumFD: 1})
	checkResources(t, &s4.rc, network.ScopeStat{NumConnsInbound: 1, NumFD: 1})
	checkResources(t, &s3.rc, network.ScopeStat{NumConnsInbound: 2, NumFD: 2})
	checkResources(t, &s2.rc, network.ScopeStat{NumConnsInbound: 2, NumFD: 2})
	checkResources(t, &s1.rc, network.ScopeStat{NumConnsInbound: 3, NumFD: 3})

	s4.RemoveConn(network.DirInbound, true)
	checkResources(t, &s6.rc, network.ScopeStat{NumConnsInbound: 1, NumFD: 1})
	checkResources(t, &s5.rc, network.ScopeStat{NumConnsInbound: 1, NumFD: 1})
	checkResources(t, &s4.rc, network.ScopeStat{})
	checkResources(t, &s3.rc, network.ScopeStat{NumConnsInbound: 1, NumFD: 1})
	checkResources(t, &s2.rc, network.ScopeStat{NumConnsInbound: 1, NumFD: 1})
	checkResources(t, &s1.rc, network.ScopeStat{NumConnsInbound: 2, NumFD: 2})

	s5.RemoveConn(network.DirInbound, true)
	checkResources(t, &s6.rc, network.ScopeStat{NumConnsInbound: 1, NumFD: 1})
	checkResources(t, &s5.rc, network.ScopeStat{})
	checkResources(t, &s4.rc, network.ScopeStat{})
	checkResources(t, &s3.rc, network.ScopeStat{NumConnsInbound: 1, NumFD: 1})
	checkResources(t, &s2.rc, network.ScopeStat{})
	checkResources(t, &s1.rc, network.ScopeStat{NumConnsInbound: 1, NumFD: 1})

	s6.RemoveConn(network.DirInbound, true)
	checkResources(t, &s6.rc, network.ScopeStat{})
	checkResources(t, &s5.rc, network.ScopeStat{})
	checkResources(t, &s4.rc, network.ScopeStat{})
	checkResources(t, &s3.rc, network.ScopeStat{})
	checkResources(t, &s2.rc, network.ScopeStat{})
	checkResources(t, &s1.rc, network.ScopeStat{})
}

func TestResourceScopeDAGTxn(t *testing.T) {
	// A small DAG of scopes
	// s1
	// +---> s2
	//        +------------> s5
	//        +----
	// +---> s3 +.  \
	//          | \  -----+-> s4 (a diamond!)
	//          |  ------/
	//          \
	//           ------> s6
	s1 := newResourceScope(
		&StaticLimit{
			Memory: 8192,
		},
		nil,
	)
	s2 := newResourceScope(
		&StaticLimit{
			Memory: 4096 + 2048,
		},
		[]*resourceScope{s1},
	)
	s3 := newResourceScope(
		&StaticLimit{
			Memory: 4096 + 2048,
		},
		[]*resourceScope{s1},
	)
	s4 := newResourceScope(
		&StaticLimit{
			Memory: 4096 + 1024,
		},
		[]*resourceScope{s2, s3, s1},
	)
	s5 := newResourceScope(
		&StaticLimit{
			Memory: 4096 + 1024,
		},
		[]*resourceScope{s2, s1},
	)
	s6 := newResourceScope(
		&StaticLimit{
			Memory: 4096 + 1024,
		},
		[]*resourceScope{s3, s1},
	)

	txn4, err := s4.BeginTransaction()
	if err != nil {
		t.Fatal(err)
	}

	txn5, err := s5.BeginTransaction()
	if err != nil {
		t.Fatal(err)
	}

	txn6, err := s6.BeginTransaction()
	if err != nil {
		t.Fatal(err)
	}

	if _, err := txn4.ReserveMemory(1024); err != nil {
		t.Fatal(err)
	}
	checkResources(t, &s4.rc, network.ScopeStat{Memory: 1024})
	checkResources(t, &s3.rc, network.ScopeStat{Memory: 1024})
	checkResources(t, &s2.rc, network.ScopeStat{Memory: 1024})
	checkResources(t, &s1.rc, network.ScopeStat{Memory: 1024})

	if _, err := txn5.ReserveMemory(1024); err != nil {
		t.Fatal(err)
	}
	checkResources(t, &s5.rc, network.ScopeStat{Memory: 1024})
	checkResources(t, &s4.rc, network.ScopeStat{Memory: 1024})
	checkResources(t, &s3.rc, network.ScopeStat{Memory: 1024})
	checkResources(t, &s2.rc, network.ScopeStat{Memory: 2048})
	checkResources(t, &s1.rc, network.ScopeStat{Memory: 2048})

	if _, err := txn6.ReserveMemory(1024); err != nil {
		t.Fatal(err)
	}
	checkResources(t, &s6.rc, network.ScopeStat{Memory: 1024})
	checkResources(t, &s5.rc, network.ScopeStat{Memory: 1024})
	checkResources(t, &s4.rc, network.ScopeStat{Memory: 1024})
	checkResources(t, &s3.rc, network.ScopeStat{Memory: 2048})
	checkResources(t, &s2.rc, network.ScopeStat{Memory: 2048})
	checkResources(t, &s1.rc, network.ScopeStat{Memory: 3072})

	if _, err := txn4.ReserveMemory(4096); err != nil {
		t.Fatal(err)
	}
	checkResources(t, &s6.rc, network.ScopeStat{Memory: 1024})
	checkResources(t, &s5.rc, network.ScopeStat{Memory: 1024})
	checkResources(t, &s4.rc, network.ScopeStat{Memory: 1024 + 4096})
	checkResources(t, &s3.rc, network.ScopeStat{Memory: 2048 + 4096})
	checkResources(t, &s2.rc, network.ScopeStat{Memory: 2048 + 4096})
	checkResources(t, &s1.rc, network.ScopeStat{Memory: 3072 + 4096})

	if _, err := txn4.ReserveMemory(1024); err == nil {
		t.Fatal("expected ReserveMemory to fail")
	}
	if _, err := txn5.ReserveMemory(1024); err == nil {
		t.Fatal("expected ReserveMemory to fail")
	}
	if _, err := txn6.ReserveMemory(1024); err == nil {
		t.Fatal("expected ReserveMemory to fail")
	}
	checkResources(t, &s6.rc, network.ScopeStat{Memory: 1024})
	checkResources(t, &s5.rc, network.ScopeStat{Memory: 1024})
	checkResources(t, &s4.rc, network.ScopeStat{Memory: 1024 + 4096})
	checkResources(t, &s3.rc, network.ScopeStat{Memory: 2048 + 4096})
	checkResources(t, &s2.rc, network.ScopeStat{Memory: 2048 + 4096})
	checkResources(t, &s1.rc, network.ScopeStat{Memory: 3072 + 4096})

	txn4.Done()

	checkResources(t, &s6.rc, network.ScopeStat{Memory: 1024})
	checkResources(t, &s5.rc, network.ScopeStat{Memory: 1024})
	checkResources(t, &s4.rc, network.ScopeStat{})
	checkResources(t, &s3.rc, network.ScopeStat{Memory: 1024})
	checkResources(t, &s2.rc, network.ScopeStat{Memory: 1024})
	checkResources(t, &s1.rc, network.ScopeStat{Memory: 2048})

	if _, err := txn5.ReserveMemory(1024); err != nil {
		t.Fatal(err)
	}
	if _, err := txn6.ReserveMemory(1024); err != nil {
		t.Fatal(err)
	}

	checkResources(t, &s6.rc, network.ScopeStat{Memory: 2048})
	checkResources(t, &s5.rc, network.ScopeStat{Memory: 2048})
	checkResources(t, &s4.rc, network.ScopeStat{})
	checkResources(t, &s3.rc, network.ScopeStat{Memory: 2048})
	checkResources(t, &s2.rc, network.ScopeStat{Memory: 2048})
	checkResources(t, &s1.rc, network.ScopeStat{Memory: 4096})

	txn5.Done()
	txn6.Done()

	checkResources(t, &s6.rc, network.ScopeStat{})
	checkResources(t, &s5.rc, network.ScopeStat{})
	checkResources(t, &s4.rc, network.ScopeStat{})
	checkResources(t, &s3.rc, network.ScopeStat{})
	checkResources(t, &s2.rc, network.ScopeStat{})
	checkResources(t, &s1.rc, network.ScopeStat{})
}
