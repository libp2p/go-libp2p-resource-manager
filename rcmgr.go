package rcmgr

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
)

type ResourceManager struct {
	limits Limiter

	system    *SystemScope
	transient *TransientScope

	mx    sync.Mutex
	svc   map[string]*ServiceScope
	proto map[protocol.ID]*ProtocolScope
	peer  map[peer.ID]*PeerScope

	cancelCtx context.Context
	cancel    func()
	wg        sync.WaitGroup
}

var _ network.ResourceManager = (*ResourceManager)(nil)

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

	peer  peer.ID
	rcmgr *ResourceManager
}

var _ network.PeerScope = (*PeerScope)(nil)

type ConnectionScope struct {
	*ResourceScope

	dir   network.Direction
	usefd bool
	rcmgr *ResourceManager
	peer  *PeerScope
}

var _ network.ConnectionScope = (*ConnectionScope)(nil)
var _ network.ConnectionManagementScope = (*ConnectionScope)(nil)

type StreamScope struct {
	*ResourceScope

	dir   network.Direction
	rcmgr *ResourceManager
	peer  *PeerScope
	svc   *ServiceScope
	proto *ProtocolScope
}

var _ network.StreamScope = (*StreamScope)(nil)
var _ network.StreamManagementScope = (*StreamScope)(nil)

func NewResourceManager(limits Limiter) *ResourceManager {
	r := &ResourceManager{
		limits: limits,
		svc:    make(map[string]*ServiceScope),
		proto:  make(map[protocol.ID]*ProtocolScope),
		peer:   make(map[peer.ID]*PeerScope),
	}

	r.system = NewSystemScope(limits.GetSystemLimits())
	r.system.IncRef()
	r.transient = NewTransientScope(limits.GetTransientLimits(), r.system)
	r.transient.IncRef()

	r.cancelCtx, r.cancel = context.WithCancel(context.Background())

	r.wg.Add(1)
	go r.background()

	return r
}

func (r *ResourceManager) ViewSystem(f func(network.ResourceScope) error) error {
	return f(r.system)
}

func (r *ResourceManager) ViewTransient(f func(network.ResourceScope) error) error {
	return f(r.transient)
}

func (r *ResourceManager) ViewService(srv string, f func(network.ServiceScope) error) error {
	s := r.getServiceScope(srv)
	defer s.DecRef()

	return f(s)
}

func (r *ResourceManager) ViewProtocol(proto protocol.ID, f func(network.ProtocolScope) error) error {
	s := r.getProtocolScope(proto)
	defer s.DecRef()

	return f(s)
}

func (r *ResourceManager) ViewPeer(p peer.ID, f func(network.PeerScope) error) error {
	s := r.getPeerScope(p)
	defer s.DecRef()

	return f(s)
}

func (r *ResourceManager) getServiceScope(svc string) *ServiceScope {
	r.mx.Lock()
	defer r.mx.Unlock()

	s, ok := r.svc[svc]
	if !ok {
		s = NewServiceScope(svc, r.limits.GetServiceLimits(svc), r.system)
		r.svc[svc] = s
	}

	s.IncRef()
	return s
}

func (r *ResourceManager) getProtocolScope(proto protocol.ID) *ProtocolScope {
	r.mx.Lock()
	defer r.mx.Unlock()

	s, ok := r.proto[proto]
	if !ok {
		s = NewProtocolScope(proto, r.limits.GetProtocolLimits(proto), r.system)
		r.proto[proto] = s
	}

	s.IncRef()
	return s
}

func (r *ResourceManager) getPeerScope(p peer.ID) *PeerScope {
	r.mx.Lock()
	defer r.mx.Unlock()

	s, ok := r.peer[p]
	if !ok {
		s = NewPeerScope(p, r.limits.GetPeerLimits(p), r)
		r.peer[p] = s
	}

	s.IncRef()
	return s
}

