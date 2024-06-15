package spool

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/gat/metrics"
)

type Server struct {
	ID   uuid.UUID
	Conn fed.Conn

	txnCount atomic.Int64

	lastMetricsRead time.Time
	state           metrics.ConnState
	peer            uuid.UUID
	since           time.Time
	util            [metrics.ConnStateCount]time.Duration
	mu              sync.Mutex
}

func NewServer(conn fed.Conn) *Server {
	return &Server{
		ID:   uuid.New(),
		Conn: conn,

		state: metrics.ConnStateIdle,
		since: time.Now(),
	}
}

func (T *Server) SetState(state metrics.ConnState, peer uuid.UUID) {
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

func (T *Server) GetState() (time.Time, metrics.ConnState, uuid.UUID) {
	T.mu.Lock()
	defer T.mu.Unlock()
	return T.since, T.state, T.peer
}

func (T *Server) TransactionComplete() {
	T.txnCount.Add(1)
}

func (T *Server) ReadMetrics(m *metrics.Conn) {
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

	m.TransactionCount = int(T.txnCount.Swap(0))

	T.lastMetricsRead = now
}
