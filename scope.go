package rcmgr

import (
	"fmt"
	"sync"

	"github.com/libp2p/go-libp2p-core/network"
)

// resources tracks the current state of resource consumption
type resources struct {
	limit Limit

	nconnsIn, nconnsOut     int
	nstreamsIn, nstreamsOut int
	nfd                     int

	memory int64
}

// A ResourceScope can be a DAG, where a downstream node is not allowed to outlive an upstream node
// (ie cannot call Done in the upstream node before the downstream node) and account for resources
// using a linearized parent set.
// A ResourceScope can be a txn scope, where it has a specific owner; txn scopes create a tree rooted
// at the owner (which can be a DAG scope) and can outlive their parents -- this is important because
// txn scopes are the main *user* interface for memory management, and the user may call
// Done in a txn scope after the system has closed the root of the txn tree in some background
// goroutine.
// If we didn't make this distinction we would have a double release problem in that case.
type ResourceScope struct {
	sync.Mutex
	done   bool
	refCnt int

	rc          *resources
	owner       *ResourceScope   // set in transaction scopes, which define trees
	constraints []*ResourceScope // set in DAG scopes, it's the linearized parent set
}

var _ network.ResourceScope = (*ResourceScope)(nil)
var _ network.TransactionalScope = (*ResourceScope)(nil)

func newResources(limit Limit) *resources {
	return &resources{
		limit: limit,
	}
}

func NewResourceScope(limit Limit, constraints []*ResourceScope) *ResourceScope {
	for _, cst := range constraints {
		cst.IncRef()
	}
	return &ResourceScope{
		rc:          newResources(limit),
		constraints: constraints,
	}
}

func NewTxnResourceScope(owner *ResourceScope) *ResourceScope {
	return &ResourceScope{
		rc:    newResources(owner.rc.limit),
		owner: owner,
	}
}

// Resources implementation
func (rc *resources) checkMemory(rsvp int64) error {
	// overflow check; this also has the side effect that we cannot reserve negative memory.
	newmem := rc.memory + rsvp
	if newmem < rc.memory {
		return fmt.Errorf("memory reservation overflow: %w", network.ErrResourceLimitExceeded)
	}

	// limit check
	if newmem > rc.limit.GetMemoryLimit() {
		return fmt.Errorf("cannot reserve memory: %w", network.ErrResourceLimitExceeded)
	}

	return nil
}

func (rc *resources) reserveMemory(size int64) error {
	if err := rc.checkMemory(size); err != nil {
		return err
	}

	rc.memory += size
	return nil
}

func (rc *resources) releaseMemory(size int64) {
	rc.memory -= size

	// sanity check for bugs upstream
	if rc.memory < 0 {
		panic("BUG: too much memory released")
	}
}

func (rc *resources) addStream(dir network.Direction) error {
	if dir == network.DirInbound {
		return rc.addStreams(1, 0)
	}
	return rc.addStreams(0, 1)
}

func (rc *resources) addStreams(incount, outcount int) error {
	if incount > 0 && rc.nstreamsIn+incount > rc.limit.GetStreamLimit(network.DirInbound) {
		return fmt.Errorf("cannot reserve stream: %w", network.ErrResourceLimitExceeded)
	}
	if outcount > 0 && rc.nstreamsOut+outcount > rc.limit.GetStreamLimit(network.DirOutbound) {
		return fmt.Errorf("cannot reserve stream: %w", network.ErrResourceLimitExceeded)
	}

	rc.nstreamsIn += incount
	rc.nstreamsOut += outcount
	return nil
}

func (rc *resources) removeStream(dir network.Direction) {
	if dir == network.DirInbound {
		rc.removeStreams(1, 0)
	} else {
		rc.removeStreams(0, 1)
	}
}

func (rc *resources) removeStreams(incount, outcount int) {
	rc.nstreamsIn -= incount
	rc.nstreamsOut -= outcount

	if rc.nstreamsIn < 0 || rc.nstreamsOut < 0 {
		panic("BUG: too many streams released")
	}
}

func (rc *resources) addConn(dir network.Direction) error {
	if dir == network.DirInbound {
		return rc.addConns(1, 0)
	}
	return rc.addConns(0, 1)
}

