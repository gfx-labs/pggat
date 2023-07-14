package pools

import "sync"

type Locked[T any] struct {
	inner Pool[T]
	mu    sync.Mutex
}

func (L *Locked[T]) Get() (T, bool) {
	L.mu.Lock()
	defer L.mu.Unlock()
	return L.inner.Get()
}

func (L *Locked[T]) Put(v T) {
	L.mu.Lock()
	defer L.mu.Unlock()
	L.inner.Put(v)
}
