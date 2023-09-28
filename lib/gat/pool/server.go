package pool

import (
	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/fed/middlewares/eqp"
	"gfx.cafe/gfx/pggat/lib/fed/middlewares/ps"
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
	conn *fed.Conn,
) *pooledServer {
	var psServer *ps.Server
	if options.ParameterStatusSync == ParameterStatusSyncDynamic {
		// add ps middleware
		psServer = ps.NewServer(conn.InitialParameters)
		conn.Middleware = append(conn.Middleware, psServer)
	}

	var eqpServer *eqp.Server
	if options.ExtendedQuerySync {
		// add eqp middleware
		eqpServer = eqp.NewServer()
		conn.Middleware = append(conn.Middleware, eqpServer)
	}

	return &pooledServer{
		pooledConn: makeConn(
			conn,
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
