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
	buffers map[interface{}][]byte
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
		limit: limit,
	}
}

func NewResourceScope(limit Limit, constraints []*ResourceScope) *ResourceScope {
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
	for key, buf := range rc.buffers {
		pool.Put(buf)
		delete(rc.buffers, key)
	}
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

func (rc *Resources) getBuffer(size int) ([]byte, error) {
	if err := rc.checkMemory(int64(size)); err != nil {
		return nil, err
	}

	buf := pool.Get(size)

	rc.memory += int64(size)
	if rc.buffers == nil {
		rc.buffers = make(map[interface{}][]byte)
	}
	rc.buffers[buf] = buf

	return buf, nil
}

func (rc *Resources) growBuffer(oldbuf []byte, newsize int) ([]byte, error) {
	grow := newsize - len(oldbuf)
	if err := rc.checkMemory(int64(grow)); err != nil {
		return nil, err
	}

	_, ok := rc.buffers[oldbuf]
	if !ok {
		return nil, fmt.Errorf("cannot grow unknown buffer")
	}

	newbuf := pool.Get(newsize)
	copy(newbuf, oldbuf)

	rc.memory += int64(grow)
	rc.buffers[newbuf] = newbuf
	delete(rc.buffers, oldbuf)

	return newbuf, nil
}

func (rc *Resources) releaseBuffer(buf []byte) {
	_, ok := rc.buffers[buf]
	if !ok {
		panic("BUG: release unknown buffer")
	}

	rc.memory -= int64(len(buf))

	// sanity check for bugs upstream
	if rc.memory < 0 {
		panic("BUG: too much memory released")
	}

	delete(rc.buffers, buf)
	pool.Put(buf)
}

func (rc *Resources) addStream(incount, outcount int) error {
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

func (rc *Resources) removeStream(incount, outcount int) {
	rc.nstreamsIn -= incount
	rc.nstreamsOut -= outcount

	if rc.nstreamsIn < 0 || rc.nstreamsOut < 0 {
		panic("BUG: too many streams released")
	}
}

func (rc *Resources) addConn(incount, outcount int) error {
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

func (rc *Resources) removeConn(incount, outcount int) {
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
		Memory:     rc.memory,
		NumConns:   rc.nconnsIn + rc.nconnsOut,
		NumStreams: rc.nstreamsIn + rc.nstreamsOut,
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

func (s *ResourceScope) AddStream(dir network.Direction) error {
	s.Lock()
	defer s.Unlock()

	if s.done {
		return ErrResourceScopeClosed
	}

	var incount, outcount int
	if dir == network.DirInbound {
		incount = 1
	} else {
		outcount = 1
	}
	if err := s.rc.addStream(incount, outcount); err != nil {
		return err
	}

	var err error
	var reserved int
	for _, cst := range s.constraints {
		var incount, outcount int
		if dir == network.DirInbound {
			incount = 1
		} else {
			outcount = 1
		}
		if err = cst.AddStreamForChild(incount, outcount); err != nil {
			break
		}
		reserved++
	}

	if err != nil {
		for _, cst := range s.constraints[:reserved] {
			cst.RemoveStreamForChild(incount, outcount)
		}
	}

	return err
}

func (s *ResourceScope) AddStreamForChild(incount, outcount int) error {
	s.Lock()
	defer s.Unlock()

	return s.rc.addStream(incount, outcount)
}

func (s *ResourceScope) RemoveStream(dir network.Direction) {
	s.Lock()
	defer s.Unlock()

	var incount, outcount int
	if dir == network.DirInbound {
		incount = 1
	} else {
		outcount = 1
	}

	s.rc.removeStream(incount, outcount)
	for _, cst := range s.constraints {
		cst.RemoveStreamForChild(incount, outcount)
	}
}

func (s *ResourceScope) RemoveStreamForChild(incount, outcount int) {
	s.Lock()
	defer s.Unlock()
	s.rc.removeStream(incount, outcount)
}

func (s *ResourceScope) AddConn(dir network.Direction) error {
	s.Lock()
	defer s.Unlock()

	if s.done {
		return ErrResourceScopeClosed
	}

	var incount, outcount int
	if dir == network.DirInbound {
		incount = 1
	} else {
		outcount = 1
	}
	if err := s.rc.addConn(incount, outcount); err != nil {
		return err
	}

	var err error
	var reserved int
	for _, cst := range s.constraints {
		if err = cst.AddConnForChild(incount, outcount); err != nil {
			break
		}
		reserved++
	}

	if err != nil {
		for _, cst := range s.constraints[:reserved] {
			cst.RemoveConnForChild(incount, outcount)
		}
	}

	return err
}

func (s *ResourceScope) AddConnForChild(incount, outcount int) error {
	s.Lock()
	defer s.Unlock()

	return s.rc.addConn(incount, outcount)
}

func (s *ResourceScope) RemoveConn(dir network.Direction) {
	s.Lock()
	defer s.Unlock()

	var incount, outcount int
	if dir == network.DirInbound {
		incount = 1
	} else {
		outcount = 1
	}

	s.rc.removeConn(incount, outcount)
	for _, cst := range s.constraints {
		cst.RemoveConnForChild(incount, outcount)
	}
}

func (s *ResourceScope) RemoveConnForChild(incount, outcount int) {
	s.Lock()
	defer s.Unlock()
	s.rc.removeConn(incount, outcount)
}

func (s *ResourceScope) AddFD(count int) error {
	s.Lock()
	defer s.Unlock()

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
	return s.rc.addFD(count)
}

func (s *ResourceScope) RemoveFD(count int) {
	s.Lock()
	defer s.Unlock()

	s.rc.removeFD(count)
	for _, cst := range s.constraints {
		cst.RemoveFDForChild(count)
	}
}

func (s *ResourceScope) RemoveFDForChild(count int) {
	s.Lock()
	defer s.Unlock()
	s.rc.removeFD(count)
}

func (s *ResourceScope) Done() {
	s.Lock()
	defer s.Unlock()

	if s.done {
		return
	}

	for _, cst := range s.constraints {
		cst.ReleaseMemoryForChild(s.rc.memory)
		cst.RemoveStreamForChild(s.rc.nstreamsIn, s.rc.nstreamsOut)
		cst.RemoveConnForChild(s.rc.nconnsIn, s.rc.nconnsOut)
		cst.RemoveFDForChild(s.rc.nfd)
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
