package v0

import (
	"github.com/google/uuid"

	"pggat2/lib/rob"
)

type Source struct {
	scheduler *Scheduler

	id        uuid.UUID
	preferred *Sink
}

func newSource(scheduler *Scheduler) *Source {
	return &Source{
		scheduler: scheduler,

		id: uuid.New(),
	}
}

func (T *Source) Schedule(a any) {
	w := newWork(T, a)
	if T.preferred == nil {
		T.preferred = T.scheduler.getSink()
	}
	if T.preferred != nil {
		T.preferred.enqueue(w)
		return
	}
	T.scheduler.backOrder(w)
}

var _ rob.Source = (*Source)(nil)
