package pool

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"pggat2/lib/fed"
	"pggat2/lib/gat/metrics"
	"pggat2/lib/util/strutil"
)

type Conn struct {
	id uuid.UUID

	conn fed.Conn

	initialParameters map[strutil.CIString]string
	backendKey        [8]byte

	// metrics

	transactionCount atomic.Int64

	lastMetricsRead time.Time

	state metrics.ConnState
	peer  uuid.UUID
	since time.Time

	util [metrics.ConnStateCount]time.Duration

	mu sync.RWMutex
}

func MakeConn(
	id uuid.UUID,
	conn fed.Conn,
	initialParameters map[strutil.CIString]string,
	backendKey [8]byte,
) Conn {
	return Conn{
		id:                id,
		conn:              conn,
		initialParameters: initialParameters,
		backendKey:        backendKey,

		since: time.Now(),
	}
}

func (T *Conn) GetID() uuid.UUID {
	return T.id
}

func (T *Conn) GetConn() fed.Conn {
	return T.conn
}

func (T *Conn) GetInitialParameters() map[strutil.CIString]string {
	return T.initialParameters
}

func (T *Conn) GetBackendKey() [8]byte {
	return T.backendKey
}

func (T *Conn) TransactionComplete() {
	T.transactionCount.Add(1)
}

func (T *Conn) SetState(state metrics.ConnState, peer uuid.UUID) {
	T.mu.Lock()
	defer T.mu.Unlock()

	now := time.Now()

	var since time.Duration
	if T.since.Before(T.lastMetricsRead) {
		since = now.Sub(T.lastMetricsRead)
	} else {
		since = now.Sub(T.since)
	}
	T.util[T.state] += since

	T.state = state
	T.peer = peer
	T.since = now
}

func (T *Conn) GetState() (state metrics.ConnState, peer uuid.UUID, since time.Time) {
	T.mu.RLock()
	defer T.mu.Unlock()
	state = T.state
	peer = T.peer
	since = T.since
	return
}

func (T *Conn) ReadMetrics(m *metrics.Conn) {
	T.mu.Lock()
	defer T.mu.Unlock()

	now := time.Now()

	m.Time = now

	m.State = T.state
	m.Peer = T.peer
	m.Since = T.since

	m.Utilization = T.util
	T.util = [metrics.ConnStateCount]time.Duration{}

	var since time.Duration
	if m.Since.Before(T.lastMetricsRead) {
		since = now.Sub(T.lastMetricsRead)
	} else {
		since = now.Sub(m.Since)
	}
	m.Utilization[m.State] += since

	m.TransactionCount = int(T.transactionCount.Swap(0))

	T.lastMetricsRead = now
}