func (rc *resources) addConns(incount, outcount int) error {
	if incount > 0 && rc.nconnsIn+incount > rc.limit.GetConnLimit(network.DirInbound) {
		return fmt.Errorf("cannot reserve connection: %w", network.ErrResourceLimitExceeded)
	}
	if outcount > 0 && rc.nconnsOut+outcount > rc.limit.GetConnLimit(network.DirOutbound) {
		return fmt.Errorf("cannot reserve connection: %w", network.ErrResourceLimitExceeded)
	}

	rc.nconnsIn += incount
	rc.nconnsOut += outcount
	return nil
}

func (rc *resources) removeConn(dir network.Direction) {
	if dir == network.DirInbound {
		rc.removeConns(1, 0)
	} else {
		rc.removeConns(0, 1)
	}
}

func (rc *resources) removeConns(incount, outcount int) {
	rc.nconnsIn -= incount
	rc.nconnsOut -= outcount

	if rc.nconnsIn < 0 || rc.nconnsOut < 0 {
		panic("BUG: too many connections released")
	}
}

func (rc *resources) addFD(count int) error {
	if rc.nfd+count > rc.limit.GetFDLimit() {
		return fmt.Errorf("cannot reserve file descriptor: %w", network.ErrResourceLimitExceeded)
	}

	rc.nfd += count
	return nil
}

func (rc *resources) removeFD(count int) {
	rc.nfd -= count

	if rc.nfd < 0 {
		panic("BUG: too many file descriptors released")
	}
}

func (rc *resources) stat() network.ScopeStat {
	return network.ScopeStat{
		Memory:             rc.memory,
		NumStreamsInbound:  rc.nstreamsIn,
		NumStreamsOutbound: rc.nstreamsOut,
		NumConnsInbound:    rc.nconnsIn,
		NumConnsOutbound:   rc.nconnsOut,
		NumFD:              rc.nfd,
	}
}

// ResourceScope implementation
func (s *ResourceScope) ReserveMemory(size int) error {
	s.Lock()
	defer s.Unlock()

	if s.done {
		return network.ErrResourceScopeClosed
	}

	if err := s.rc.reserveMemory(int64(size)); err != nil {
		return err
	}

	if err := s.reserveMemoryForConstraints(size); err != nil {
		s.rc.releaseMemory(int64(size))
		return err
	}

	return nil
}

func (s *ResourceScope) reserveMemoryForConstraints(size int) error {
	if s.owner != nil {
		return s.owner.ReserveMemory(size)
	}

	var reserved int
	var err error
	for _, cst := range s.constraints {
		if err = cst.ReserveMemoryForChild(int64(size)); err != nil {
			break
		}
		reserved++
	}

	if err != nil {
		// we failed because of a constraint; undo memory reservations
		for _, cst := range s.constraints[:reserved] {
			cst.ReleaseMemoryForChild(int64(size))
		}
	}

	return err
}

func (s *ResourceScope) releaseMemoryForConstraints(size int) {
	if s.owner != nil {
		s.owner.ReleaseMemory(size)
		return
	}

	for _, cst := range s.constraints {
		cst.ReleaseMemoryForChild(int64(size))
	}
}

func (s *ResourceScope) ReserveMemoryForChild(size int64) error {
	s.Lock()
	defer s.Unlock()

	if s.done {
		return network.ErrResourceScopeClosed
	}

	return s.rc.reserveMemory(size)
}

func (s *ResourceScope) ReleaseMemory(size int) {
	s.Lock()
	defer s.Unlock()

	if s.done {
		return
	}

	s.rc.releaseMemory(int64(size))
	s.releaseMemoryForConstraints(size)
}

func (s *ResourceScope) ReleaseMemoryForChild(size int64) {
	s.Lock()
	defer s.Unlock()

	if s.done {
		return
	}

	s.rc.releaseMemory(size)
}

func (s *ResourceScope) AddStream(dir network.Direction) error {
	s.Lock()
	defer s.Unlock()

	if s.done {
		return network.ErrResourceScopeClosed
	}

	if err := s.rc.addStream(dir); err != nil {
		return err
	}

	if err := s.addStreamForConstraints(dir); err != nil {
		s.rc.removeStream(dir)
		return err
	}

	return nil
}

