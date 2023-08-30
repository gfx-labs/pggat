package rob

import (
	"github.com/google/uuid"
)

type Scheduler interface {
	AddWorker(worker uuid.UUID)
	RemoveWorker(worker uuid.UUID)

	AddUser(user uuid.UUID)
	RemoveUser(user uuid.UUID)

	// AcquireConcurrent tries to acquire a peer for the user without stalling.
	// Returns uuid.Nil if no peer can be acquired
	AcquireConcurrent(user uuid.UUID) uuid.UUID
	// AcquireAsync will stall until a peer is available
	AcquireAsync(user uuid.UUID) uuid.UUID

	// Release will release a worker.
	// This should be called after acquire unless the worker is removed with RemoveWorker
	Release(worker uuid.UUID)
}

func Acquire(scheduler Scheduler, user uuid.UUID) uuid.UUID {
	if s := scheduler.AcquireConcurrent(user); s != uuid.Nil {
		return s
	}

	return scheduler.AcquireAsync(user)
}
