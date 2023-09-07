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

	state State
	peer  uuid.UUID
	since time.Time
	mu    sync.RWMutex
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

func (T *Conn) SetState(state State, peer uuid.UUID) {
	T.mu.Lock()
	defer T.mu.Unlock()
	T.state = state
	T.peer = peer
	T.since = time.Now()
}

func (T *Conn) GetState() (state State, peer uuid.UUID, since time.Time) {
	T.mu.RLock()
	defer T.mu.Unlock()
	state = T.state
	peer = T.peer
	since = T.since
	return
}

func (T *Conn) ReadMetrics(m *metrics.Conn) {

}