func (s *ResourceScope) addStreamForConstraints(dir network.Direction) error {
	if s.owner != nil {
		return s.owner.AddStream(dir)
	}

	var err error
	var reserved int
	for _, cst := range s.constraints {
		if err = cst.AddStreamForChild(dir); err != nil {
			break
		}
		reserved++
	}

	if err != nil {
		for _, cst := range s.constraints[:reserved] {
			cst.RemoveStreamForChild(dir)
		}
	}

	return err
}

func (s *ResourceScope) AddStreamForChild(dir network.Direction) error {
	s.Lock()
	defer s.Unlock()

	if s.done {
		return network.ErrResourceScopeClosed
	}

	return s.rc.addStream(dir)
}

func (s *ResourceScope) RemoveStream(dir network.Direction) {
	s.Lock()
	defer s.Unlock()

	if s.done {
		return
	}

	s.rc.removeStream(dir)
	s.removeStreamForConstraints(dir)
}

func (s *ResourceScope) removeStreamForConstraints(dir network.Direction) {
	if s.owner != nil {
		s.owner.RemoveStream(dir)
		return
	}

	for _, cst := range s.constraints {
		cst.RemoveStreamForChild(dir)
	}
}

func (s *ResourceScope) RemoveStreamForChild(dir network.Direction) {
	s.Lock()
	defer s.Unlock()

	if s.done {
		return
	}

	s.rc.removeStream(dir)
}

func (s *ResourceScope) AddConn(dir network.Direction) error {
	s.Lock()
	defer s.Unlock()

	if s.done {
		return network.ErrResourceScopeClosed
	}

	if err := s.rc.addConn(dir); err != nil {
		return err
	}

	if err := s.addConnForConstraints(dir); err != nil {
		s.rc.removeConn(dir)
		return err
	}

	return nil
}

func (s *ResourceScope) addConnForConstraints(dir network.Direction) error {
	if s.owner != nil {
		return s.owner.AddConn(dir)
	}

	var err error
	var reserved int
	for _, cst := range s.constraints {
		if err = cst.AddConnForChild(dir); err != nil {
			break
		}
		reserved++
	}

	if err != nil {
		for _, cst := range s.constraints[:reserved] {
			cst.RemoveConnForChild(dir)
		}
	}

	return err
}

func (s *ResourceScope) AddConnForChild(dir network.Direction) error {
	s.Lock()
	defer s.Unlock()

	if s.done {
		return network.ErrResourceScopeClosed
	}

	return s.rc.addConn(dir)
}

func (s *ResourceScope) RemoveConn(dir network.Direction) {
	s.Lock()
	defer s.Unlock()

	if s.done {
		return
	}

	s.rc.removeConn(dir)
	s.removeConnForConstraints(dir)
}

func (s *ResourceScope) removeConnForConstraints(dir network.Direction) {
	if s.owner != nil {
		s.owner.RemoveConn(dir)
	}

	for _, cst := range s.constraints {
		cst.RemoveConnForChild(dir)
	}
}

func (s *ResourceScope) RemoveConnForChild(dir network.Direction) {
	s.Lock()
	defer s.Unlock()

	if s.done {
		return
	}

	s.rc.removeConn(dir)
}

func (s *ResourceScope) AddFD(count int) error {
	s.Lock()
	defer s.Unlock()

	if s.done {
		return network.ErrResourceScopeClosed
	}

	if err := s.rc.addFD(count); err != nil {
		return err
	}

	if err := s.addFDForConstraints(count); err != nil {
		s.rc.removeFD(count)
		return err
	}

	return nil
}

func (s *ResourceScope) addFDForConstraints(count int) error {
	if s.owner != nil {
		return s.owner.AddFD(count)
	}

	var err error
	var reserved int
	for _, cst := range s.constraints {
		if err = cst.AddFDForChild(count); err != nil {
			break
		}
		reserved++
	}

	if err != nil {
		for _, cst := range s.constraints[:reserved] {
			cst.RemoveFDForChild(count)
		}
	}

	return err
}

func (s *ResourceScope) AddFDForChild(count int) error {
	s.Lock()
	defer s.Unlock()

	if s.done {
		return network.ErrResourceScopeClosed
	}

	return s.rc.addFD(count)
}

func (s *ResourceScope) RemoveFD(count int) {
	s.Lock()
	defer s.Unlock()

	if s.done {
		return
	}

	s.rc.removeFD(count)
	s.removeFDForConstraints(count)
}

