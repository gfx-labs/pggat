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

func (T *Scheduler) AddSink(constraints rob.Constraints, worker rob.Worker) {
	T.pool.AddSink(sink.NewSink(constraints, worker))
}

func (T *Scheduler) NewSource() rob.Worker {
	return source.MakeSource(&T.pool)
}

var _ rob.Scheduler = (*Scheduler)(nil)
