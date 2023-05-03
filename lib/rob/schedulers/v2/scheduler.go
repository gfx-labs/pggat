package schedulers

import (
	"github.com/google/uuid"

	"pggat2/lib/rob"
	"pggat2/lib/rob/schedulers/v2/pool"
	"pggat2/lib/rob/schedulers/v2/sink"
	"pggat2/lib/rob/schedulers/v2/source"
)

type Scheduler struct {
	pool pool.Pool
}

func NewScheduler() *Scheduler {
	return &Scheduler{
		pool: pool.MakePool(),
	}
}

func (T *Scheduler) NewSink(fulfills rob.Constraints) rob.Sink {
	id := uuid.New()
	q := T.pool.NewQueue(id, fulfills)
	return sink.NewSink(id, &T.pool, q)
}

func (T *Scheduler) NewSource() rob.Source {
	return source.NewSource(&T.pool)
}

var _ rob.Scheduler = (*Scheduler)(nil)
