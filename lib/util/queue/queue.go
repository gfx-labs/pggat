package queue

import "sync"

type LIFO[T any] struct {
	items  []T
	signal sync.Cond
	mu     sync.Mutex
}

func (v *LIFO[T]) init() {
	if v.signal.L == nil {
		v.signal.L = &v.mu
	}
}

func (v *LIFO[T]) Push(item T) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.init()

	v.items = append(v.items, item)
	v.signal.Signal()
}

func (v *LIFO[T]) Pop() T {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.init()

	for len(v.items) == 0 {
		v.signal.Wait()
	}
	item := v.items[len(v.items)-1]
	v.items = v.items[:len(v.items)-1]

	return item
}
