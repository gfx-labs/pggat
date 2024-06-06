package lifo

import (
	"sync"
	"time"

	"github.com/google/uuid"

	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool"
	"gfx.cafe/gfx/pggat/lib/util/pools"
	"gfx.cafe/gfx/pggat/lib/util/ring"
	"gfx.cafe/gfx/pggat/lib/util/slices"
)

type Pooler struct {
	waiting chan struct{}

	pool pools.Locked[chan uuid.UUID]

	servers map[uuid.UUID]struct{}
	queue   []uuid.UUID
	waiters ring.Ring[chan<- uuid.UUID]
	closed  bool
	mu      sync.Mutex
}

func NewPooler() *Pooler {
	return &Pooler{
		waiting: make(chan struct{}),
	}
}

func (*Pooler) AddClient(_ uuid.UUID) {}

func (*Pooler) DeleteClient(_ uuid.UUID) {
	// nothing to do
}

func (T *Pooler) AddServer(server uuid.UUID) {
	T.mu.Lock()
	defer T.mu.Unlock()

	if T.servers == nil {
		T.servers = make(map[uuid.UUID]struct{})
	}
	T.servers[server] = struct{}{}

	T.release(server)
}

func (T *Pooler) DeleteServer(server uuid.UUID) {
	T.mu.Lock()
	defer T.mu.Unlock()

	// remove server from queue
	T.queue = slices.Remove(T.queue, server)

	delete(T.servers, server)
}

func (T *Pooler) Acquire(_ uuid.UUID, timeout time.Duration) uuid.UUID {
	v, c := func() (uuid.UUID, chan uuid.UUID) {
		T.mu.Lock()
		defer T.mu.Unlock()

		if T.closed {
			return uuid.Nil, nil
		}

		if len(T.queue) > 0 {
			worker := T.queue[len(T.queue)-1]
			T.queue = T.queue[:len(T.queue)-1]

			return worker, nil
		}

		ready, _ := T.pool.Get()
		if ready == nil {
			ready = make(chan uuid.UUID, 1)
		}
		T.waiters.PushBack(ready)

		select {
		case T.waiting <- struct{}{}:
		default:
		}

		return uuid.Nil, ready
	}()

	if v != uuid.Nil {
		return v
	}

	if c != nil {
		var timeoutC <-chan time.Time
		if timeout != 0 {
			timer := time.NewTimer(timeout)
			defer timer.Stop()
			timeoutC = timer.C
		}

		var ok bool
		select {
		case v, ok = <-c:
			if ok {
				T.pool.Put(c)
			}
		case <-timeoutC:
			T.mu.Lock()
			defer T.mu.Unlock()

			// try to remove the channel from the queue, we might've lost the race though
			waitCount := T.waiters.Length()
			var found bool
			for i := 0; i < waitCount; i++ {
				cc, _ := T.waiters.PopFront()
				if c == cc {
					found = true
					// we still have to go around the whole thing to maintain order
				} else {
					T.waiters.PushBack(cc)
				}
			}

			if found {
				T.pool.Put(c)
			} else {
				// we lost the race :(, we have a worker though
				v, ok = <-c
				if ok {
					T.pool.Put(c)
				}
			}
		}
	}

	return v
}

func (T *Pooler) release(server uuid.UUID) {
	// check if server was removed
	if _, ok := T.servers[server]; !ok {
		return
	}

	if c, ok := T.waiters.PopFront(); ok {
		c <- server
		return
	}

	T.queue = append(T.queue, server)
}

func (T *Pooler) Release(server uuid.UUID) {
	T.mu.Lock()
	defer T.mu.Unlock()

	T.release(server)
}

func (T *Pooler) Waiting() <-chan struct{} {
	return T.waiting
}

func (T *Pooler) Waiters() int {
	T.mu.Lock()
	defer T.mu.Unlock()

	return T.waiters.Length()
}

func (T *Pooler) Close() {
	T.mu.Lock()
	defer T.mu.Unlock()

	T.closed = true
	clear(T.servers)
	T.queue = T.queue[:0]
	for c, ok := T.waiters.PopFront(); ok; c, ok = T.waiters.PopFront() {
		close(c)
	}
}

var _ pool.Pooler = (*Pooler)(nil)