func (s *ResourceScope) removeFDForConstraints(count int) {
	if s.owner != nil {
		s.owner.RemoveFD(count)
		return
	}

	for _, cst := range s.constraints {
		cst.RemoveFDForChild(count)
	}
}

func (s *ResourceScope) RemoveFDForChild(count int) {
	s.Lock()
	defer s.Unlock()

	if s.done {
		return
	}

	s.rc.removeFD(count)
}

func (s *ResourceScope) ReserveForChild(st network.ScopeStat) error {
	s.Lock()
	defer s.Unlock()

	if s.done {
		return network.ErrResourceScopeClosed
	}

	if err := s.rc.reserveMemory(st.Memory); err != nil {
		return err
	}

	if err := s.rc.addStreams(st.NumStreamsInbound, st.NumStreamsOutbound); err != nil {
		s.rc.releaseMemory(st.Memory)
		return err
	}

	if err := s.rc.addConns(st.NumConnsInbound, st.NumConnsOutbound); err != nil {
		s.rc.releaseMemory(st.Memory)
		s.rc.removeStreams(st.NumStreamsInbound, st.NumStreamsOutbound)
		return err
	}

	if err := s.rc.addFD(st.NumFD); err != nil {
		s.rc.releaseMemory(st.Memory)
		s.rc.removeStreams(st.NumStreamsInbound, st.NumStreamsOutbound)
		s.rc.removeConns(st.NumConnsInbound, st.NumConnsOutbound)
		return err
	}

	return nil
}

func (s *ResourceScope) ReleaseForChild(st network.ScopeStat) {
	s.Lock()
	defer s.Unlock()

	if s.done {
		return
	}

	s.rc.releaseMemory(st.Memory)
	s.rc.removeStreams(st.NumStreamsInbound, st.NumStreamsOutbound)
	s.rc.removeConns(st.NumConnsInbound, st.NumConnsOutbound)
	s.rc.removeFD(st.NumFD)
}

func (s *ResourceScope) ReleaseResources(st network.ScopeStat) {
	s.Lock()
	defer s.Unlock()

	if s.done {
		return
	}

	s.rc.releaseMemory(st.Memory)
	s.rc.removeStreams(st.NumStreamsInbound, st.NumStreamsOutbound)
	s.rc.removeConns(st.NumConnsInbound, st.NumConnsOutbound)
	s.rc.removeFD(st.NumFD)

	if s.owner != nil {
		s.owner.ReleaseResources(st)
	} else {
		for _, cst := range s.constraints {
			cst.ReleaseForChild(st)
		}
	}
}

func (s *ResourceScope) BeginTransaction() (network.TransactionalScope, error) {
	s.Lock()
	defer s.Unlock()

	if s.done {
		return nil, network.ErrResourceScopeClosed
	}

	s.refCnt++
	return NewTxnResourceScope(s), nil
}

func (s *ResourceScope) Done() {
	s.Lock()
	defer s.Unlock()

	if s.done {
		return
	}

	stat := s.rc.stat()
	if s.owner != nil {
		s.owner.ReleaseResources(stat)
		s.owner.DecRef()
	} else {
		for _, cst := range s.constraints {
			cst.ReleaseForChild(stat)
			cst.DecRef()
		}
	}

	s.rc.nstreamsIn = 0
	s.rc.nstreamsOut = 0
	s.rc.nconnsIn = 0
	s.rc.nconnsOut = 0
	s.rc.nfd = 0
	s.rc.memory = 0

	s.done = true
}

func (s *ResourceScope) Stat() network.ScopeStat {
	s.Lock()
	defer s.Unlock()

	return s.rc.stat()
}

func (s *ResourceScope) IncRef() {
	s.Lock()
	defer s.Unlock()

	s.refCnt++
}

func (s *ResourceScope) DecRef() {
	s.Lock()
	defer s.Unlock()

	s.refCnt--
}

func (s *ResourceScope) IsUnused() bool {
	s.Lock()
	defer s.Unlock()

	if s.done {
		return true
	}

	if s.refCnt > 0 {
		return false
	}

	st := s.rc.stat()
	return st.NumStreamsInbound == 0 &&
		st.NumStreamsOutbound == 0 &&
		st.NumConnsInbound == 0 &&
		st.NumConnsOutbound == 0 &&
		st.NumFD == 0
}
