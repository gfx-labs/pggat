package pool

import (
	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/middleware"
	"gfx.cafe/gfx/pggat/lib/middleware/interceptor"
	"gfx.cafe/gfx/pggat/lib/middleware/middlewares/eqp"
	"gfx.cafe/gfx/pggat/lib/middleware/middlewares/ps"
)

type pooledServer struct {
	pooledConn

	recipe string

	ps  *ps.Server
	eqp *eqp.Server
}

func newServer(
	options Config,
	recipe string,
	conn fed.Conn,
	backendKey [8]byte,
) *pooledServer {
	var middlewares []middleware.Middleware

	initialParameters := conn.InitialParameters()

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

	return &pooledServer{
		pooledConn: makeConn(
			conn,
			initialParameters,
			backendKey,
		),
		recipe: recipe,
		ps:     psServer,
		eqp:    eqpServer,
	}
}

func (T *pooledServer) GetRecipe() string {
	return T.recipe
}

func (T *pooledServer) GetEQP() *eqp.Server {
	return T.eqp
}

func (T *pooledServer) GetPS() *ps.Server {
	return T.ps
}
