package rcmgr

import (
	"fmt"
	"sync"

	"github.com/libp2p/go-buffer-pool"
	"github.com/libp2p/go-libp2p-core/network"
)

type ResourceScope struct {
	sync.Mutex

	limit   Limit
	memory  int64
	buffers map[interface{}][]byte
}

var _ network.ResourceScope = (*ResourceScope)(nil)

func (rc *ResourceScope) checkMemory(rsvp int) error {
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

func (rc *ResourceScope) releaseBuffers() {
	for key, buf := range rc.buffers {
		pool.Put(buf)
		delete(rc.buffers, key)
	}
}

func (rc *ResourceScope) ReserveMemory(size int) error {
	rc.Lock()
	defer rc.Unlock()

	if err := rc.checkMemory(size); err != nil {
		return err
	}

	rc.memory += int64(size)
	return nil
}

func (rc *ResourceScope) ReleaseMemory(size int) {
	rc.Lock()
	defer rc.Unlock()

	rc.memory -= int64(size)

	// sanity check for bugs upstream
	if rc.memory < 0 {
		panic("BUG: too much memory released")
	}
}

func (rc *ResourceScope) GetBuffer(size int) ([]byte, error) {
	rc.Lock()
	defer rc.Unlock()

	if err := rc.checkMemory(size); err != nil {
		return nil, err
	}

	buf := pool.Get(size)

	rc.memory += int64(size)
	rc.buffers[buf] = buf

	return buf, nil
}

func (rc *ResourceScope) GrowBuffer(oldbuf []byte, newsize int) ([]byte, error) {
	rc.Lock()
	defer rc.Unlock()

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

func (rc *ResourceScope) ReleaseBuffer(buf []byte) {
	rc.Lock()
	defer rc.Unlock()

	rc.memory -= int64(len(buf))

	// sanity check for bugs upstream
	if rc.memory < 0 {
		panic("BUG: too much memory released")
	}

	delete(rc.buffers, buf)
	pool.Put(buf)
}

func (rc *ResourceScope) Stat() network.ScopeStat {
	rc.Lock()
	defer rc.Unlock()

	return network.ScopeStat{Memory: rc.memory}
}
