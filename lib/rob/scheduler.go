package rob

import (
	"time"

	"github.com/google/uuid"
)

type SyncMode int

type Scheduler interface {
	AddWorker(id uuid.UUID)
	DeleteWorker(worker uuid.UUID)

	AddUser(id uuid.UUID)
	DeleteUser(user uuid.UUID)

	// Acquire will acquire a worker with timeout
	Acquire(user uuid.UUID, timeout time.Duration) uuid.UUID

	// Release will release a worker.
	// This should be called after acquire unless the worker is removed with RemoveWorker
	Release(worker uuid.UUID)

	Close()
}
