package pool

import "github.com/google/uuid"

type SyncMode int

const (
	SyncModeNonBlocking SyncMode = iota
	SyncModeBlocking
)

type Pooler interface {
	NewClient() uuid.UUID
	DeleteClient(client uuid.UUID)

	NewServer() uuid.UUID
	DeleteServer(server uuid.UUID)

	Acquire(client uuid.UUID, sync SyncMode) (server uuid.UUID)
	Release(server uuid.UUID)
}
