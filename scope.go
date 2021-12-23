package rcmgr

import (
	"fmt"
	"sync"

	"github.com/libp2p/go-buffer-pool"
	"github.com/libp2p/go-libp2p-core/network"
)

// Basic resource mamagement.
type Resources struct {
	limit    Limit
	nconns   int
	nstreams int
	memory   int64
	buffers  map[interface{}][]byte
}

// DAG ResourceScopes.
// Resources accounts for the node usage, constraints signify
// the dependencies that constrain resource usage.
type ResourceScope struct {
	sync.Mutex
	done bool

	rc          *Resources
	constraints []*ResourceScope
}

var _ network.ResourceScope = (*ResourceScope)(nil)
var _ network.TransactionalScope = (*ResourceScope)(nil)

func NewResources(limit Limit) *Resources {
	return &Resources{
		limit:   limit,
		buffers: make(map[interface{}][]byte),
	}
}

func NewResourceScope(limit Limit, constraints []*ResourceScope) *ResourceScope {
	return &ResourceScope{
		rc:          NewResources(limit),
		constraints: constraints,
	}
}

// Resources implementation
func (rc *Resources) checkMemory(rsvp int) error {
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
	for key, buf := range rc.buffers {
		pool.Put(buf)
		delete(rc.buffers, key)
	}
}

