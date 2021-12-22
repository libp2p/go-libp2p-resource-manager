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

func (rc *ResourceScope) ReserveMemory(size int) error {
	rc.Lock()
	defer rc.Unlock()

	if rc.memory+int64(size) > rc.limit.GetMemoryLimit() {
		return fmt.Errorf("cannot reserve memory: %w", ErrResourceLimitExceeded)
	}

	rc.memory += int64(size)
	return nil
}

func (rc *ResourceScope) ReleaseMemory(size int) {
	rc.Lock()
	defer rc.Unlock()

	rc.memory -= int64(size)
}

func (rc *ResourceScope) GetBuffer(size int) ([]byte, error) {
	rc.Lock()
	defer rc.Unlock()

	if rc.memory+int64(size) > rc.limit.GetMemoryLimit() {
		return nil, fmt.Errorf("cannot reserve memory: %w", ErrResourceLimitExceeded)
	}

	buf := pool.Get(size)

	rc.memory += int64(size)
	rc.buffers[buf] = buf

	return buf, nil
}

func (rc *ResourceScope) GrowBuffer(oldbuf []byte, newsize, ncopy int) ([]byte, error) {
	rc.Lock()
	defer rc.Unlock()

	grow := int64(newsize - len(oldbuf))
	if rc.memory+grow > rc.limit.GetMemoryLimit() {
		return nil, fmt.Errorf("cannot reserve memory: %w", ErrResourceLimitExceeded)
	}

	newbuf := pool.Get(newsize)

	if ncopy > 0 {
		copy(newbuf, oldbuf[:ncopy])
	}

	rc.memory += grow
	rc.buffers[newbuf] = newbuf
	delete(rc.buffers, oldbuf)

	return newbuf, nil
}

func (rc *ResourceScope) ReleaseBuffer(buf []byte) {
	rc.Lock()
	defer rc.Unlock()

	rc.memory -= int64(len(buf))
	delete(rc.buffers, buf)
	pool.Put(buf)
}

func (rc *ResourceScope) Stat() network.ScopeStat {
	rc.Lock()
	defer rc.Unlock()

	return network.ScopeStat{Memory: rc.memory}
}
