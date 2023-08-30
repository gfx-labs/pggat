package transaction

import (
	"github.com/google/uuid"

	"pggat2/lib/gat/pool"
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

func (T *Pooler) AcquireConcurrent(client uuid.UUID) uuid.UUID {
	return T.s.AcquireConcurrent(client)
}

func (T *Pooler) AcquireAsync(client uuid.UUID) uuid.UUID {
	return T.s.AcquireAsync(client)
}

func (*Pooler) ReleaseAfterTransaction() bool {
	return true
}

func (T *Pooler) Release(server uuid.UUID) {
	T.s.Release(server)
}

var _ pool.Pooler = (*Pooler)(nil)
