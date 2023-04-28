package v0

import (
	"math/rand"
	"sync"

	"pggat2/lib/rob"
)

type Scheduler struct {
	sinks     []*Sink
	backorder []*Source
	mu        sync.Mutex
}

func NewScheduler() *Scheduler {
	return &Scheduler{}
}

func (T *Scheduler) NewSink() rob.Sink {
	sink := newSink()
	T.mu.Lock()
	defer T.mu.Unlock()
	for _, source := range T.backorder {
		source.assign(sink)
	}
	T.backorder = T.backorder[:0]
	T.sinks = append(T.sinks, sink)
	return sink
}

func (T *Scheduler) NewSource() rob.Source {
	source := newSource()
	T.mu.Lock()
	defer T.mu.Unlock()
	if len(T.sinks) == 0 {
		T.backorder = append(T.backorder, source)
	} else {
		idx := rand.Intn(len(T.sinks))
		sink := T.sinks[idx]
		source.assign(sink)
	}
	return source
}

var _ rob.Scheduler = (*Scheduler)(nil)
