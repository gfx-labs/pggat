package source

import (
	"github.com/google/uuid"
	"pggat2/lib/rob"
	"pggat2/lib/rob/schedulers/v0/job"
	"pggat2/lib/rob/schedulers/v0/pool"
)

type Source struct {
	id    uuid.UUID
	stall chan rob.Worker
	pool  *pool.Pool
}

func MakeSource(p *pool.Pool) Source {
	return Source{
		id:   uuid.New(),
		pool: p,
	}
}

func (T Source) Do(constraints rob.Constraints, work any) {
	T.pool.Do(job.Job{
		Source:      T.id,
		Constraints: constraints,
		Work:        work,
	})
}

var _ rob.Worker = Source{}
