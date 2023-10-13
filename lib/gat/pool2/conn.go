package pool2

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/gat/metrics"
)

type Conn struct {
	ID   uuid.UUID
	Conn *fed.Conn
	// Recipe that created this conn, optional.
	Recipe string

	// metrics

	txnCount atomic.Int64

	lastMetricsRead time.Time

	state metrics.ConnState
	peer  uuid.UUID
	since time.Time

	util [metrics.ConnStateCount]time.Duration

	mu sync.RWMutex
}

func NewConn(conn *fed.Conn) *Conn {
	return &Conn{
		ID:   uuid.New(),
		Conn: conn,
	}
}

func (T *Conn) TransactionComplete() {
	T.txnCount.Add(1)
}

func (T *Conn) GetState() (metrics.ConnState, time.Time) {
	T.mu.RLock()
	defer T.mu.RUnlock()

	return T.state, T.since
}

func (T *Conn) setState(now time.Time, state metrics.ConnState, peer uuid.UUID) {
	T.mu.Lock()
	defer T.mu.Unlock()

	var dur time.Duration
	if T.since.Before(T.lastMetricsRead) {
		dur = now.Sub(T.lastMetricsRead)
	} else {
		dur = now.Sub(T.since)
	}
	T.util[T.state] += dur

	T.state = state
	T.peer = peer
	T.since = now
}

func SetConnState(state metrics.ConnState, conns ...*Conn) {
	now := time.Now()

	for i, conn := range conns {
		var peer uuid.UUID
		if i == 0 {
			if len(conns) > 1 {
				peer = conns[1].ID
			}
		} else {
			peer = conns[0].ID
		}
		conn.setState(now, state, peer)
	}
}

func (T *Conn) ReadMetrics(m *metrics.Conn) {
	T.mu.Lock()
	defer T.mu.Unlock()

	m.Time = time.Now()

	m.State = T.state
	m.Peer = T.peer
	m.Since = T.since

	m.Utilization = T.util
	T.util = [metrics.ConnStateCount]time.Duration{}

	var dur time.Duration
	if m.Since.Before(T.lastMetricsRead) {
		dur = m.Time.Sub(T.lastMetricsRead)
	} else {
		dur = m.Time.Sub(m.Since)
	}
	m.Utilization[m.State] += dur

	m.TransactionCount = int(T.txnCount.Swap(0))

	T.lastMetricsRead = m.Time
}
