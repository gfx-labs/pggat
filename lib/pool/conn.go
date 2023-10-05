package pool

import (
	"sync"
	"time"

	"github.com/google/uuid"

	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/gat/metrics"
)

type Conn struct {
	conn *fed.Conn

	state metrics.ConnState
	peer  uuid.UUID
	since time.Time
	mu    sync.RWMutex
}

func NewConn(conn *fed.Conn) *Conn {
	return &Conn{
		conn: conn,
	}
}

func (T *Conn) GetState() (code metrics.ConnState, peer uuid.UUID, since time.Time) {
	T.mu.RLock()
	defer T.mu.RUnlock()

	code, peer, since = T.state, T.peer, T.since
	return
}

func (T *Conn) SetState(code metrics.ConnState, peer uuid.UUID) {
	T.mu.Lock()
	defer T.mu.Unlock()

	T.state, T.peer, T.since = code, peer, time.Now()
}

func (T *Conn) Close() error {
	return T.conn.Close()
}
