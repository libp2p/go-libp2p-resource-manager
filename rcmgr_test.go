package rcmgr

import (
	"testing"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
)

func TestResourceManager(t *testing.T) {
	peerA := peer.ID("A")
	peerB := peer.ID("B")
	protoA := protocol.ID("/A")
	protoB := protocol.ID("/B")
	svcA := "A.svc"
	mgr := NewResourceManager(
		&BasicLimiter{
			SystemLimits: &StaticLimit{
				Memory:          16384,
				StreamsInbound:  3,
				StreamsOutbound: 3,
				ConnsInbound:    3,
				ConnsOutbound:   3,
				FD:              2,
			},
			TransientLimits: &StaticLimit{
				Memory:          4096,
				StreamsInbound:  1,
				StreamsOutbound: 1,
				ConnsInbound:    1,
				ConnsOutbound:   1,
				FD:              1,
			},
			DefaultServiceLimits: &StaticLimit{
				Memory:          4096,
				StreamsInbound:  1,
				StreamsOutbound: 1,
				ConnsInbound:    1,
				ConnsOutbound:   1,
				FD:              1,
			},
			ServiceLimits: map[string]Limit{
				svcA: &StaticLimit{
					Memory:          8192,
					StreamsInbound:  2,
					StreamsOutbound: 2,
					ConnsInbound:    2,
					ConnsOutbound:   2,
					FD:              1,
				},
			},
			DefaultProtocolLimits: &StaticLimit{
				Memory:          4096,
				StreamsInbound:  1,
				StreamsOutbound: 1,
			},
			ProtocolLimits: map[protocol.ID]Limit{
				protoA: &StaticLimit{
					Memory:          8192,
					StreamsInbound:  2,
					StreamsOutbound: 2,
				},
			},
			DefaultPeerLimits: &StaticLimit{
				Memory:          4096,
				StreamsInbound:  1,
				StreamsOutbound: 1,
				ConnsInbound:    1,
				ConnsOutbound:   1,
				FD:              1,
			},
			PeerLimits: map[peer.ID]Limit{
				peerA: &StaticLimit{
					Memory:          8192,
					StreamsInbound:  2,
					StreamsOutbound: 2,
					ConnsInbound:    2,
					ConnsOutbound:   2,
					FD:              1,
				},
			},
			ConnLimits: &StaticLimit{
				Memory:        4096,
				ConnsInbound:  1,
				ConnsOutbound: 1,
				FD:            1,
			},
			StreamLimits: &StaticLimit{
				Memory:          4096,
				StreamsInbound:  1,
				StreamsOutbound: 1,
			},
		})

	defer mgr.Close()

	checkRefCnt := func(s *ResourceScope, count int) {
		t.Helper()
		if refCnt := s.refCnt; refCnt != count {
			t.Fatalf("expected refCnt of %d, got %d", count, refCnt)
		}
	}
	checkSystem := func(check func(s *ResourceScope)) {
		if err := mgr.ViewSystem(func(s network.ResourceScope) error {
			check(s.(*SystemScope).ResourceScope)
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	checkTransient := func(check func(s *ResourceScope)) {
		if err := mgr.ViewTransient(func(s network.ResourceScope) error {
			check(s.(*TransientScope).ResourceScope)
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	checkService := func(svc string, check func(s *ResourceScope)) {
		if err := mgr.ViewService(svc, func(s network.ServiceScope) error {
			check(s.(*ServiceScope).ResourceScope)
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	checkProtocol := func(p protocol.ID, check func(s *ResourceScope)) {
		if err := mgr.ViewProtocol(p, func(s network.ProtocolScope) error {
			check(s.(*ProtocolScope).ResourceScope)
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	checkPeer := func(p peer.ID, check func(s *ResourceScope)) {
		if err := mgr.ViewPeer(p, func(s network.PeerScope) error {
			check(s.(*PeerScope).ResourceScope)
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}

	checkSystem(func(s *ResourceScope) {
		checkRefCnt(s, 2)
		checkResources(t, &s.rc, network.ScopeStat{})
	})
	checkTransient(func(s *ResourceScope) {
		checkRefCnt(s, 1)
		checkResources(t, &s.rc, network.ScopeStat{})
	})

	// open an inbound connection, using an fd
	conn, err := mgr.OpenConnection(network.DirInbound, true)
	if err != nil {
		t.Fatal(err)
	}

	checkSystem(func(s *ResourceScope) {
		checkRefCnt(s, 3)
		checkResources(t, &s.rc, network.ScopeStat{NumConnsInbound: 1, NumFD: 1})
	})
	checkTransient(func(s *ResourceScope) {
		checkRefCnt(s, 2)
		checkResources(t, &s.rc, network.ScopeStat{NumConnsInbound: 1, NumFD: 1})
	})

	// the connection is transient, we shouldn't be able to open a second one
	if _, err := mgr.OpenConnection(network.DirInbound, true); err == nil {
		t.Fatal("expected OpenConnection to fail")
	}
	if _, err := mgr.OpenConnection(network.DirInbound, false); err == nil {
		t.Fatal("expected OpenConnection to fail")
	}

	// close it to check resources are reclaimed
	conn.Done()

	checkSystem(func(s *ResourceScope) {
		checkRefCnt(s, 2)
		checkResources(t, &s.rc, network.ScopeStat{})
	})
	checkTransient(func(s *ResourceScope) {
		checkRefCnt(s, 1)
		checkResources(t, &s.rc, network.ScopeStat{})
	})

	// open another inbound connection, using an fd
	conn1, err := mgr.OpenConnection(network.DirInbound, true)
	if err != nil {
		t.Fatal(err)
	}

	checkSystem(func(s *ResourceScope) {
		checkRefCnt(s, 3)
		checkResources(t, &s.rc, network.ScopeStat{NumConnsInbound: 1, NumFD: 1})
	})
	checkTransient(func(s *ResourceScope) {
		checkRefCnt(s, 2)
		checkResources(t, &s.rc, network.ScopeStat{NumConnsInbound: 1, NumFD: 1})
	})

	// attach to a peer
	if err := conn1.SetPeer(peerA); err != nil {
		t.Fatal(err)
	}

	checkPeer(peerA, func(s *ResourceScope) {
		checkRefCnt(s, 2)
		checkResources(t, &s.rc, network.ScopeStat{NumConnsInbound: 1, NumFD: 1})
	})
	checkSystem(func(s *ResourceScope) {
		checkRefCnt(s, 4)
		checkResources(t, &s.rc, network.ScopeStat{NumConnsInbound: 1, NumFD: 1})
	})
	checkTransient(func(s *ResourceScope) {
		checkRefCnt(s, 1)
		checkResources(t, &s.rc, network.ScopeStat{})
	})

	// we should be able to open a second transient connection now
	conn2, err := mgr.OpenConnection(network.DirInbound, true)
	if err != nil {
		t.Fatal(err)
	}

	checkSystem(func(s *ResourceScope) {
		checkRefCnt(s, 5)
		checkResources(t, &s.rc, network.ScopeStat{NumConnsInbound: 2, NumFD: 2})
	})
	checkTransient(func(s *ResourceScope) {
		checkRefCnt(s, 2)
		checkResources(t, &s.rc, network.ScopeStat{NumConnsInbound: 1, NumFD: 1})
	})

	// but we shouldn't be able to attach it to the same peer due to the fd limit
	if err := conn2.SetPeer(peerA); err == nil {
		t.Fatal("expected SetPeer to fail")
	}

	checkPeer(peerA, func(s *ResourceScope) {
		checkRefCnt(s, 2)
		checkResources(t, &s.rc, network.ScopeStat{NumConnsInbound: 1, NumFD: 1})
	})
	checkSystem(func(s *ResourceScope) {
		checkRefCnt(s, 5)
		checkResources(t, &s.rc, network.ScopeStat{NumConnsInbound: 2, NumFD: 2})
	})
	checkTransient(func(s *ResourceScope) {
		checkRefCnt(s, 2)
		checkResources(t, &s.rc, network.ScopeStat{NumConnsInbound: 1, NumFD: 1})
	})

	// close it and reopen without using an FD -- we should be able to attach now
	conn2.Done()

	conn2, err = mgr.OpenConnection(network.DirInbound, false)
	if err != nil {
		t.Fatal(err)
	}

	checkSystem(func(s *ResourceScope) {
		checkRefCnt(s, 5)
		checkResources(t, &s.rc, network.ScopeStat{NumConnsInbound: 2, NumFD: 1})
	})
	checkTransient(func(s *ResourceScope) {
		checkRefCnt(s, 2)
		checkResources(t, &s.rc, network.ScopeStat{NumConnsInbound: 1, NumFD: 0})
	})

	if err := conn2.SetPeer(peerA); err != nil {
		t.Fatal(err)
	}

	checkPeer(peerA, func(s *ResourceScope) {
		checkRefCnt(s, 3)
		checkResources(t, &s.rc, network.ScopeStat{NumConnsInbound: 2, NumFD: 1})
	})
	checkSystem(func(s *ResourceScope) {
		checkRefCnt(s, 5)
		checkResources(t, &s.rc, network.ScopeStat{NumConnsInbound: 2, NumFD: 1})
	})
	checkTransient(func(s *ResourceScope) {
		checkRefCnt(s, 1)
		checkResources(t, &s.rc, network.ScopeStat{})
	})

	// open a stream
	stream, err := mgr.OpenStream(peerA, network.DirInbound)
	if err != nil {
		t.Fatal(err)
	}

	checkPeer(peerA, func(s *ResourceScope) {
		checkRefCnt(s, 4)
		checkResources(t, &s.rc, network.ScopeStat{NumStreamsInbound: 1, NumConnsInbound: 2, NumFD: 1})
	})
	checkSystem(func(s *ResourceScope) {
		checkRefCnt(s, 6)
		checkResources(t, &s.rc, network.ScopeStat{NumStreamsInbound: 1, NumConnsInbound: 2, NumFD: 1})
	})
	checkTransient(func(s *ResourceScope) {
		checkRefCnt(s, 2)
		checkResources(t, &s.rc, network.ScopeStat{NumStreamsInbound: 1})
	})

	// the stream is transient we shouldn't be able to open a second one
	if _, err := mgr.OpenStream(peerA, network.DirInbound); err == nil {
		t.Fatal("expected OpenStream to fail")
	}

	// close the stream to check resource reclamation
	stream.Done()

	checkPeer(peerA, func(s *ResourceScope) {
		checkRefCnt(s, 3)
		checkResources(t, &s.rc, network.ScopeStat{NumConnsInbound: 2, NumFD: 1})
	})
	checkSystem(func(s *ResourceScope) {
		checkRefCnt(s, 5)
		checkResources(t, &s.rc, network.ScopeStat{NumConnsInbound: 2, NumFD: 1})
	})
	checkTransient(func(s *ResourceScope) {
		checkRefCnt(s, 1)
		checkResources(t, &s.rc, network.ScopeStat{})
	})

	// open another stream, but this time attach it to a protocol
	stream1, err := mgr.OpenStream(peerA, network.DirInbound)
	if err != nil {
		t.Fatal(err)
	}

	checkPeer(peerA, func(s *ResourceScope) {
		checkRefCnt(s, 4)
		checkResources(t, &s.rc, network.ScopeStat{NumStreamsInbound: 1, NumConnsInbound: 2, NumFD: 1})
	})
	checkSystem(func(s *ResourceScope) {
		checkRefCnt(s, 6)
		checkResources(t, &s.rc, network.ScopeStat{NumStreamsInbound: 1, NumConnsInbound: 2, NumFD: 1})
	})
	checkTransient(func(s *ResourceScope) {
		checkRefCnt(s, 2)
		checkResources(t, &s.rc, network.ScopeStat{NumStreamsInbound: 1})
	})

	if err := stream1.SetProtocol(protoA); err != nil {
		t.Fatal(err)
	}

	checkPeer(peerA, func(s *ResourceScope) {
		checkRefCnt(s, 4)
		checkResources(t, &s.rc, network.ScopeStat{NumStreamsInbound: 1, NumConnsInbound: 2, NumFD: 1})
	})
	checkProtocol(protoA, func(s *ResourceScope) {
		checkRefCnt(s, 2)
		checkResources(t, &s.rc, network.ScopeStat{NumStreamsInbound: 1})
	})
	checkSystem(func(s *ResourceScope) {
		checkRefCnt(s, 7)
		checkResources(t, &s.rc, network.ScopeStat{NumStreamsInbound: 1, NumConnsInbound: 2, NumFD: 1})
	})
	checkTransient(func(s *ResourceScope) {
		checkRefCnt(s, 1)
		checkResources(t, &s.rc, network.ScopeStat{})
	})

	// and now we should be able to open another stream and attach it to the protocol
	stream2, err := mgr.OpenStream(peerA, network.DirInbound)
	if err != nil {
		t.Fatal(err)
	}

	checkPeer(peerA, func(s *ResourceScope) {
		checkRefCnt(s, 5)
		checkResources(t, &s.rc, network.ScopeStat{NumStreamsInbound: 2, NumConnsInbound: 2, NumFD: 1})
	})
	checkSystem(func(s *ResourceScope) {
		checkRefCnt(s, 8)
		checkResources(t, &s.rc, network.ScopeStat{NumStreamsInbound: 2, NumConnsInbound: 2, NumFD: 1})
	})
	checkTransient(func(s *ResourceScope) {
		checkRefCnt(s, 2)
		checkResources(t, &s.rc, network.ScopeStat{NumStreamsInbound: 1})
	})

	if err := stream2.SetProtocol(protoA); err != nil {
		t.Fatal(err)
	}

	checkPeer(peerA, func(s *ResourceScope) {
		checkRefCnt(s, 5)
		checkResources(t, &s.rc, network.ScopeStat{NumStreamsInbound: 2, NumConnsInbound: 2, NumFD: 1})
	})
	checkProtocol(protoA, func(s *ResourceScope) {
		checkRefCnt(s, 3)
		checkResources(t, &s.rc, network.ScopeStat{NumStreamsInbound: 2})
	})
	checkSystem(func(s *ResourceScope) {
		checkRefCnt(s, 8)
		checkResources(t, &s.rc, network.ScopeStat{NumStreamsInbound: 2, NumConnsInbound: 2, NumFD: 1})
	})
	checkTransient(func(s *ResourceScope) {
		checkRefCnt(s, 1)
		checkResources(t, &s.rc, network.ScopeStat{})
	})

	// open a 3rd stream, and try to attach it to the same protocol
	stream3, err := mgr.OpenStream(peerB, network.DirInbound)
	if err != nil {
		t.Fatal(err)
	}

	checkPeer(peerA, func(s *ResourceScope) {
		checkRefCnt(s, 5)
		checkResources(t, &s.rc, network.ScopeStat{NumStreamsInbound: 2, NumConnsInbound: 2, NumFD: 1})
	})
	checkPeer(peerB, func(s *ResourceScope) {
		checkRefCnt(s, 2)
		checkResources(t, &s.rc, network.ScopeStat{NumStreamsInbound: 1})
	})
	checkSystem(func(s *ResourceScope) {
		checkRefCnt(s, 10)
		checkResources(t, &s.rc, network.ScopeStat{NumStreamsInbound: 3, NumConnsInbound: 2, NumFD: 1})
	})
	checkTransient(func(s *ResourceScope) {
		checkRefCnt(s, 2)
		checkResources(t, &s.rc, network.ScopeStat{NumStreamsInbound: 1})
	})

	if err := stream3.SetProtocol(protoA); err == nil {
		t.Fatal("expected SetProtocol to fail")
	}

	checkPeer(peerA, func(s *ResourceScope) {
		checkRefCnt(s, 5)
		checkResources(t, &s.rc, network.ScopeStat{NumStreamsInbound: 2, NumConnsInbound: 2, NumFD: 1})
	})
	checkPeer(peerB, func(s *ResourceScope) {
		checkRefCnt(s, 2)
		checkResources(t, &s.rc, network.ScopeStat{NumStreamsInbound: 1})
	})
	checkProtocol(protoA, func(s *ResourceScope) {
		checkRefCnt(s, 3)
		checkResources(t, &s.rc, network.ScopeStat{NumStreamsInbound: 2})
	})
	checkSystem(func(s *ResourceScope) {
		checkRefCnt(s, 10)
		checkResources(t, &s.rc, network.ScopeStat{NumStreamsInbound: 3, NumConnsInbound: 2, NumFD: 1})
	})
	checkTransient(func(s *ResourceScope) {
		checkRefCnt(s, 2)
		checkResources(t, &s.rc, network.ScopeStat{NumStreamsInbound: 1})
	})

	// but we should be able to set to another protocol
	if err := stream3.SetProtocol(protoB); err != nil {
		t.Fatal(err)
	}

	checkPeer(peerA, func(s *ResourceScope) {
		checkRefCnt(s, 5)
		checkResources(t, &s.rc, network.ScopeStat{NumStreamsInbound: 2, NumConnsInbound: 2, NumFD: 1})
	})
	checkPeer(peerB, func(s *ResourceScope) {
		checkRefCnt(s, 2)
		checkResources(t, &s.rc, network.ScopeStat{NumStreamsInbound: 1})
	})
	checkProtocol(protoA, func(s *ResourceScope) {
		checkRefCnt(s, 3)
		checkResources(t, &s.rc, network.ScopeStat{NumStreamsInbound: 2})
	})
	checkProtocol(protoB, func(s *ResourceScope) {
		checkRefCnt(s, 2)
		checkResources(t, &s.rc, network.ScopeStat{NumStreamsInbound: 1})
	})
	checkSystem(func(s *ResourceScope) {
		checkRefCnt(s, 11)
		checkResources(t, &s.rc, network.ScopeStat{NumStreamsInbound: 3, NumConnsInbound: 2, NumFD: 1})
	})
	checkTransient(func(s *ResourceScope) {
		checkRefCnt(s, 1)
		checkResources(t, &s.rc, network.ScopeStat{})
	})

	// we should be able to attach stream1 and stream2 to svcA, but stream3 should fail due to limit
	if err := stream1.SetService(svcA); err != nil {
		t.Fatal(err)
	}

	checkPeer(peerA, func(s *ResourceScope) {
		checkRefCnt(s, 5)
		checkResources(t, &s.rc, network.ScopeStat{NumStreamsInbound: 2, NumConnsInbound: 2, NumFD: 1})
	})
	checkPeer(peerB, func(s *ResourceScope) {
		checkRefCnt(s, 2)
		checkResources(t, &s.rc, network.ScopeStat{NumStreamsInbound: 1})
	})
	checkService(svcA, func(s *ResourceScope) {
		checkRefCnt(s, 2)
		checkResources(t, &s.rc, network.ScopeStat{NumStreamsInbound: 1})
	})
	checkProtocol(protoA, func(s *ResourceScope) {
		checkRefCnt(s, 3)
		checkResources(t, &s.rc, network.ScopeStat{NumStreamsInbound: 2})
	})
	checkProtocol(protoB, func(s *ResourceScope) {
		checkRefCnt(s, 2)
		checkResources(t, &s.rc, network.ScopeStat{NumStreamsInbound: 1})
	})
	checkSystem(func(s *ResourceScope) {
		checkRefCnt(s, 12)
		checkResources(t, &s.rc, network.ScopeStat{NumStreamsInbound: 3, NumConnsInbound: 2, NumFD: 1})
	})
	checkTransient(func(s *ResourceScope) {
		checkRefCnt(s, 1)
		checkResources(t, &s.rc, network.ScopeStat{})
	})

	if err := stream2.SetService(svcA); err != nil {
		t.Fatal(err)
	}

	checkPeer(peerA, func(s *ResourceScope) {
		checkRefCnt(s, 5)
		checkResources(t, &s.rc, network.ScopeStat{NumStreamsInbound: 2, NumConnsInbound: 2, NumFD: 1})
	})
	checkPeer(peerB, func(s *ResourceScope) {
		checkRefCnt(s, 2)
		checkResources(t, &s.rc, network.ScopeStat{NumStreamsInbound: 1})
	})
	checkService(svcA, func(s *ResourceScope) {
		checkRefCnt(s, 3)
		checkResources(t, &s.rc, network.ScopeStat{NumStreamsInbound: 2})
	})
	checkProtocol(protoA, func(s *ResourceScope) {
		checkRefCnt(s, 3)
		checkResources(t, &s.rc, network.ScopeStat{NumStreamsInbound: 2})
	})
	checkProtocol(protoB, func(s *ResourceScope) {
		checkRefCnt(s, 2)
		checkResources(t, &s.rc, network.ScopeStat{NumStreamsInbound: 1})
	})
	checkSystem(func(s *ResourceScope) {
		checkRefCnt(s, 12)
		checkResources(t, &s.rc, network.ScopeStat{NumStreamsInbound: 3, NumConnsInbound: 2, NumFD: 1})
	})
	checkTransient(func(s *ResourceScope) {
		checkRefCnt(s, 1)
		checkResources(t, &s.rc, network.ScopeStat{})
	})

	if err := stream3.SetService(svcA); err == nil {
		t.Fatal("expected SetService to fail")
	}

	checkPeer(peerA, func(s *ResourceScope) {
		checkRefCnt(s, 5)
		checkResources(t, &s.rc, network.ScopeStat{NumStreamsInbound: 2, NumConnsInbound: 2, NumFD: 1})
	})
	checkPeer(peerB, func(s *ResourceScope) {
		checkRefCnt(s, 2)
		checkResources(t, &s.rc, network.ScopeStat{NumStreamsInbound: 1})
	})
	checkService(svcA, func(s *ResourceScope) {
		checkRefCnt(s, 3)
		checkResources(t, &s.rc, network.ScopeStat{NumStreamsInbound: 2})
	})
	checkProtocol(protoA, func(s *ResourceScope) {
		checkRefCnt(s, 3)
		checkResources(t, &s.rc, network.ScopeStat{NumStreamsInbound: 2})
	})
	checkProtocol(protoB, func(s *ResourceScope) {
		checkRefCnt(s, 2)
		checkResources(t, &s.rc, network.ScopeStat{NumStreamsInbound: 1})
	})
	checkSystem(func(s *ResourceScope) {
		checkRefCnt(s, 12)
		checkResources(t, &s.rc, network.ScopeStat{NumStreamsInbound: 3, NumConnsInbound: 2, NumFD: 1})
	})
	checkTransient(func(s *ResourceScope) {
		checkRefCnt(s, 1)
		checkResources(t, &s.rc, network.ScopeStat{})
	})

	// and now let's reclaim our resources to make sure we can gc unused peer and proto scopes
	// but first check internal refs
	mgr.mx.Lock()
	_, okProtoA := mgr.proto[protoA]
	_, okProtoB := mgr.proto[protoB]
	_, okPeerA := mgr.peer[peerA]
	_, okPeerB := mgr.peer[peerB]
	mgr.mx.Unlock()

	if !okProtoA {
		t.Fatal("protocol scope is not stored")
	}
	if !okProtoB {
		t.Fatal("protocol scope is not stored")
	}
	if !okPeerA {
		t.Fatal("peer scope is not stored")
	}
	if !okPeerB {
		t.Fatal("peer scope is not stored")
	}

	// ok, reclaim
	stream1.Done()
	stream2.Done()
	stream3.Done()
	conn1.Done()
	conn2.Done()

	// check everything released
	checkPeer(peerA, func(s *ResourceScope) {
		checkRefCnt(s, 1)
		checkResources(t, &s.rc, network.ScopeStat{})
	})
	checkPeer(peerB, func(s *ResourceScope) {
		checkRefCnt(s, 1)
		checkResources(t, &s.rc, network.ScopeStat{})
	})
	checkService(svcA, func(s *ResourceScope) {
		checkRefCnt(s, 1)
		checkResources(t, &s.rc, network.ScopeStat{})
	})
	checkProtocol(protoA, func(s *ResourceScope) {
		checkRefCnt(s, 1)
		checkResources(t, &s.rc, network.ScopeStat{})
	})
	checkProtocol(protoB, func(s *ResourceScope) {
		checkRefCnt(s, 1)
		checkResources(t, &s.rc, network.ScopeStat{})
	})
	checkSystem(func(s *ResourceScope) {
		checkRefCnt(s, 7)
		checkResources(t, &s.rc, network.ScopeStat{})
	})
	checkTransient(func(s *ResourceScope) {
		checkRefCnt(s, 1)
		checkResources(t, &s.rc, network.ScopeStat{})
	})

	mgr.gc()

	checkSystem(func(s *ResourceScope) {
		checkRefCnt(s, 3)
		checkResources(t, &s.rc, network.ScopeStat{})
	})
	checkTransient(func(s *ResourceScope) {
		checkRefCnt(s, 1)
		checkResources(t, &s.rc, network.ScopeStat{})
	})

	mgr.mx.Lock()
	lenProto := len(mgr.proto)
	lenPeer := len(mgr.peer)
	mgr.mx.Unlock()

	if lenProto != 0 {
		t.Fatal("protocols were not gc'ed")
	}
	if lenPeer != 0 {
		t.Fatal("perrs were not gc'ed")
	}

}