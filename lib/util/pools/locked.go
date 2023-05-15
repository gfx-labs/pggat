package pools

import "sync"

type Locked[T any] struct {
	inner []T
	mu    sync.Mutex
}

func (L *Locked[T]) Get() (T, bool) {
	L.mu.Lock()
	defer L.mu.Unlock()
	if len(L.inner) == 0 {
		return *new(T), false
	}
	v := L.inner[len(L.inner)-1]
	L.inner = L.inner[:len(L.inner)-1]
	return v, true
}

func (L *Locked[T]) Put(v T) {
	L.mu.Lock()
	defer L.mu.Unlock()
	L.inner = append(L.inner, v)
}
