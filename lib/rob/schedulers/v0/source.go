package v0

import (
	"sync"
	"sync/atomic"

	"pggat2/lib/rob"
)

type Source struct {
	sink atomic.Pointer[Sink]

	// vruntime in CFS
	runtime atomic.Int64

	queue []any
	mu    sync.RWMutex
}

func newSource() *Source {
	return &Source{}
}

func (T *Source) Schedule(w any) {
	T.mu.Lock()
	wasEmpty := len(T.queue) == 0
	T.queue = append(T.queue, w)
	T.mu.Unlock()
	if wasEmpty {
		sink := T.sink.Load()
		if sink != nil {
			sink.runnable(T)
		}
	}
}

func (T *Source) idle() bool {
	T.mu.RLock()
	defer T.mu.RUnlock()
	return len(T.queue) == 0
}

func (T *Source) pop() any {
	T.mu.Lock()
	defer T.mu.Unlock()

	if len(T.queue) == 0 {
		panic("pop on empty Source")
	}

	w := T.queue[0]
	for i := 1; i < len(T.queue)-1; i++ {
		T.queue[i] = T.queue[i+1]
	}
	T.queue = T.queue[:len(T.queue)-1]
	return w
}

func (T *Source) assign(sink *Sink) {
	T.sink.Store(sink)
}

var _ rob.Source = (*Source)(nil)
