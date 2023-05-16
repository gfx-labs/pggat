package schedulers

import "pggat2/lib/rob"

type Scheduler struct {
}

func (T *Scheduler) AddSink(constraints rob.Constraints, worker rob.Worker) {
	// TODO implement me
	panic("implement me")
}

func (T *Scheduler) NewSource() rob.Worker {
	// TODO implement me
	panic("implement me")
}

var _ rob.Scheduler = (*Scheduler)(nil)
