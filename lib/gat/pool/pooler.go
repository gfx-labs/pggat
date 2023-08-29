package pool

import "github.com/google/uuid"

type Pooler interface {
	AddClient(client uuid.UUID)
	RemoveClient(client uuid.UUID)

	AddServer(server uuid.UUID)
	RemoveServer(server uuid.UUID)

	// AcquireConcurrent tries to acquire a peer for the client without stalling.
	// Returns uuid.Nil if no peer can be acquired
	AcquireConcurrent(client uuid.UUID) uuid.UUID

	// AcquireAsync will stall until a peer is available.
	AcquireAsync(client uuid.UUID) uuid.UUID

	// ReleaseAfterTransaction queries whether servers should be immediately released after a transaction is completed.
	ReleaseAfterTransaction() bool

	// Release will force release the server.
	// This should be called when the paired client has disconnected, or after CanRelease returns true.
	Release(server uuid.UUID)
}
