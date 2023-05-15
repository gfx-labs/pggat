package schedulers

import (
	"pggat2/lib/rob"
	"pggat2/lib/rob/schedulers/v0/pool"
	"pggat2/lib/rob/schedulers/v0/sink"
	"pggat2/lib/rob/schedulers/v0/source"
)

type Scheduler struct {
	pool pool.Pool
}

func MakeScheduler() Scheduler {
	return Scheduler{
		pool: pool.MakePool(),
	}
}

func NewScheduler() *Scheduler {
	s := MakeScheduler()
	return &s
}

func (T *Scheduler) AddSink(constraints rob.Constraints, worker rob.Worker) {
	T.pool.AddSink(sink.NewSink(constraints, worker))
}

func (T *Scheduler) NewSource() rob.Worker {
	return source.NewSource(&T.pool)
}

var _ rob.Scheduler = (*Scheduler)(nil)