func (r *ResourceManager) OpenConnection(dir network.Direction, usefd bool) (network.ConnectionManagementScope, error) {
	conn := NewConnectionScope(dir, usefd, r.limits.GetConnLimits(), r)

	if err := conn.AddConn(dir); err != nil {
		conn.Done()
		return nil, err
	}

	if usefd {
		if err := conn.AddFD(1); err != nil {
			conn.Done()
			return nil, err
		}
	}

	return conn, nil
}

func (r *ResourceManager) OpenStream(p peer.ID, dir network.Direction) (network.StreamManagementScope, error) {
	peer := r.getPeerScope(p)
	stream := NewStreamScope(dir, r.limits.GetStreamLimits(p), peer)
	peer.DecRef() // we have the reference in constraints

	err := stream.AddStream(dir)
	if err != nil {
		stream.Done()
		return nil, err
	}

	return stream, nil
}

func (r *ResourceManager) Close() error {
	r.cancel()
	r.wg.Wait()

	return nil
}

func (r *ResourceManager) background() {
	defer r.wg.Done()

	// periodically garbage collects unused peer and protocol scopes
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.gc()
		case <-r.cancelCtx.Done():
			return
		}
	}
}

func (r *ResourceManager) gc() {
	r.mx.Lock()
	defer r.mx.Unlock()

	for proto, s := range r.proto {
		if s.IsUnused() {
			s.Done()
			delete(r.proto, proto)
		}
	}

	for p, s := range r.peer {
		if s.IsUnused() {
			s.Done()
			delete(r.peer, p)
		}
	}
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
	}
}

func NewConnectionScope(dir network.Direction, usefd bool, limit Limit, rcmgr *ResourceManager) *ConnectionScope {
	return &ConnectionScope{
		ResourceScope: NewResourceScope(limit, []*ResourceScope{rcmgr.transient.ResourceScope, rcmgr.system.ResourceScope}),
		dir:           dir,
		usefd:         usefd,
		rcmgr:         rcmgr,
	}
}

func NewStreamScope(dir network.Direction, limit Limit, peer *PeerScope) *StreamScope {
	return &StreamScope{
		ResourceScope: NewResourceScope(limit, []*ResourceScope{peer.ResourceScope, peer.rcmgr.transient.ResourceScope, peer.rcmgr.system.ResourceScope}),
		dir:           dir,
		rcmgr:         peer.rcmgr,
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
	stat := s.ResourceScope.rc.stat()
	if err := s.peer.ReserveForChild(stat); err != nil {
		s.peer.DecRef()
		s.peer = nil
		return err
	}

	s.rcmgr.transient.ReleaseForChild(stat)
	s.rcmgr.transient.DecRef() // removed from constraints

	// update constraints
	constraints := []*ResourceScope{
		s.peer.ResourceScope,
		s.rcmgr.system.ResourceScope,
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
	stat := s.ResourceScope.rc.stat()
	if err := s.proto.ReserveForChild(stat); err != nil {
		s.proto.DecRef()
		s.proto = nil
		return err
	}

	s.rcmgr.transient.ReleaseForChild(stat)
	s.rcmgr.transient.DecRef() // removed from constraints

	// update constraints
	constraints := []*ResourceScope{
		s.peer.ResourceScope,
		s.proto.ResourceScope,
		s.rcmgr.system.ResourceScope,
	}
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
	if err := s.svc.ReserveForChild(s.ResourceScope.rc.stat()); err != nil {
		s.svc.DecRef()
		s.svc = nil
		return err
	}

	// update constraints
	constraints := []*ResourceScope{
		s.peer.ResourceScope,
		s.proto.ResourceScope,
		s.svc.ResourceScope,
		s.rcmgr.system.ResourceScope,
	}
	s.ResourceScope.constraints = constraints

	return nil
}

func (s *StreamScope) PeerScope() network.PeerScope {
	s.Lock()
	defer s.Unlock()
	return s.peer
}
