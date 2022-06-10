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
	allowedIPs      []net.IP
	allowedNetworks []*net.IPNet

	// Only the specified peers can use these IPs
	allowedPeerByIP      map[peer.ID][]net.IP
	allowedPeerByNetwork map[peer.ID][]*net.IPNet
}

func newAllowList() allowlist {
	return allowlist{
		allowedPeerByIP:      make(map[peer.ID][]net.IP),
		allowedPeerByNetwork: make(map[peer.ID][]*net.IPNet),
	}
}

func toIPOrIPNet(ma multiaddr.Multiaddr) (net.IP, *net.IPNet, string, error) {
	var ipString string
	var mask string
	var allowedPeer string

	multiaddr.ForEach(ma, func(c multiaddr.Component) bool {
		if c.Protocol().Code == multiaddr.P_IP4 || c.Protocol().Code == multiaddr.P_IP6 {
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
		return nil, nil, allowedPeer, errors.New("missing ip address")
	}

	if mask == "" {
		ip := net.ParseIP(ipString)
		if ip == nil {
			return nil, nil, allowedPeer, errors.New("invalid ip address")
		}
		return ip, nil, allowedPeer, nil
	}

	_, ipnet, err := net.ParseCIDR(ipString + "/" + mask)
	return nil, ipnet, allowedPeer, err

}

func (al *allowlist) Add(ma multiaddr.Multiaddr) error {
	ip, ipnet, allowedPeerStr, err := toIPOrIPNet(ma)
	if err != nil {
		return err
	}

	if allowedPeerStr != "" {
		// We have a peerID constraint
		allowedPeer, err := peer.Decode(allowedPeerStr)
		if err != nil {
			return err
		}
		if ip != nil {
			al.allowedPeerByIP[allowedPeer] = append(al.allowedPeerByIP[allowedPeer], ip)
		} else if ipnet != nil {
			al.allowedPeerByNetwork[allowedPeer] = append(al.allowedPeerByNetwork[allowedPeer], ipnet)
		}
	} else {
		if ip != nil {
			al.allowedIPs = append(al.allowedIPs, ip)
		} else if ipnet != nil {
			al.allowedNetworks = append(al.allowedNetworks, ipnet)
		}
	}
	return nil
}

func (al *allowlist) Remove(ma multiaddr.Multiaddr) error {
	ip, ipnet, allowedPeerStr, err := toIPOrIPNet(ma)
	if err != nil {
		return err
	}
	ipList := al.allowedIPs
	ipNetList := al.allowedNetworks

	var allowedPeer peer.ID
	if allowedPeerStr != "" {
		// We have a peerID constraint
		allowedPeer, err = peer.Decode(allowedPeerStr)
		if err != nil {
			return err
		}
		ipList = al.allowedPeerByIP[allowedPeer]
		ipNetList = al.allowedPeerByNetwork[allowedPeer]
	}

	if ip != nil {
		i := len(ipList)
		for i > 0 {
			i--
			if ipList[i].Equal(ip) {
				if i == len(ipList)-1 {
					// Trim this element from the end
					ipList = ipList[:i]
				} else {
					// swap remove
					ipList[i] = ipList[len(ipList)-1]
					ipList = ipList[:len(ipList)-1]
				}
			}
		}
	} else if ipnet != nil {
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
		al.allowedPeerByIP[allowedPeer] = ipList
		al.allowedPeerByNetwork[allowedPeer] = ipNetList
	} else {
		al.allowedIPs = ipList
		al.allowedNetworks = ipNetList
	}

	return nil
}

func (al *allowlist) Allowed(ma multiaddr.Multiaddr) bool {
	ip, err := manet.ToIP(ma)
	if err != nil {
		return false
	}

	for _, allowedIP := range al.allowedIPs {
		if allowedIP.Equal(ip) {
			return true
		}
	}

	for _, network := range al.allowedNetworks {
		if network.Contains(ip) {
			return true
		}
	}

	for _, allowedIPs := range al.allowedPeerByIP {
		for _, allowedIP := range allowedIPs {
			if allowedIP.Equal(ip) {
				return true
			}
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

	for _, allowedIP := range al.allowedIPs {
		if allowedIP.Equal(ip) {
			// We found a match that isn't constrained by a peerID
			return true
		}
	}

	for _, network := range al.allowedNetworks {
		if network.Contains(ip) {
			// We found a match that isn't constrained by a peerID
			return true
		}
	}

	if expectedIPs, ok := al.allowedPeerByIP[peerID]; ok {
		for _, expectedIP := range expectedIPs {
			if expectedIP.Equal(ip) {
				return true
			}
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
