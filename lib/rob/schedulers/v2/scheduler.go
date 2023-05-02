package schedulers

import (
	"pggat2/lib/rob"
	"pggat2/lib/rob/schedulers/v2/pool"
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
	return T.pool.NewSink(fulfills)
}

func (T *Scheduler) NewSource() rob.Source {
	return source.NewSource(&T.pool)
}

var _ rob.Scheduler = (*Scheduler)(nil)
