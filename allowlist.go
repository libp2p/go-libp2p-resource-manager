package rcmgr

import (
	"bytes"
	"errors"
	"net"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
)

type allowlist struct {
	// TODO do we want to make this lookup faster?

	// Any peer with these IPs are allowed
	allowedNetworks []*net.IPNet

	// Only the specified peers can use these IPs
	allowedPeerByNetwork map[peer.ID][]*net.IPNet
}

func newAllowList() allowlist {
	return allowlist{
		allowedPeerByNetwork: make(map[peer.ID][]*net.IPNet),
	}
}

func toIPNet(ma multiaddr.Multiaddr) (*net.IPNet, string, error) {
	var ipString string
	var mask string
	var allowedPeer string
	var isIPV4 bool

	multiaddr.ForEach(ma, func(c multiaddr.Component) bool {
		if c.Protocol().Code == multiaddr.P_IP4 || c.Protocol().Code == multiaddr.P_IP6 {
			isIPV4 = c.Protocol().Code == multiaddr.P_IP4
			ipString = c.Value()
		}
		if c.Protocol().Code == multiaddr.P_IPCIDR {
			mask = c.Value()
		}
		if c.Protocol().Code == multiaddr.P_P2P {
			allowedPeer = c.Value()
		}
		return ipString == "" || mask == "" || allowedPeer == ""
	})

	if ipString == "" {
		return nil, allowedPeer, errors.New("missing ip address")
	}

	if mask == "" {
		ip := net.ParseIP(ipString)
		if ip == nil {
			return nil, allowedPeer, errors.New("invalid ip address")
		}
		var mask net.IPMask
		if isIPV4 {
			mask = net.CIDRMask(32, 32)
		} else {
			mask = net.CIDRMask(128, 128)
		}

		net := &net.IPNet{IP: ip, Mask: mask}
		return net, allowedPeer, nil
	}

	_, ipnet, err := net.ParseCIDR(ipString + "/" + mask)
	return ipnet, allowedPeer, err

}

func (al *allowlist) Add(ma multiaddr.Multiaddr) error {
	ipnet, allowedPeerStr, err := toIPNet(ma)
	if err != nil {
		return err
	}

	if allowedPeerStr != "" {
		// We have a peerID constraint
		allowedPeer, err := peer.Decode(allowedPeerStr)
		if err != nil {
			return err
		}
		if ipnet != nil {
			al.allowedPeerByNetwork[allowedPeer] = append(al.allowedPeerByNetwork[allowedPeer], ipnet)
		}
	} else {
		if ipnet != nil {
			al.allowedNetworks = append(al.allowedNetworks, ipnet)
		}
	}
	return nil
}

func (al *allowlist) Remove(ma multiaddr.Multiaddr) error {
	ipnet, allowedPeerStr, err := toIPNet(ma)
	if err != nil {
		return err
	}
	ipNetList := al.allowedNetworks

	var allowedPeer peer.ID
	if allowedPeerStr != "" {
		// We have a peerID constraint
		allowedPeer, err = peer.Decode(allowedPeerStr)
		if err != nil {
			return err
		}
		ipNetList = al.allowedPeerByNetwork[allowedPeer]
	}

	if ipnet != nil {
		i := len(ipNetList)
		for i > 0 {
			i--
			if ipNetList[i].IP.Equal(ipnet.IP) && bytes.Equal(ipNetList[i].Mask, ipnet.Mask) {
				if i == len(ipNetList)-1 {
					// Trim this element from the end
					ipNetList = ipNetList[:i]
				} else {
					// swap remove
					ipNetList[i] = ipNetList[len(ipNetList)-1]
					ipNetList = ipNetList[:len(ipNetList)-1]
				}
			}
		}
	}

	if allowedPeer != "" {
		al.allowedPeerByNetwork[allowedPeer] = ipNetList
	} else {
		al.allowedNetworks = ipNetList
	}

	return nil
}

func (al *allowlist) Allowed(ma multiaddr.Multiaddr) bool {
	ip, err := manet.ToIP(ma)
	if err != nil {
		return false
	}

	_ = ip
	for _, network := range al.allowedNetworks {
		_ = network
		if network.Contains(ip) {
			return true
		}
	}

	for _, allowedNetworks := range al.allowedPeerByNetwork {
		for _, network := range allowedNetworks {
			if network.Contains(ip) {
				return true
			}
		}
	}

	return false
}

func (al *allowlist) AllowedPeerAndMultiaddr(peerID peer.ID, ma multiaddr.Multiaddr) bool {
	ip, err := manet.ToIP(ma)
	if err != nil {
		return false
	}

	for _, network := range al.allowedNetworks {
		if network.Contains(ip) {
			// We found a match that isn't constrained by a peerID
			return true
		}
	}

	if expectedNetworks, ok := al.allowedPeerByNetwork[peerID]; ok {
		for _, expectedNetwork := range expectedNetworks {
			if expectedNetwork.Contains(ip) {
				return true
			}
		}
	}

	return false
}
