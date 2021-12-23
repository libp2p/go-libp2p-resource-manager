package rcmgr

import (
	"fmt"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
)

type ResourceManager struct {
	system    *SystemScope
	transient *TransientScope
}

type SystemScope struct {
	*ResourceScope
}

var _ network.ResourceScope = (*SystemScope)(nil)

type TransientScope struct {
	*ResourceScope

	system *SystemScope
}

var _ network.ResourceScope = (*TransientScope)(nil)

type ServiceScope struct {
	*ResourceScope

	name   string
	system *SystemScope
}

var _ network.ServiceScope = (*ServiceScope)(nil)

type ProtocolScope struct {
	*ResourceScope

	proto  protocol.ID
	system *SystemScope
}

var _ network.ProtocolScope = (*ProtocolScope)(nil)

type PeerScope struct {
	*ResourceScope

	peer      peer.ID
	rcmgr     *ResourceManager
	system    *SystemScope
	transient *TransientScope
}

var _ network.PeerScope = (*PeerScope)(nil)

type ConnectionScope struct {
	*ResourceScope

	dir       network.Direction
	rcmgr     *ResourceManager
	system    *SystemScope
	transient *TransientScope
	peer      *PeerScope
}

var _ network.ConnectionScope = (*ConnectionScope)(nil)

type StreamScope struct {
	*ResourceScope

	dir       network.Direction
	rcmgr     *ResourceManager
	system    *SystemScope
	transient *TransientScope
	peer      *PeerScope
	svc       *ServiceScope
	proto     *ProtocolScope
}

var _ network.StreamScope = (*StreamScope)(nil)

func (r *ResourceManager) getProtocolScope(proto protocol.ID) *ProtocolScope {
	// TODO
	return nil
}

func (r *ResourceManager) getServiceScope(svc string) *ServiceScope {
	// TODO
	return nil
}

func (r *ResourceManager) getPeerScope(p peer.ID) *PeerScope {
	// TODO
	return nil
}

func (r *ResourceManager) getConnLimit() Limit {
	// TODO
	return nil
}

func (r *ResourceManager) getStreamLimit(p peer.ID) Limit {
	// TODO
	return nil
}

func (r *ResourceManager) OpenConnection(dir network.Direction, usefd bool) (network.ConnectionScope, error) {
	conn := NewConnectionScope(dir, r.getConnLimit(), r)

	if err := conn.AddConn(dir); err != nil {
		return nil, err
	}

	if err := conn.AddFD(1); err != nil {
		conn.RemoveConn(dir)
		return nil, err
	}

	return conn, nil
}

func NewSystemScope(limit Limit) *SystemScope {
	return &SystemScope{
		ResourceScope: NewResourceScope(limit, nil),
	}
}

func NewTransientScope(limit Limit, system *SystemScope) *TransientScope {
	return &TransientScope{
		ResourceScope: NewResourceScope(limit, []*ResourceScope{system.ResourceScope}),
		system:        system,
	}
}

func NewServiceScope(name string, limit Limit, system *SystemScope) *ServiceScope {
	return &ServiceScope{
		ResourceScope: NewResourceScope(limit, []*ResourceScope{system.ResourceScope}),
		name:          name,
		system:        system,
	}
}

func NewProtocolScope(proto protocol.ID, limit Limit, system *SystemScope) *ProtocolScope {
	return &ProtocolScope{
		ResourceScope: NewResourceScope(limit, []*ResourceScope{system.ResourceScope}),
		proto:         proto,
		system:        system,
	}
}

func NewPeerScope(p peer.ID, limit Limit, rcmgr *ResourceManager) *PeerScope {
	return &PeerScope{
		ResourceScope: NewResourceScope(limit, []*ResourceScope{rcmgr.system.ResourceScope}),
		peer:          p,
		rcmgr:         rcmgr,
		system:        rcmgr.system,
		transient:     rcmgr.transient,
	}
}

func NewConnectionScope(dir network.Direction, limit Limit, rcmgr *ResourceManager) *ConnectionScope {
	return &ConnectionScope{
		ResourceScope: NewResourceScope(limit, []*ResourceScope{rcmgr.transient.ResourceScope, rcmgr.system.ResourceScope}),
		dir:           dir,
		rcmgr:         rcmgr,
		system:        rcmgr.system,
		transient:     rcmgr.transient,
	}
}

func NewStreamScope(dir network.Direction, limit Limit, peer *PeerScope) *StreamScope {
	return &StreamScope{
		ResourceScope: NewResourceScope(limit, []*ResourceScope{peer.ResourceScope, peer.transient.ResourceScope, peer.system.ResourceScope}),
		dir:           dir,
		rcmgr:         peer.rcmgr,
		system:        peer.system,
		transient:     peer.transient,
		peer:          peer,
	}
}

