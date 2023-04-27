package v0

import (
	"math/rand"

	"pggat2/lib/rob"
)

type Scheduler struct {
	sinks   []*Sink
	sources []*Source

	backOrders []*work
}

func NewScheduler() *Scheduler {
	return &Scheduler{}
}

func (T *Scheduler) NewSink() rob.Sink {
	sink := newSink(T)
	T.sinks = append(T.sinks, sink)
	for _, backOrder := range T.backOrders {
		sink.enqueue(backOrder)
	}
	T.backOrders = T.backOrders[:0]
	return sink
}

func (T *Scheduler) NewSource() rob.Source {
	source := newSource(T)
	T.sources = append(T.sources, source)
	return source
}

func (T *Scheduler) getSink() *Sink {
	if len(T.sinks) == 0 {
		return nil
	}
	return T.sinks[rand.Intn(len(T.sinks))]
}

func (T *Scheduler) backOrder(w *work) {
	T.backOrders = append(T.backOrders, w)
}

var _ rob.Scheduler = (*Scheduler)(nil)
