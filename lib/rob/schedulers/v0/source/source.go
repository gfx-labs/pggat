package source

import (
	"github.com/google/uuid"

	"pggat2/lib/rob"
	"pggat2/lib/rob/schedulers/v0/job"
	"pggat2/lib/rob/schedulers/v0/pool"
	"pggat2/lib/rob/schedulers/v0/sink"
	"pggat2/lib/util/pools"
)

type Source struct {
	id   uuid.UUID
	pool *pool.Pool

	stall pools.Locked[chan any]
}

func NewSource(p *pool.Pool) *Source {
	return &Source{
		id:   uuid.New(),
		pool: p,
	}
}

func (T *Source) Do(constraints rob.Constraints, work any) {
	if T.pool.DoConcurrent(job.Concurrent{
		Source:      T.id,
		Constraints: constraints,
		Work:        work,
	}) {
		return
	}
	out, ok := T.stall.Get()
	if !ok {
		out = make(chan any)
	}
	defer T.stall.Put(out)

	T.pool.DoStalled(job.Stalled{
		Source:      T.id,
		Constraints: constraints,
		Out:         out,
	})
	worker := (<-out).(*sink.Sink)
	if hasMore := worker.Do(constraints, work); !hasMore {
		T.pool.StealFor(worker)
	}
}

var _ rob.Worker = (*Source)(nil)
