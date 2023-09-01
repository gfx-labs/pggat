package pool

import (
	"github.com/google/uuid"
)

type SyncMode int

const (
	// SyncModeNonBlocking will obtain a server without blocking
	SyncModeNonBlocking SyncMode = iota
	// SyncModeBlocking will obtain a server by stalling
	SyncModeBlocking
)

type Pooler interface {
	AddClient(client uuid.UUID)
	RemoveClient(client uuid.UUID)

	AddServer(server uuid.UUID)
	RemoveServer(server uuid.UUID)

	// Acquire a peer with SyncMode
	Acquire(client uuid.UUID, sync SyncMode) uuid.UUID

	// ReleaseAfterTransaction queries whether servers should be immediately released after a transaction is completed.
	ReleaseAfterTransaction() bool

	// Release will force release the server.
	// This should be called when the paired client has disconnected, or after CanRelease returns true.
	Release(server uuid.UUID)
}
