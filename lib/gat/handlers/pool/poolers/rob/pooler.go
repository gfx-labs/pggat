package rob

import (
	"time"

	"github.com/google/uuid"

	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool"
	"gfx.cafe/gfx/pggat/lib/rob/schedulers/v3"
)

type Pooler struct {
	s schedulers.Scheduler
}

func NewPooler() *Pooler {
	return &Pooler{
		s: schedulers.MakeScheduler(),
	}
}

func (T *Pooler) AddClient(id uuid.UUID) {
	T.s.AddUser(id)
}

func (T *Pooler) DeleteClient(client uuid.UUID) {
	T.s.DeleteUser(client)
}

func (T *Pooler) AddServer(id uuid.UUID) {
	T.s.AddWorker(id)
}

func (T *Pooler) DeleteServer(server uuid.UUID) {
	T.s.DeleteWorker(server)
}

func (T *Pooler) Acquire(client uuid.UUID, timeout time.Duration) (server uuid.UUID) {
	return T.s.Acquire(client, timeout)
}

func (T *Pooler) Release(server uuid.UUID) {
	T.s.Release(server)
}

func (T *Pooler) Waiting() <-chan struct{} {
	return T.s.Waiting()
}

func (T *Pooler) Waiters() int {
	return T.s.Waiters()
}

func (T *Pooler) Close() {
	T.s.Close()
}

var _ pool.Pooler = (*Pooler)(nil)
