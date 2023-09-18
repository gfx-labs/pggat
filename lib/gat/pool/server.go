package pool

import (
	"pggat/lib/fed"
	"pggat/lib/middleware"
	"pggat/lib/middleware/interceptor"
	"pggat/lib/middleware/middlewares/eqp"
	"pggat/lib/middleware/middlewares/ps"
	"pggat/lib/util/strutil"
)

type Server struct {
	Conn

	recipe string

	ps  *ps.Server
	eqp *eqp.Server
}

func NewServer(
	options Options,
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
		Conn: MakeConn(
			conn,
			initialParameters,
			backendKey,
		),
		recipe: recipe,
		ps:     psServer,
		eqp:    eqpServer,
	}
}

func (T *Server) GetRecipe() string {
	return T.recipe
}

func (T *Server) GetEQP() *eqp.Server {
	return T.eqp
}

func (T *Server) GetPS() *ps.Server {
	return T.ps
}
