package pool

import (
	"time"

	"github.com/google/uuid"
)

type Metrics struct {
	Servers map[uuid.UUID]ServerMetrics
	Clients map[uuid.UUID]ClientMetrics
}

type ItemMetrics struct {
	// Peer is the currently connected server or client. If uuid.Nil, there is no connection
	Peer uuid.UUID
	// Since is the last time that Peer changed.
	Since time.Time

	// Idle is how long (since the last metrics read) this has been idle (Peer == uuid.Nil)
	Idle time.Duration
	// Active is how long (since the last metrics read) this has been active (Peer != uuid.Nil)
	Active time.Duration

	// Transactions is the number of handled transactions since last metrics reset
	Transactions int
}

func MakeItemMetrics() ItemMetrics {
	return ItemMetrics{
		Since: time.Now(),
	}
}

func (T *ItemMetrics) SetPeer(peer uuid.UUID) {
	now := time.Now()
	if T.Peer == uuid.Nil {
		T.Idle += now.Sub(T.Since)
	} else {
		T.Active += now.Sub(T.Since)
	}

	T.Peer = peer
	T.Since = now
}

type ServerMetrics struct {
	ItemMetrics
}

func MakeServerMetrics() ServerMetrics {
	return ServerMetrics{
		ItemMetrics: MakeItemMetrics(),
	}
}

type ClientMetrics struct {
	ItemMetrics

	// Stalled is the time the client started stalling (because it couldn't find a server)
	// If not stalling, this will be the zero value of time.Time
	Stalled time.Time
}

func MakeClientMetrics() ClientMetrics {
	return ClientMetrics{
		ItemMetrics: MakeItemMetrics(),
	}
}
