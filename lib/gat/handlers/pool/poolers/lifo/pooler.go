package lifo

import (
	"sync"

	"github.com/google/uuid"

	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool"
	"gfx.cafe/gfx/pggat/lib/util/slices"
)

type Pooler struct {
	queue   []uuid.UUID
	servers map[uuid.UUID]struct{}
	ready   sync.Cond
	closed  bool
	mu      sync.Mutex
}

func (*Pooler) AddClient(_ uuid.UUID) {}

func (*Pooler) DeleteClient(_ uuid.UUID) {
	// nothing to do
}

func (T *Pooler) AddServer(server uuid.UUID) {
	T.mu.Lock()
	defer T.mu.Unlock()

	T.queue = append(T.queue, server)

	if T.servers == nil {
		T.servers = make(map[uuid.UUID]struct{})
	}
	T.servers[server] = struct{}{}

	if T.ready.L == nil {
		T.ready.L = &T.mu
	}
	T.ready.Signal()
}

func (T *Pooler) DeleteServer(server uuid.UUID) {
	T.mu.Lock()
	defer T.mu.Unlock()

	// remove server from queue
	T.queue = slices.Remove(T.queue, server)

	delete(T.servers, server)
}

func (T *Pooler) TryAcquire() uuid.UUID {
	T.mu.Lock()
	defer T.mu.Unlock()

	if T.closed {
		return uuid.Nil
	}

	if len(T.queue) == 0 {
		return uuid.Nil
	}

	server := T.queue[len(T.queue)-1]
	T.queue = T.queue[:len(T.queue)-1]
	return server
}

func (T *Pooler) AcquireBlocking() uuid.UUID {
	T.mu.Lock()
	defer T.mu.Unlock()

	if T.closed {
		return uuid.Nil
	}

	for len(T.queue) == 0 {
		if T.ready.L == nil {
			T.ready.L = &T.mu
		}
		T.ready.Wait()
	}

	if T.closed {
		return uuid.Nil
	}

	server := T.queue[len(T.queue)-1]
	T.queue = T.queue[:len(T.queue)-1]
	return server
}

func (T *Pooler) Acquire(_ uuid.UUID, mode pool.SyncMode) uuid.UUID {
	switch mode {
	case pool.SyncModeNonBlocking:
		return T.TryAcquire()
	case pool.SyncModeBlocking:
		return T.AcquireBlocking()
	default:
		return uuid.Nil
	}
}

func (T *Pooler) Release(server uuid.UUID) {
	T.mu.Lock()
	defer T.mu.Unlock()

	// check if server was removed
	if _, ok := T.servers[server]; !ok {
		return
	}

	T.queue = append(T.queue, server)
}

func (T *Pooler) Close() {
	T.mu.Lock()
	defer T.mu.Unlock()

	T.closed = true
	if T.ready.L == nil {
		T.ready.L = &T.mu
	}
	T.ready.Broadcast()
}

var _ pool.Pooler = (*Pooler)(nil)