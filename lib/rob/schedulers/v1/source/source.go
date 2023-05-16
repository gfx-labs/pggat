package source

import (
	"github.com/google/uuid"

	"pggat2/lib/rob"
	"pggat2/lib/rob/schedulers/v1/pool"
	"pggat2/lib/rob/schedulers/v1/pool/job"
	"pggat2/lib/util/pools"
)

type Source struct {
	id   uuid.UUID
	pool *pool.Pool

	stall pools.Locked[chan uuid.UUID]
}

func NewSource(p *pool.Pool) *Source {
	return &Source{
		id:   uuid.New(),
		pool: p,
	}
}

func (T *Source) Do(constraints rob.Constraints, work any) {
	base := job.Base{
		Source:      T.id,
		Constraints: constraints,
	}
	if T.pool.DoConcurrent(job.Concurrent{
		Base: base,
		Work: work,
	}) {
		return
	}
	out, ok := T.stall.Get()
	if !ok {
		out = make(chan uuid.UUID)
	}
	defer T.stall.Put(out)

	T.pool.DoStalled(job.Stalled{
		Base:  base,
		Ready: out,
	})
	worker := <-out
	T.pool.Do(worker, constraints, work)
}

var _ rob.Worker = (*Source)(nil)
