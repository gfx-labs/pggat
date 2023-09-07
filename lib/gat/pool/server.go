package pool

import (
	"sync/atomic"

	"github.com/google/uuid"

	"pggat2/lib/fed"
	"pggat2/lib/middleware"
	"pggat2/lib/middleware/interceptor"
	"pggat2/lib/middleware/middlewares/eqp"
	"pggat2/lib/middleware/middlewares/ps"
	"pggat2/lib/util/strutil"
)

type Server struct {
	id     uuid.UUID
	recipe string

	conn fed.Conn

	ps  *ps.Server
	eqp *eqp.Server

	initialParameters map[strutil.CIString]string
	backendKey        [8]byte

	transactionCount atomic.Int64
}

func NewServer(
	options Options,
	id uuid.UUID,
	recipe string,
	conn fed.Conn,
	initialParameters map[strutil.CIString]string,
	backendKey [8]byte,
) *Server {
	var middlewares []middleware.Middleware

	var psServer *ps.Server
	if options.ParameterStatusSync == ParameterStatusSyncDynamic {
		// add ps middleware
		psServer = ps.NewServer(initialParameters)
		middlewares = append(middlewares, psServer)
	}

	var eqpServer *eqp.Server
	if options.ExtendedQuerySync {
		// add eqp middleware
		eqpServer = eqp.NewServer()
		middlewares = append(middlewares, eqpServer)
	}

	if len(middlewares) > 0 {
		conn = interceptor.NewInterceptor(
			conn,
			middlewares...,
		)
	}

	return &Server{
		id:         id,
		recipe:     recipe,
		conn:       conn,
		ps:         psServer,
		eqp:        eqpServer,
		backendKey: backendKey,
	}
}

func (T *Server) GetID() uuid.UUID {
	return T.id
}

func (T *Server) GetRecipe() string {
	return T.recipe
}

func (T *Server) GetConn() fed.Conn {
	return T.conn
}

func (T *Server) GetEQP() *eqp.Server {
	return T.eqp
}

func (T *Server) GetPS() *ps.Server {
	return T.ps
}

func (T *Server) TransactionComplete() {
	T.transactionCount.Add(1)
}

func (T *Server) GetInitialParameters() map[strutil.CIString]string {
	return T.initialParameters
}

func (T *Server) SetState(state State, peer uuid.UUID) {

}
