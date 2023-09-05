package pool

import (
	"sync"
	"time"

	"github.com/google/uuid"

	"pggat2/lib/fed"
)

type Client struct {
	conn       fed.Conn
	backendKey [8]byte

	metrics ItemMetrics
	mu      sync.RWMutex
}

func NewClient(
	conn fed.Conn,
	backendKey [8]byte,
) *Client {
	return &Client{
		conn:       conn,
		backendKey: backendKey,

		metrics: MakeItemMetrics(),
	}
}

func (T *Client) GetConn() fed.Conn {
	return T.conn
}

func (T *Client) GetBackendKey() [8]byte {
	return T.backendKey
}

// SetState replaces the peer. Returns the old peer
func (T *Client) SetState(state State, peer uuid.UUID) uuid.UUID {
	T.mu.Lock()
	defer T.mu.Unlock()

	old := T.metrics.Peer
	T.metrics.SetState(state, peer)
	return old
}

func (T *Client) GetPeer() uuid.UUID {
	T.mu.RLock()
	defer T.mu.RUnlock()

	return T.metrics.Peer
}

func (T *Client) GetConnection() (uuid.UUID, time.Time) {
	T.mu.RLock()
	defer T.mu.RUnlock()

	return T.metrics.Peer, T.metrics.Since
}

func (T *Client) TransactionComplete() {
	T.mu.Lock()
	defer T.mu.Unlock()

	T.metrics.Transactions++
}

func (T *Client) ReadMetrics(metrics *ItemMetrics) {
	T.mu.Lock()
	defer T.mu.Unlock()

	T.metrics.Read(metrics)
}
