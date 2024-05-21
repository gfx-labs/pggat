package pool

import (
	"time"

	"github.com/google/uuid"
)

type Pooler interface {
	AddClient(id uuid.UUID)
	DeleteClient(client uuid.UUID)

	AddServer(id uuid.UUID)
	DeleteServer(server uuid.UUID)

	Acquire(client uuid.UUID, timeout time.Duration) (server uuid.UUID)
	Release(server uuid.UUID)

	// Waiting is signalled when a client begins waiting
	Waiting() <-chan struct{}
	// Waiters returns the number of waiters
	Waiters() int

	Close()
}

type PoolerFactory interface {
	NewPooler() Pooler
}
