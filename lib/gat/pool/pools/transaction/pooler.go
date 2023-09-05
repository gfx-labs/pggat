package transaction

import (
	"github.com/google/uuid"

	"pggat2/lib/gat/pool"
	"pggat2/lib/rob"
	"pggat2/lib/rob/schedulers/v2"
)

type Pooler struct {
	s schedulers.Scheduler
}

func (T *Pooler) NewClient() uuid.UUID {
	return T.s.NewUser()
}

func (T *Pooler) DeleteClient(client uuid.UUID) {
	T.s.DeleteUser(client)
}

func (T *Pooler) NewServer() uuid.UUID {
	return T.s.NewWorker()
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

func (*Pooler) ReleaseAfterTransaction() bool {
	return true
}

func (T *Pooler) Release(server uuid.UUID) {
	T.s.Release(server)
}

var _ pool.Pooler = (*Pooler)(nil)
