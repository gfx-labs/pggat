package pool

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
	peer  *Conn
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

func (T *Conn) GetState() (metrics.ConnState, time.Time) {
	T.mu.RLock()
	defer T.mu.RUnlock()

	return T.state, T.since
}

func (T *Conn) GetPeer() *Conn {
	T.mu.RLock()
	defer T.mu.RUnlock()

	return T.peer
}

func (T *Conn) setState(now time.Time, state metrics.ConnState, peer *Conn) {
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
		var peer *Conn
		if i == 0 {
			if len(conns) > 1 {
				peer = conns[1]
			}
		} else {
			peer = conns[0]
		}
		conn.setState(now, state, peer)
	}
}

func ConnTransactionComplete(conns ...*Conn) {
	for _, conn := range conns {
		conn.txnCount.Add(1)
	}
}

func (T *Conn) ReadMetrics(m *metrics.Conn) {
	T.mu.Lock()
	defer T.mu.Unlock()

	m.Time = time.Now()

	m.State = T.state
	if T.peer != nil {
		m.Peer = T.peer.ID
	} else {
		m.Peer = uuid.Nil
	}
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
