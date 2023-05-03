package sink

import (
	"github.com/google/uuid"

	"pggat2/lib/rob"
	"pggat2/lib/rob/schedulers/v2/pool"
	"pggat2/lib/rob/schedulers/v2/queue"
)

type Sink struct {
	id    uuid.UUID
	pool  *pool.Pool
	queue *queue.Queue
}

func NewSink(id uuid.UUID, p *pool.Pool, q *queue.Queue) *Sink {
	return &Sink{
		id:    id,
		pool:  p,
		queue: q,
	}
}

func (T *Sink) findWork() {
	T.pool.StealFor(T.id)
	// see if we stole some work
	if T.queue.HasWork() {
		return
	}
	// there is no work to steal, wait until some more is scheduled
	<-T.queue.Ready()
}

func (T *Sink) Read() any {
	for {
		v, ok := T.queue.Read()
		if ok {
			return v.Work
		}
		T.findWork()
	}
}

var _ rob.Sink = (*Sink)(nil)