func (s *ServiceScope) Name() string {
	return s.name
}

func (s *ProtocolScope) Protocol() protocol.ID {
	return s.proto
}

func (s *PeerScope) Peer() peer.ID {
	return s.peer
}

func (s *PeerScope) OpenStream(dir network.Direction) (network.StreamScope, error) {
	stream := NewStreamScope(dir, s.rcmgr.getStreamLimit(s.peer), s)
	err := stream.AddStream(dir)
	if err != nil {
		return nil, err
	}

	return stream, nil
}

func (s *ConnectionScope) PeerScope() network.PeerScope {
	s.Lock()
	defer s.Unlock()
	return s.peer
}

func (s *ConnectionScope) SetPeer(p peer.ID) error {
	s.Lock()
	defer s.Unlock()

	if s.peer != nil {
		return fmt.Errorf("connection scope already attached to a peer")
	}
	s.peer = s.rcmgr.getPeerScope(p)

	// juggle resources from transient scope to peer scope
	mem := s.ResourceScope.rc.memory

	var incount, outcount int
	if s.dir == network.DirInbound {
		incount = 1
	} else {
		outcount = 1
	}

	if err := s.peer.ReserveMemoryForChild(mem); err != nil {
		return err
	}
	if err := s.peer.AddConnForChild(incount, outcount); err != nil {
		s.peer.ReleaseMemoryForChild(mem)
		return err
	}
	if err := s.peer.AddFDForChild(1); err != nil {
		s.peer.ReleaseMemoryForChild(mem)
		s.peer.RemoveConnForChild(incount, outcount)
		return err
	}

	s.transient.ReleaseMemoryForChild(mem)
	s.transient.RemoveConnForChild(incount, outcount)
	s.transient.RemoveFDForChild(1)

	// update constraints
	constraints := []*ResourceScope{
		s.peer.ResourceScope,
		s.system.ResourceScope,
	}
	s.ResourceScope.constraints = constraints

	return nil
}

func (s *StreamScope) ProtocolScope() network.ProtocolScope {
	s.Lock()
	defer s.Unlock()
	return s.proto
}

func (s *StreamScope) SetProtocol(proto protocol.ID) error {
	s.Lock()
	defer s.Unlock()

	if s.proto != nil {
		return fmt.Errorf("stream scope already attached to a protocol")
	}

	s.proto = s.rcmgr.getProtocolScope(proto)

	// juggle resources from transient scope to protocol scope
	mem := s.ResourceScope.rc.memory

	var incount, outcount int
	if s.dir == network.DirInbound {
		incount = 1
	} else {
		outcount = 1
	}

	if err := s.proto.ReserveMemoryForChild(mem); err != nil {
		return err
	}
	if err := s.proto.AddStreamForChild(incount, outcount); err != nil {
		s.proto.ReleaseMemoryForChild(mem)
		return err
	}

	s.transient.ReleaseMemoryForChild(mem)
	s.transient.RemoveStreamForChild(incount, outcount)

	// update constraints
	constraints := []*ResourceScope{
		s.peer.ResourceScope,
		s.proto.ResourceScope,
	}
	if s.svc != nil {
		constraints = append(constraints, s.svc.ResourceScope)
	}
	constraints = append(constraints, s.system.ResourceScope)
	s.ResourceScope.constraints = constraints

	return nil
}

func (s *StreamScope) ServiceScope() network.ServiceScope {
	s.Lock()
	defer s.Unlock()
	return s.svc
}

func (s *StreamScope) SetService(svc string) error {
	s.Lock()
	defer s.Unlock()

	if s.proto == nil {
		return fmt.Errorf("stream scope not attached to a protocol")
	}
	if s.svc != nil {
		return fmt.Errorf("stream scope already attached to a service")
	}

	s.svc = s.rcmgr.getServiceScope(svc)

	// reserve resources in service
	mem := s.ResourceScope.rc.memory

	var incount, outcount int
	if s.dir == network.DirInbound {
		incount = 1
	} else {
		outcount = 1
	}

	if err := s.svc.ReserveMemoryForChild(mem); err != nil {
		return err
	}
	if err := s.svc.AddStreamForChild(incount, outcount); err != nil {
		s.svc.ReleaseMemoryForChild(mem)
		return err
	}

	// update constraints
	constraints := []*ResourceScope{
		s.peer.ResourceScope,
		s.proto.ResourceScope,
		s.svc.ResourceScope,
		s.system.ResourceScope,
	}
	s.ResourceScope.constraints = constraints

	return nil
}

func (s *StreamScope) PeerScope() network.PeerScope {
	s.Lock()
	defer s.Unlock()
	return s.peer
}
