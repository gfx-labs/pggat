package pool

import (
	"github.com/google/uuid"
	"pggat2/lib/fed"
	"sync"
	"time"
)

type Client struct {
	conn       fed.Conn
	backendKey [8]byte

	metrics ClientMetrics
	mu      sync.RWMutex
}

func NewClient(
	conn fed.Conn,
	backendKey [8]byte,
) *Client {
	return &Client{
		conn:       conn,
		backendKey: backendKey,

		metrics: MakeClientMetrics(),
	}
}

func (T *Client) GetConn() fed.Conn {
	return T.conn
}

func (T *Client) GetBackendKey() [8]byte {
	return T.backendKey
}

// SetPeer replaces the peer. Returns the old peer
func (T *Client) SetPeer(peer uuid.UUID) uuid.UUID {
	T.mu.Lock()
	defer T.mu.Unlock()

	old := T.metrics.Peer
	T.metrics.SetPeer(peer)
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

func (T *Client) ReadMetrics(metrics *ClientMetrics) {
	T.mu.RLock()
	defer T.mu.RUnlock()

	panic("TODO(garet)")
}
