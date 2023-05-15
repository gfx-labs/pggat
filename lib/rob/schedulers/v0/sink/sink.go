package sink

import (
	"github.com/google/uuid"
	"pggat2/lib/rob"
	"pggat2/lib/rob/schedulers/v0/job"
)

type Sink struct {
	id          uuid.UUID
	constraints rob.Constraints
	worker      rob.Worker
}

func NewSink(constraints rob.Constraints, worker rob.Worker) *Sink {
	return &Sink{
		id:          uuid.New(),
		constraints: constraints,
		worker:      worker,
	}
}

func (T *Sink) ID() uuid.UUID {
	return T.id
}

func (T *Sink) Constraints() rob.Constraints {
	return T.constraints
}

// DoIfIdle will call Do if the target Sink is idle.
// Returns true if the job is complete
func (T *Sink) DoIfIdle(j job.Job) bool {
	if !T.constraints.Satisfies(j.Constraints) {
		return false
	}

	// TODO(garet) check if idle

	T.Do(j)
	return true
}

// Do will do the work if the constraints match
// Returns true if the job is complete
func (T *Sink) Do(j job.Job) bool {
	if !T.constraints.Satisfies(j.Constraints) {
		return false
	}

	// TODO(garet) queue if we are too busy

	T.worker.Do(j.Constraints, j.Work)
	return true
}
