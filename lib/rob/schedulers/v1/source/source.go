package source

import (
	"time"

	"github.com/google/uuid"

	"pggat2/lib/rob"
	"pggat2/lib/rob/schedulers/v1/pool"
	"pggat2/lib/rob/schedulers/v1/pool/job"
	"pggat2/lib/util/chans"
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

func (T *Source) Do(ctx *rob.Context, work any) {
	ctx.Reset()

	base := job.Base{
		Created: time.Now(),
		ID:      uuid.New(),
		Source:  T.id,
		Context: ctx,
	}
	if T.pool.ExecuteConcurrent(job.Concurrent{
		Base: base,
		Work: work,
	}) {
		return
	}

	if ctx.OnWait != nil {
		chans.TrySend(ctx.OnWait, struct{}{})
	}

	out, ok := T.stall.Get()
	if !ok {
		out = make(chan uuid.UUID, 1)
	}
	defer T.stall.Put(out)

	T.pool.ExecuteStalled(job.Stalled{
		Base:  base,
		Ready: out,
	})
	worker := <-out
	T.pool.Execute(worker, ctx, work)
	return
}

var _ rob.Worker = (*Source)(nil)
