package hybrid

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool/spool"
	"gfx.cafe/gfx/pggat/lib/gat/metrics"
)

type Client struct {
	ID   uuid.UUID
	Conn fed.Conn

	txnCount atomic.Int64

	lastMetricsRead time.Time
	state           metrics.ConnState
	peer            *spool.Server
	peerIsReplica   bool
	since           time.Time
	util            [metrics.ConnStateCount]time.Duration
	mu              sync.Mutex
}

func NewClient(conn fed.Conn) *Client {
	return &Client{
		ID:   uuid.New(),
		Conn: conn,

		state: metrics.ConnStateIdle,
		since: time.Now(),
	}
}

func (T *Client) SetState(state metrics.ConnState, peer *spool.Server, replica bool) {
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
	T.peerIsReplica = replica
	T.since = now
}

func (T *Client) GetState() (time.Time, metrics.ConnState, *spool.Server, bool) {
	T.mu.Lock()
	defer T.mu.Unlock()
	return T.since, T.state, T.peer, T.peerIsReplica
}

func (T *Client) TransactionComplete() {
	T.txnCount.Add(1)
}

func (T *Client) ReadMetrics(m *metrics.Conn) {
	T.mu.Lock()
	defer T.mu.Unlock()

	now := time.Now()

	m.Time = now

	m.State = T.state
	if T.peer != nil {
		m.Peer = T.peer.ID
	} else {
		m.Peer = uuid.Nil
	}
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
