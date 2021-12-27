package rcmgr

import (
	"fmt"
	"sync"

	"github.com/libp2p/go-buffer-pool"
	"github.com/libp2p/go-libp2p-core/network"
)

// Basic resource mamagement.
type Resources struct {
	limit Limit

	nconnsIn, nconnsOut     int
	nstreamsIn, nstreamsOut int
	nfd                     int

	memory  int64
	buffers map[int][]byte
	nextBuf int
}

// DAG ResourceScopes.
// Resources accounts for the node usage, constraints signify
// the dependencies that constrain resource usage.
type ResourceScope struct {
	sync.Mutex
	done   bool
	refCnt int

	rc          *Resources
	constraints []*ResourceScope
}

var _ network.ResourceScope = (*ResourceScope)(nil)
var _ network.TransactionalScope = (*ResourceScope)(nil)

type Buffer struct {
	s    *ResourceScope
	data []byte
	key  int
}

var _ network.Buffer = (*Buffer)(nil)

func NewResources(limit Limit) *Resources {
	return &Resources{
		limit: limit,
	}
}

func NewResourceScope(limit Limit, constraints []*ResourceScope) *ResourceScope {
	for _, cst := range constraints {
		cst.IncRef()
	}
	return &ResourceScope{
		rc:          NewResources(limit),
		constraints: constraints,
	}
}

// Resources implementation
func (rc *Resources) checkMemory(rsvp int64) error {
	// overflow check; this also has the side-effect that we cannot reserve negative memory.
	newmem := rc.memory + int64(rsvp)
	if newmem < rc.memory {
		return fmt.Errorf("memory reservation overflow: %w", ErrResourceLimitExceeded)
	}

	// limit check
	if newmem > rc.limit.GetMemoryLimit() {
		return fmt.Errorf("cannot reserve memory: %w", ErrResourceLimitExceeded)
	}

	return nil
}

func (rc *Resources) releaseBuffers() {
	for _, buf := range rc.buffers {
		pool.Put(buf)
	}
	rc.buffers = nil
}

func (rc *Resources) reserveMemory(size int64) error {
	if err := rc.checkMemory(size); err != nil {
		return err
	}

	rc.memory += int64(size)
	return nil
}

func (rc *Resources) releaseMemory(size int64) {
	rc.memory -= size

	// sanity check for bugs upstream
	if rc.memory < 0 {
		panic("BUG: too much memory released")
	}
}

func (rc *Resources) getBuffer(size int) ([]byte, int, error) {
	if err := rc.checkMemory(int64(size)); err != nil {
		return nil, -1, err
	}

	buf := pool.Get(size)
	key := rc.nextBuf

	rc.memory += int64(size)
	if rc.buffers == nil {
		rc.buffers = make(map[int][]byte)
	}
	rc.buffers[key] = buf
	rc.nextBuf++

	return buf, key, nil
}

func (rc *Resources) growBuffer(key int, newsize int) ([]byte, error) {
	oldbuf, ok := rc.buffers[key]
	if !ok {
		return nil, fmt.Errorf("invalid buffer; cannot grow buffer not allocated through this scope")
	}

	grow := newsize - len(oldbuf)
	if err := rc.checkMemory(int64(grow)); err != nil {
		return nil, err
	}

	newbuf := pool.Get(newsize)
	copy(newbuf, oldbuf)

	rc.memory += int64(grow)
	rc.buffers[key] = newbuf

	return newbuf, nil
}

func (rc *Resources) releaseBuffer(key int) {
	buf, ok := rc.buffers[key]
	if !ok {
		panic("BUG: release unknown buffer")
	}

	rc.memory -= int64(len(buf))

	// sanity check for bugs upstream
	if rc.memory < 0 {
		panic("BUG: too much memory released")
	}

	delete(rc.buffers, key)
	pool.Put(buf)
}

func (rc *Resources) addStream(dir network.Direction) error {
	if dir == network.DirInbound {
		return rc.addStreams(1, 0)
	}
	return rc.addStreams(0, 1)
}

func (rc *Resources) addStreams(incount, outcount int) error {
	if incount > 0 && rc.nstreamsIn+incount > rc.limit.GetStreamLimit(network.DirInbound) {
		return fmt.Errorf("cannot reserve stream: %w", ErrResourceLimitExceeded)
	}
	if outcount > 0 && rc.nstreamsOut+outcount > rc.limit.GetStreamLimit(network.DirOutbound) {
		return fmt.Errorf("cannot reserve stream: %w", ErrResourceLimitExceeded)
	}

	rc.nstreamsIn += incount
	rc.nstreamsOut += outcount
	return nil
}

func (rc *Resources) removeStream(dir network.Direction) {
	if dir == network.DirInbound {
		rc.removeStreams(1, 0)
	} else {
		rc.removeStreams(0, 1)
	}
}

func (rc *Resources) removeStreams(incount, outcount int) {
	rc.nstreamsIn -= incount
	rc.nstreamsOut -= outcount

	if rc.nstreamsIn < 0 || rc.nstreamsOut < 0 {
		panic("BUG: too many streams released")
	}
}

func (rc *Resources) addConn(dir network.Direction) error {
	if dir == network.DirInbound {
		return rc.addConns(1, 0)
	}
	return rc.addConns(0, 1)
}

