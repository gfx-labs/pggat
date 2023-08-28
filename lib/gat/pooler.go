package gat

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

	// CanRelease will check if a server can be released after a transaction.
	// Some poolers (such as session poolers) do not release servers after each transaction.
	// Returns true if Release could be called.
	CanRelease(server uuid.UUID) bool

	// Release will force release the server.
	// This should be called when the paired client has disconnected, or after CanRelease returns true.
	Release(server uuid.UUID)
}
