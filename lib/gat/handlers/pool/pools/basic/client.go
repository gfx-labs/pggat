package basic

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool/spool"
	"gfx.cafe/gfx/pggat/lib/gat/metrics"
)

type Client struct {
	ID   uuid.UUID
	Conn *fed.Conn

	txnCount atomic.Int64

	lastMetricsRead time.Time
	state           metrics.ConnState
	peer            *spool.Server
	since           time.Time
	util            [metrics.ConnStateCount]time.Duration
	mu              sync.Mutex

	// cancelMu is held while a cancel is forwarded to peer and while peer is detached,
	// so a backend is never released while a cancel for this client can still reach it.
	// Lock order: cancelMu before mu.
	cancelMu sync.Mutex
	// unconfirmed is a peer that may still receive a forwarded cancel.
	unconfirmed *spool.Server
}

func NewClient(conn *fed.Conn) *Client {
	return &Client{
		ID:   uuid.New(),
		Conn: conn,

		state: metrics.ConnStateIdle,
		since: time.Now(),
	}
}

func (T *Client) SetState(state metrics.ConnState, peer *spool.Server) {
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

func (T *Client) GetState() (time.Time, metrics.ConnState, *spool.Server) {
	T.mu.Lock()
	defer T.mu.Unlock()
	return T.since, T.state, T.peer
}

// Cancel forwards a cancel to the current peer and keeps it attached until cancel returns.
// It is a no-op while another cancel or a detach holds the client.
func (T *Client) Cancel(cancel func(peer *spool.Server) error) {
	if !T.cancelMu.TryLock() {
		return
	}
	defer T.cancelMu.Unlock()

	_, _, peer := T.GetState()
	if peer == nil {
		return
	}
	if errors.Is(cancel(peer), pool.ErrCancelUnconfirmed) {
		T.unconfirmed = peer
	}
}

// Detach clears the peer after any in-flight cancel finishes. It reports whether server may
// still receive a cancel and must not be reused, and any error sending buffered output to the client.
func (T *Client) Detach(ctx context.Context, state metrics.ConnState, server *spool.Server) (bool, error) {
	var err error
	if !T.cancelMu.TryLock() {
		// Send the finished response now, since waiting for the cancel can take up to its timeout.
		err = T.Conn.Flush(ctx)
		T.cancelMu.Lock()
	}
	defer T.cancelMu.Unlock()

	T.SetState(state, nil)
	if server == nil || T.unconfirmed != server {
		return false, err
	}
	T.unconfirmed = nil
	return true, err
}

func (T *Client) TransactionComplete() {
	T.txnCount.Add(1)
}

func (T *Client) ReadMetrics(_ context.Context, m *metrics.Conn) {
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