func (rc *Resources) addConns(incount, outcount int) error {
	if incount > 0 && rc.nconnsIn+incount > rc.limit.GetConnLimit(network.DirInbound) {
		return fmt.Errorf("cannot reserve connection: %w", ErrResourceLimitExceeded)
	}
	if outcount > 0 && rc.nconnsOut+outcount > rc.limit.GetConnLimit(network.DirOutbound) {
		return fmt.Errorf("cannot reserve connection: %w", ErrResourceLimitExceeded)
	}

	rc.nconnsIn += incount
	rc.nconnsOut += outcount
	return nil
}

func (rc *Resources) removeConn(dir network.Direction) {
	if dir == network.DirInbound {
		rc.removeConns(1, 0)
	} else {
		rc.removeConns(0, 1)
	}
}

func (rc *Resources) removeConns(incount, outcount int) {
	rc.nconnsIn -= incount
	rc.nconnsOut -= outcount

	if rc.nconnsIn < 0 || rc.nconnsOut < 0 {
		panic("BUG: too many connections released")
	}
}

func (rc *Resources) addFD(count int) error {
	if rc.nfd+count > rc.limit.GetFDLimit() {
		return fmt.Errorf("cannot reserve file descriptor: %w", ErrResourceLimitExceeded)
	}

	rc.nfd += count
	return nil
}

func (rc *Resources) removeFD(count int) {
	rc.nfd -= count

	if rc.nfd < 0 {
		panic("BUG: too many file descriptors released")
	}
}

func (rc *Resources) stat() network.ScopeStat {
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
		return ErrResourceScopeClosed
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
	for _, cst := range s.constraints {
		cst.ReleaseMemoryForChild(int64(size))
	}
}

func (s *ResourceScope) ReserveMemoryForChild(size int64) error {
	s.Lock()
	defer s.Unlock()

	if s.done {
		return ErrResourceScopeClosed
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
	for _, cst := range s.constraints {
		cst.ReleaseMemoryForChild(int64(size))
	}
}

func (s *ResourceScope) ReleaseMemoryForChild(size int64) {
	s.Lock()
	defer s.Unlock()

	if s.done {
		return
	}

	s.rc.releaseMemory(size)
}

func (s *ResourceScope) GetBuffer(size int) (network.Buffer, error) {
	s.Lock()
	defer s.Unlock()

	if s.done {
		return nil, ErrResourceScopeClosed
	}

	buf, key, err := s.rc.getBuffer(size)
	if err != nil {
		return nil, err
	}

	if err := s.reserveMemoryForConstraints(size); err != nil {
		s.rc.releaseBuffer(key)
		return nil, err
	}

	return &Buffer{s: s, data: buf, key: key}, nil
}

func (b *Buffer) Data() []byte { return b.data }

func (b *Buffer) Grow(newsize int) error {
	b.s.Lock()
	defer b.s.Unlock()

	if b.s.done {
		return ErrResourceScopeClosed
	}

	grow := newsize - len(b.data)
	if err := b.s.reserveMemoryForConstraints(grow); err != nil {
		return err
	}

	newbuf, err := b.s.rc.growBuffer(b.key, newsize)
	if err != nil {
		b.s.releaseMemoryForConstraints(grow)
		return err
	}

	b.data = newbuf
	return nil
}

func (b *Buffer) Release() {
	b.s.Lock()
	defer b.s.Unlock()

	if b.data == nil {
		return
	}

	if b.s.done {
		return
	}

	for _, cst := range b.s.constraints {
		cst.ReleaseMemoryForChild(int64(len(b.data)))
	}
	b.s.rc.releaseBuffer(b.key)
	b.data = nil
}

func (s *ResourceScope) AddStream(dir network.Direction) error {
	s.Lock()
	defer s.Unlock()

	if s.done {
		return ErrResourceScopeClosed
	}

	if err := s.rc.addStream(dir); err != nil {
		return err
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
		return ErrResourceScopeClosed
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
		return ErrResourceScopeClosed
	}

	if err := s.rc.addConn(dir); err != nil {
		return err
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
		s.rc.removeConn(dir)
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
		return ErrResourceScopeClosed
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
		return ErrResourceScopeClosed
	}

	if err := s.rc.addFD(count); err != nil {
		return err
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
		return ErrResourceScopeClosed
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
		return ErrResourceScopeClosed
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

func (s *ResourceScope) BeginTxn() (network.TransactionalScope, error) {
	s.Lock()
	defer s.Unlock()

	if s.done {
		return nil, ErrResourceScopeClosed
	}

	constraints := make([]*ResourceScope, len(s.constraints)+1)
	constraints[0] = s
	copy(constraints[1:], s.constraints)

	return NewResourceScope(s.rc.limit, constraints), nil
}

func (s *ResourceScope) Done() {
	s.Lock()
	defer s.Unlock()

	if s.done {
		return
	}

	stat := s.rc.stat()
	for _, cst := range s.constraints {
		cst.ReleaseForChild(stat)
		cst.DecRef()
	}

	s.rc.releaseBuffers()

	s.rc.nstreamsIn = 0
	s.rc.nstreamsOut = 0
	s.rc.nconnsIn = 0
	s.rc.nconnsOut = 0
	s.rc.nfd = 0
	s.rc.memory = 0
	s.rc.buffers = nil

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
