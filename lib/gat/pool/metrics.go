package pool

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"pggat2/lib/util/maps"
)

var Stalling = uuid.UUID{1}

type Metrics struct {
	Servers map[uuid.UUID]ItemMetrics
	Clients map[uuid.UUID]ItemMetrics
}

func (T *Metrics) TransactionCount() int {
	var serverTransactions int
	var clientTransactions int

	for _, server := range T.Servers {
		serverTransactions += server.Transactions
	}

	for _, client := range T.Clients {
		clientTransactions += client.Transactions
	}

	if clientTransactions > serverTransactions {
		return clientTransactions
	}
	return serverTransactions
}

func (T *Metrics) ServerStateCount() (active, idle, stalling int) {
	for _, server := range T.Servers {
		switch server.Peer {
		case uuid.Nil:
			idle++
		case Stalling:
			stalling++
		default:
			active++
		}
	}
	return
}

func (T *Metrics) ServerStateUtil() (active, idle, stalling float64) {
	var totalUtil time.Duration
	var activeUtil time.Duration
	var idleUtil time.Duration
	var stallingUtil time.Duration
	for _, server := range T.Servers {
		totalUtil += server.Idle + server.Stalled + server.Active
		activeUtil += server.Active
		idleUtil += server.Idle
		stallingUtil += server.Stalled
	}

	active = float64(activeUtil) / float64(totalUtil)
	idle = float64(idleUtil) / float64(totalUtil)
	stalling = float64(stallingUtil) / float64(totalUtil)
	return
}

func (T *Metrics) ClientStateCount() (active, idle, stalling int) {
	for _, client := range T.Clients {
		switch client.Peer {
		case uuid.Nil:
			idle++
		case Stalling:
			stalling++
		default:
			active++
		}
	}
	return
}

func (T *Metrics) ClientStateUtil() (active, idle, stalling float64) {
	var totalUtil time.Duration
	var activeUtil time.Duration
	var idleUtil time.Duration
	var stallingUtil time.Duration
	for _, client := range T.Clients {
		totalUtil += client.Idle + client.Stalled + client.Active
		activeUtil += client.Active
		idleUtil += client.Idle
		stallingUtil += client.Stalled
	}

	active = float64(activeUtil) / float64(totalUtil)
	idle = float64(idleUtil) / float64(totalUtil)
	stalling = float64(stallingUtil) / float64(totalUtil)
	return
}

func (T *Metrics) Clear() {
	maps.Clear(T.Servers)
	maps.Clear(T.Clients)
}

func (T *Metrics) String() string {
	serverActive, serverIdle, serverStalling := T.ServerStateCount()
	serverActiveUtil, serverIdleUtil, serverStallingUtil := T.ServerStateUtil()
	clientActive, clientIdle, clientStalling := T.ClientStateCount()
	clientActiveUtil, clientIdleUtil, clientStallingUtil := T.ClientStateUtil()
	return fmt.Sprintf("%d transactions | %d servers (%d (%.2f%%) active, %d (%.2f%%) idle, %d (%.2f%%) stalling) | %d clients (%d (%.2f%%) active, %d (%.2f%%) idle, %d (%.2f%%) stalling)",
		T.TransactionCount(),
		len(T.Servers),
		serverActive,
		serverActiveUtil*100,
		serverIdle,
		serverIdleUtil*100,
		serverStalling,
		serverStallingUtil*100,
		len(T.Clients),
		clientActive,
		clientActiveUtil*100,
		clientIdle,
		clientIdleUtil*100,
		clientStalling,
		clientStallingUtil*100,
	)
}

type ItemMetrics struct {
	// Time is the time of this metrics read
	Time time.Time

	// Peer is the currently connected server or client. If uuid.Nil, there is no connection. If Stalling, currently stalling
	Peer uuid.UUID
	// Since is the last time that Peer changed.
	Since time.Time

	// Idle is how long (since the last metrics read) this has been idle (Peer == uuid.Nil)
	Idle time.Duration
	// Active is how long (since the last metrics read) this has been active (Peer != uuid.Nil)
	Active time.Duration
	// Stalled is how long (since the last metrics read) has been spent in other states (waiting for a peer, running cleanup queries, etc)
	Stalled time.Duration

	// Transactions is the number of handled transactions since last metrics reset
	Transactions int
}

func MakeItemMetrics() ItemMetrics {
	now := time.Now()

	return ItemMetrics{
		Time:  now,
		Since: now,
	}
}

func (T *ItemMetrics) commitSince(now time.Time) {
	since := now.Sub(T.Since)
	if T.Since.Before(T.Time) {
		since = now.Sub(T.Time)
	}

	switch T.Peer {
	case uuid.Nil:
		T.Idle += since
	case Stalling:
		T.Stalled += since
	default:
		T.Active += since
	}
}

func (T *ItemMetrics) SetPeer(peer uuid.UUID) {
	now := time.Now()

	T.commitSince(now)

	T.Peer = peer
	T.Since = now
}

func (T *ItemMetrics) Read(metrics *ItemMetrics) {
	now := time.Now()

	*metrics = *T

	metrics.commitSince(now)

	T.Time = now
	T.Idle = 0
	T.Active = 0
	T.Stalled = 0
	T.Transactions = 0
}
