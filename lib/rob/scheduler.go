package rob

import (
	"github.com/google/uuid"
)

type SyncMode int

const (
	// SyncModeNonBlocking will attempt to acquire a worker without blocking
	SyncModeNonBlocking SyncMode = iota
	// SyncModeBlocking will block to acquire a worker
	SyncModeBlocking
	// SyncModeTryNonBlocking will attempt to acquire without blocking first, then fallback to blocking if none were available
	SyncModeTryNonBlocking
)

type Scheduler interface {
	AddWorker(id uuid.UUID)
	DeleteWorker(worker uuid.UUID)

	AddUser(id uuid.UUID)
	DeleteUser(user uuid.UUID)

	// Acquire will acquire a worker with the desired SyncMode
	Acquire(user uuid.UUID, sync SyncMode) uuid.UUID

	// Release will release a worker.
	// This should be called after acquire unless the worker is removed with RemoveWorker
	Release(worker uuid.UUID)

	Close()
}
