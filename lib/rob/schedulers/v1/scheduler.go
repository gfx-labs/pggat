package schedulers

import (
	"math/rand"
	"sync"

	"pggat2/lib/rob"
)

type Scheduler struct {
	sinks     []*Sink
	backorder []*Source

	mu sync.RWMutex
}

func NewScheduler() *Scheduler {
	return new(Scheduler)
}

func (T *Scheduler) NewSink(constraints rob.Constraints) rob.Sink {
	sink := newSink(T, constraints)

	T.mu.Lock()
	defer T.mu.Unlock()

	T.sinks = append(T.sinks, sink)
	for _, source := range T.backorder {
		sink.assign(source)
	}
	T.backorder = T.backorder[:0]

	return sink
}

func (T *Scheduler) NewSource() rob.Source {
	source := newSource()

	T.mu.Lock()
	defer T.mu.Unlock()

	if len(T.sinks) != 0 {
		sink := T.sinks[rand.Intn(len(T.sinks))]
		sink.assign(source)
	} else {
		T.backorder = append(T.backorder, source)
	}

	return source
}

func (T *Scheduler) steal(ignore *Sink) *Source {
	T.mu.RLock()
	defer T.mu.RUnlock()

	for _, sink := range T.sinks {
		if sink == ignore {
			continue
		}
		if source := sink.steal(); source != nil {
			return source
		}
	}
	return nil
}

var _ rob.Scheduler = (*Scheduler)(nil)
var _ stealer = (*Scheduler)(nil)
