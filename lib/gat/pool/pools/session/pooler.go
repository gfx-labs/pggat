package session

import (
	"sync"

	"github.com/google/uuid"

	"pggat2/lib/gat/pool"
	"pggat2/lib/util/slices"
)

type Pooler struct {
	queue   []uuid.UUID
	servers map[uuid.UUID]struct{}
	ready   *sync.Cond
	mu      sync.Mutex
}

func (*Pooler) NewClient() uuid.UUID {
	return uuid.New()
}

func (*Pooler) DeleteClient(_ uuid.UUID) {
	// nothing to do
}

func (T *Pooler) NewServer() uuid.UUID {
	server := uuid.New()

	T.mu.Lock()
	defer T.mu.Unlock()

	T.queue = append(T.queue, server)

	if T.servers == nil {
		T.servers = make(map[uuid.UUID]struct{})
	}
	T.servers[server] = struct{}{}

	if T.ready != nil {
		T.ready.Signal()
	}

	return server
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

	for len(T.queue) == 0 {
		if T.ready == nil {
			T.ready = sync.NewCond(&T.mu)
		}
		T.ready.Wait()
	}

	server := T.queue[len(T.queue)-1]
	T.queue = T.queue[:len(T.queue)-1]
	return server
}

func (T *Pooler) Acquire(_ uuid.UUID, mode pool.SyncMode) uuid.UUID {
	switch mode {
	case pool.SyncModeBlocking:
		return T.TryAcquire()
	case pool.SyncModeNonBlocking:
		return T.AcquireBlocking()
	default:
		return uuid.Nil
	}
}

func (*Pooler) ReleaseAfterTransaction() bool {
	// servers are released when the client is removed
	return false
}

func (T *Pooler) Release(server uuid.UUID) {
	// check if server was removed
	if _, ok := T.servers[server]; !ok {
		return
	}

	T.queue = append(T.queue, server)
}

var _ pool.Pooler = (*Pooler)(nil)
