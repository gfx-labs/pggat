package transaction

import (
	"github.com/google/uuid"

	"gfx.cafe/gfx/pggat/lib/gat/pool"
	"gfx.cafe/gfx/pggat/lib/rob"
	"gfx.cafe/gfx/pggat/lib/rob/schedulers/v2"
)

type Pooler struct {
	s schedulers.Scheduler
}

func NewPooler() pool.Pooler {
	return new(Pooler)
}

func (T *Pooler) AddClient(client uuid.UUID) {
	T.s.AddUser(client)
}

func (T *Pooler) DeleteClient(client uuid.UUID) {
	T.s.DeleteUser(client)
}

func (T *Pooler) AddServer(server uuid.UUID) {
	T.s.AddWorker(server)
}

func (T *Pooler) DeleteServer(server uuid.UUID) {
	T.s.DeleteWorker(server)
}

func (T *Pooler) Acquire(client uuid.UUID, sync pool.SyncMode) uuid.UUID {
	switch sync {
	case pool.SyncModeNonBlocking:
		return T.s.Acquire(client, rob.SyncModeNonBlocking)
	case pool.SyncModeBlocking:
		return T.s.Acquire(client, rob.SyncModeBlocking)
	default:
		return uuid.Nil
	}
}

func (T *Pooler) Release(server uuid.UUID) {
	T.s.Release(server)
}

func (T *Pooler) Close() {
	T.s.Close()
}

var _ pool.Pooler = (*Pooler)(nil)
