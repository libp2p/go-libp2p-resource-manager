package rcmgr

import (
	"fmt"
	"testing"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/test"
	"github.com/multiformats/go-multiaddr"
)

func TestAllowed(t *testing.T) {
	allowlist := newAllowList()
	ma, _ := multiaddr.NewMultiaddr("/ip4/1.2.3.4/tcp/1234")
	err := allowlist.Add(ma)
	if err != nil {
		t.Fatalf("failed to add ip4: %s", err)
	}

	if !allowlist.Allowed(ma) {
		t.Fatalf("addr should be allowed")
	}
}

func TestAllowedNetwork(t *testing.T) {
	allowlist := newAllowList()
	ma, _ := multiaddr.NewMultiaddr("/ip4/1.2.3.0/ipcidr/24")
	err := allowlist.Add(ma)
	if err != nil {
		t.Fatalf("failed to add ip4: %s", err)
	}

	ma2, _ := multiaddr.NewMultiaddr("/ip4/1.2.3.20/tcp/1234")
	if !allowlist.Allowed(ma2) {
		t.Fatalf("addr should be allowed")
	}
}

func TestAllowedPeerOnIP(t *testing.T) {
	allowlist := newAllowList()
	p, err := test.RandPeerID()
	if err != nil {
		t.Fatalf("failed to gen peer ip4: %s", err)
	}

	ma, _ := multiaddr.NewMultiaddr("/ip4/1.2.3.4/p2p/" + peer.Encode(p))
	err = allowlist.Add(ma)
	if err != nil {
		t.Fatalf("failed to add ip4: %s", err)
	}

	ma2, _ := multiaddr.NewMultiaddr("/ip4/1.2.3.4")
	if !allowlist.AllowedPeerAndMultiaddr(p, ma2) {
		t.Fatalf("addr should be allowed")
	}
}

func TestAllowedPeerOnNetwork(t *testing.T) {
	allowlist := newAllowList()
	p, err := test.RandPeerID()
	if err != nil {
		t.Fatalf("failed to gen peer ip4: %s", err)
	}

	ma, _ := multiaddr.NewMultiaddr("/ip4/1.2.3.0/ipcidr/24/p2p/" + peer.Encode(p))
	err = allowlist.Add(ma)
	if err != nil {
		t.Fatalf("failed to add ip4: %s", err)
	}

	ma2, _ := multiaddr.NewMultiaddr("/ip4/1.2.3.4")
	if !allowlist.AllowedPeerAndMultiaddr(p, ma2) {
		t.Fatalf("addr should be allowed")
	}
}

func TestAllowedWithPeer(t *testing.T) {
	type testcase struct {
		name      string
		allowlist []string
		endpoint  multiaddr.Multiaddr
		peer      peer.ID
		isAllowed bool
	}

	peerA := test.RandPeerIDFatal(t)
	peerB := test.RandPeerIDFatal(t)
	multiaddrA, _ := multiaddr.NewMultiaddr("/ip4/1.2.3.4/tcp/1234")
	multiaddrB, _ := multiaddr.NewMultiaddr("/ip4/2.2.3.4/tcp/1234")

	testcases := []testcase{
		{
			name:      "Blocked",
			isAllowed: false,
			allowlist: []string{"/ip4/1.2.3.1"},
			endpoint:  multiaddrA,
			peer:      peerA,
		},
		{
			name:      "Blocked wrong peer",
			isAllowed: false,
			allowlist: []string{"/ip4/1.2.3.4" + "/p2p/" + peer.Encode(peerB)},
			endpoint:  multiaddrA,
			peer:      peerA,
		},
		{
			name:      "allowed on network",
			isAllowed: true,
			allowlist: []string{"/ip4/1.2.3.0/ipcidr/24"},
			endpoint:  multiaddrA,
			peer:      peerA,
		},
		{
			name:      "Blocked peer not on network",
			isAllowed: true,
			allowlist: []string{"/ip4/1.2.3.0/ipcidr/24"},
			endpoint:  multiaddrA,
			peer:      peerA,
		}, {
			name:      "allowed. right network, right peer",
			isAllowed: true,
			allowlist: []string{"/ip4/1.2.3.0/ipcidr/24" + "/p2p/" + peer.Encode(peerA)},
			endpoint:  multiaddrA,
			peer:      peerA,
		}, {
			name:      "allowed. right network, no peer",
			isAllowed: true,
			allowlist: []string{"/ip4/1.2.3.0/ipcidr/24"},
			endpoint:  multiaddrA,
			peer:      peerA,
		},
		{
			name:      "Blocked. right network, wrong peer",
			isAllowed: false,
			allowlist: []string{"/ip4/1.2.3.0/ipcidr/24" + "/p2p/" + peer.Encode(peerB)},
			endpoint:  multiaddrA,
			peer:      peerA,
		},
		{
			name:      "allowed peer any ip",
			isAllowed: true,
			allowlist: []string{"/ip4/0.0.0.0/ipcidr/0"},
			endpoint:  multiaddrA,
			peer:      peerA,
		},
		{
			name:      "allowed peer multiple ips in allowlist",
			isAllowed: true,
			allowlist: []string{"/ip4/1.2.3.4/p2p/" + peer.Encode(peerA), "/ip4/2.2.3.4/p2p/" + peer.Encode(peerA)},
			endpoint:  multiaddrA,
			peer:      peerA,
		},
		{
			name:      "allowed peer multiple ips in allowlist",
			isAllowed: true,
			allowlist: []string{"/ip4/1.2.3.4/p2p/" + peer.Encode(peerA), "/ip4/2.2.3.4/p2p/" + peer.Encode(peerA)},
			endpoint:  multiaddrB,
			peer:      peerA,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			allowlist := newAllowList()
			for _, maStr := range tc.allowlist {
				ma, err := multiaddr.NewMultiaddr(maStr)
				if err != nil {
					fmt.Printf("failed to parse multiaddr: %s", err)
				}
				allowlist.Add(ma)
			}

			if allowlist.AllowedPeerAndMultiaddr(tc.peer, tc.endpoint) != tc.isAllowed {
				t.Fatalf("%v: expected %v", !tc.isAllowed, tc.isAllowed)
			}
		})
	}

}

func TestRemoved(t *testing.T) {
	type testCase struct {
		name      string
		allowedMA string
	}
	peerA := test.RandPeerIDFatal(t)
	maA, _ := multiaddr.NewMultiaddr("/ip4/1.2.3.4")

	testCases := []testCase{
		{name: "ip4", allowedMA: "/ip4/1.2.3.4"},
		{name: "ip4 with peer", allowedMA: "/ip4/1.2.3.4/p2p/" + peer.Encode(peerA)},
		{name: "ip4 network", allowedMA: "/ip4/0.0.0.0/ipcidr/0"},
		{name: "ip4 network with peer", allowedMA: "/ip4/0.0.0.0/ipcidr/0/p2p/" + peer.Encode(peerA)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			allowlist := newAllowList()
			ma, err := multiaddr.NewMultiaddr(tc.allowedMA)
			if err != nil {
				t.Fatalf("failed to parse ma: %s", err)
			}

			err = allowlist.Add(ma)
			if err != nil {
				t.Fatalf("failed to add ip4: %s", err)
			}

			if !allowlist.AllowedPeerAndMultiaddr(peerA, maA) {
				t.Fatalf("addr should be allowed")
			}

			allowlist.Remove((ma))

			if allowlist.AllowedPeerAndMultiaddr(peerA, maA) {
				t.Fatalf("addr should not be allowed")
			}
		})
	}
}
