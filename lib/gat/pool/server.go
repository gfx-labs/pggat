package pool

import (
	"sync"
	"time"

	"github.com/google/uuid"

	"pggat2/lib/fed"
	"pggat2/lib/middleware/middlewares/eqp"
	"pggat2/lib/middleware/middlewares/ps"
	"pggat2/lib/util/strutil"
)

type Server struct {
	conn              fed.Conn
	backendKey        [8]byte
	initialParameters map[strutil.CIString]string

	psServer  *ps.Server
	eqpServer *eqp.Server

	metrics ServerMetrics
	mu      sync.RWMutex
}

func NewServer(
	conn fed.Conn,
	backendKey [8]byte,
	initialParameters map[strutil.CIString]string,

	psServer *ps.Server,
	eqpServer *eqp.Server,
) *Server {
	return &Server{
		conn:              conn,
		backendKey:        backendKey,
		initialParameters: initialParameters,

		psServer:  psServer,
		eqpServer: eqpServer,

		metrics: MakeServerMetrics(),
	}
}

func (T *Server) GetConn() fed.Conn {
	return T.conn
}

func (T *Server) GetBackendKey() [8]byte {
	return T.backendKey
}

func (T *Server) GetInitialParameters() map[strutil.CIString]string {
	return T.initialParameters
}

func (T *Server) GetPSServer() *ps.Server {
	return T.psServer
}

func (T *Server) GetEQPServer() *eqp.Server {
	return T.eqpServer
}

// SetPeer replaces the peer. Returns the old peer
func (T *Server) SetPeer(peer uuid.UUID) uuid.UUID {
	T.mu.Lock()
	defer T.mu.Unlock()

	old := T.metrics.Peer
	T.metrics.SetPeer(peer)
	return old
}

func (T *Server) GetPeer() uuid.UUID {
	T.mu.RLock()
	defer T.mu.RUnlock()

	return T.metrics.Peer
}

func (T *Server) GetConnection() (uuid.UUID, time.Time) {
	T.mu.RLock()
	defer T.mu.RUnlock()

	return T.metrics.Peer, T.metrics.Since
}

func (T *Server) ReadMetrics(metrics *ServerMetrics) {
	T.mu.RLock()
	defer T.mu.RUnlock()

	panic("TODO(garet)")
}
