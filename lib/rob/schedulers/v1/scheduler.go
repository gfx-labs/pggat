package schedulers

import (
	"time"

	"github.com/google/uuid"

	"pggat2/lib/rob"
	"pggat2/lib/rob/schedulers/v1/pool"
	"pggat2/lib/rob/schedulers/v1/source"
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

func (T *Scheduler) AddWorker(constraints rob.Constraints, worker rob.Worker) uuid.UUID {
	return T.pool.AddWorker(constraints, worker)
}

func (T *Scheduler) GetWorker(id uuid.UUID) rob.Worker {
	return T.pool.GetWorker(id)
}

func (T *Scheduler) GetIdleWorker() (uuid.UUID, time.Time) {
	return T.pool.GetIdleWorker()
}

func (T *Scheduler) RemoveWorker(id uuid.UUID) rob.Worker {
	return T.pool.RemoveWorker(id)
}

func (T *Scheduler) WorkerCount() int {
	return T.pool.WorkerCount()
}

func (T *Scheduler) NewSource() rob.Worker {
	return source.NewSource(&T.pool)
}

func (T *Scheduler) ReadMetrics(metrics *rob.Metrics) {
	T.pool.ReadMetrics(metrics)
}

var _ rob.Scheduler = (*Scheduler)(nil)