func (rc *Resources) reserveMemory(size int) error {
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

func (rc *Resources) getBuffer(size int) ([]byte, error) {
	if err := rc.checkMemory(size); err != nil {
		return nil, err
	}

	buf := pool.Get(size)

	rc.memory += int64(size)
	rc.buffers[buf] = buf

	return buf, nil
}

func (rc *Resources) growBuffer(oldbuf []byte, newsize int) ([]byte, error) {
	grow := newsize - len(oldbuf)
	if err := rc.checkMemory(grow); err != nil {
		return nil, err
	}

	newbuf := pool.Get(newsize)
	copy(newbuf, oldbuf)

	rc.memory += int64(grow)
	rc.buffers[newbuf] = newbuf
	delete(rc.buffers, oldbuf)

	return newbuf, nil
}

func (rc *Resources) releaseBuffer(buf []byte) {
	rc.memory -= int64(len(buf))

	// sanity check for bugs upstream
	if rc.memory < 0 {
		panic("BUG: too much memory released")
	}

	delete(rc.buffers, buf)
	pool.Put(buf)
}

func (rc *Resources) addStream(count int) error {
	if rc.nstreams+count > rc.limit.GetStreamLimit() {
		return fmt.Errorf("cannot reserve stream: %w", ErrResourceLimitExceeded)
	}

	rc.nstreams += count
	return nil
}

func (rc *Resources) removeStream(count int) {
	rc.nstreams -= count

	if rc.nstreams < 0 {
		panic("BUG: too many streams released")
	}
}

func (rc *Resources) addConn(count int) error {
	if rc.nconns+count > rc.limit.GetConnLimit() {
		return fmt.Errorf("cannot reserve connection: %w", ErrResourceLimitExceeded)
	}

	rc.nconns += count
	return nil
}

func (rc *Resources) removeConn(count int) {
	rc.nconns -= count

	if rc.nconns < 0 {
		panic("BUG: too many connections released")
	}
}

func (rc *Resources) stat() network.ScopeStat {
	return network.ScopeStat{
		Memory:     rc.memory,
		NumConns:   rc.nconns,
		NumStreams: rc.nstreams,
	}
}

// ResourceScope implementation
func (s *ResourceScope) ReserveMemory(size int) error {
	s.Lock()
	defer s.Unlock()

	if s.done {
		return ErrResourceScopeClosed
	}

	if err := s.rc.reserveMemory(size); err != nil {
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
		if err = cst.ReserveMemoryForChild(size); err != nil {
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

func (s *ResourceScope) ReserveMemoryForChild(size int) error {
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

func (s *ResourceScope) GetBuffer(size int) ([]byte, error) {
	s.Lock()
	defer s.Unlock()

	if s.done {
		return nil, ErrResourceScopeClosed
	}

	buf, err := s.rc.getBuffer(size)
	if err != nil {
		return nil, err
	}

	if err := s.reserveMemoryForConstraints(size); err != nil {
		s.rc.releaseBuffer(buf)
		return nil, err
	}

	return buf, err
}

func (s *ResourceScope) GrowBuffer(oldbuf []byte, newsize int) ([]byte, error) {
	s.Lock()
	defer s.Unlock()

	if s.done {
		return nil, ErrResourceScopeClosed
	}

	buf, err := s.rc.growBuffer(oldbuf, newsize)
	if err != nil {
		return nil, err
	}

	if err := s.reserveMemoryForConstraints(newsize - len(oldbuf)); err != nil {
		s.rc.releaseBuffer(buf)
		return nil, err
	}

	return buf, err
}

func (s *ResourceScope) ReleaseBuffer(buf []byte) {
	s.Lock()
	defer s.Unlock()

	if s.done {
		return
	}

	s.rc.releaseBuffer(buf)
	for _, cst := range s.constraints {
		cst.ReleaseMemoryForChild(int64(len(buf)))
	}
}

func (s *ResourceScope) AddStream(count int) error {
	s.Lock()
	defer s.Unlock()

	if s.done {
		return ErrResourceScopeClosed
	}

	if err := s.rc.addStream(count); err != nil {
		return err
	}

	var err error
	var reserved int
	for _, cst := range s.constraints {
		if err = cst.AddStreamForChild(count); err != nil {
			break
		}
		reserved++
	}

	if err != nil {
		for _, cst := range s.constraints[:reserved] {
			cst.RemoveStreamForChild(count)
		}
	}

	return err
}

func (s *ResourceScope) AddStreamForChild(count int) error {
	s.Lock()
	defer s.Unlock()

	return s.rc.addStream(count)
}

func (s *ResourceScope) RemoveStream(count int) {
	s.Lock()
	defer s.Unlock()

	s.rc.removeStream(count)
	for _, cst := range s.constraints {
		cst.RemoveStreamForChild(count)
	}
}

func (s *ResourceScope) RemoveStreamForChild(count int) {
	s.Lock()
	defer s.Unlock()
	s.rc.removeStream(count)
}

func (s *ResourceScope) AddConn(count int) error {
	s.Lock()
	defer s.Unlock()

	if s.done {
		return ErrResourceScopeClosed
	}

	if err := s.rc.addConn(count); err != nil {
		return err
	}

	var err error
	var reserved int
	for _, cst := range s.constraints {
		if err = cst.AddConnForChild(count); err != nil {
			break
		}
		reserved++
	}

	if err != nil {
		for _, cst := range s.constraints[:reserved] {
			cst.RemoveConnForChild(count)
		}
	}

	return err
}

func (s *ResourceScope) AddConnForChild(count int) error {
	s.Lock()
	defer s.Unlock()

	return s.rc.addConn(count)
}

func (s *ResourceScope) RemoveConn(count int) {
	s.Lock()
	defer s.Unlock()

	s.rc.removeConn(count)
	for _, cst := range s.constraints {
		cst.RemoveConnForChild(count)
	}
}

func (s *ResourceScope) RemoveConnForChild(count int) {
	s.Lock()
	defer s.Unlock()
	s.rc.removeConn(count)
}

func (s *ResourceScope) Done() {
	s.Lock()
	defer s.Unlock()

	if s.done {
		return
	}

	for _, cst := range s.constraints {
		cst.ReleaseMemoryForChild(s.rc.memory)
		cst.RemoveStreamForChild(s.rc.nstreams)
		cst.RemoveConnForChild(s.rc.nconns)
	}

	s.rc.releaseBuffers()

	s.rc.nstreams = 0
	s.rc.nconns = 0
	s.rc.memory = 0
	s.rc.buffers = nil

	s.done = true
}

func (s *ResourceScope) Stat() network.ScopeStat {
	s.Lock()
	defer s.Unlock()

	return s.rc.stat()
}
