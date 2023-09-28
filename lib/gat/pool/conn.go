package pool

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/gat/metrics"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

type pooledConn struct {
	id uuid.UUID

	conn fed.Conn
	// please someone fix runtime.convI2I
	rw fed.ReadWriter

	// metrics

	transactionCount atomic.Int64

	lastMetricsRead time.Time

	state metrics.ConnState
	peer  uuid.UUID
	since time.Time

	util [metrics.ConnStateCount]time.Duration

	mu sync.RWMutex
}

func makeConn(
	conn fed.Conn,
) pooledConn {
	return pooledConn{
		id:   uuid.New(),
		conn: conn,
		rw:   conn,

		since: time.Now(),
	}
}

func (T *pooledConn) GetID() uuid.UUID {
	return T.id
}

func (T *pooledConn) GetConn() fed.Conn {
	return T.conn
}

// GetReadWriter is the exact same as GetConn but bypasses the runtime.convI2I
func (T *pooledConn) GetReadWriter() fed.ReadWriter {
	return T.rw
}

func (T *pooledConn) GetInitialParameters() map[strutil.CIString]string {
	return T.conn.InitialParameters()
}

func (T *pooledConn) GetBackendKey() [8]byte {
	return T.conn.BackendKey()
}

func (T *pooledConn) TransactionComplete() {
	T.transactionCount.Add(1)
}

func (T *pooledConn) SetState(state metrics.ConnState, peer uuid.UUID) {
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

func (T *pooledConn) GetState() (state metrics.ConnState, peer uuid.UUID, since time.Time) {
	T.mu.RLock()
	defer T.mu.RUnlock()
	state = T.state
	peer = T.peer
	since = T.since
	return
}

func (T *pooledConn) ReadMetrics(m *metrics.Conn) {
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
