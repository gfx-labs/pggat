package rob

import (
	"github.com/google/uuid"

	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool"
	"gfx.cafe/gfx/pggat/lib/rob"
	"gfx.cafe/gfx/pggat/lib/rob/schedulers/v3"
)

type Pooler struct {
	s schedulers.Scheduler
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

func (T *Pooler) Acquire(client uuid.UUID, sync pool.SyncMode) (server uuid.UUID) {
	switch sync {
	case pool.SyncModeBlocking:
		return T.s.Acquire(client, rob.SyncModeBlocking)
	case pool.SyncModeNonBlocking:
		return T.s.Acquire(client, rob.SyncModeNonBlocking)
	default:
		panic("unreachable")
	}
}

func (T *Pooler) Release(server uuid.UUID) {
	T.s.Release(server)
}

func (T *Pooler) Close() {
	T.s.Close()
}

var _ pool.Pooler = (*Pooler)(nil)
