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

func (T *Pooler) AddClient(client uuid.UUID) {
	T.s.AddUser(client)
}

func (T *Pooler) RemoveClient(client uuid.UUID) {
	T.s.RemoveUser(client)
}

func (T *Pooler) AddServer(server uuid.UUID) {
	T.s.AddWorker(server)
}

func (T *Pooler) RemoveServer(server uuid.UUID) {
	T.s.RemoveWorker(server)
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
