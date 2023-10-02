package pool

import (
	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/fed/middlewares/eqp"
	"gfx.cafe/gfx/pggat/lib/fed/middlewares/ps"
	"gfx.cafe/gfx/pggat/lib/fed/middlewares/unterminate"
)

type pooledClient struct {
	pooledConn

	ps  *ps.Client
	eqp *eqp.Client
}

func newClient(
	options Config,
	conn *fed.Conn,
) *pooledClient {
	conn.Middleware = append(
		conn.Middleware,
		unterminate.Unterminate,
	)

	var psClient *ps.Client
	if options.ParameterStatusSync == ParameterStatusSyncDynamic {
		// add ps middleware
		psClient = ps.NewClient(conn.InitialParameters)
		conn.Middleware = append(conn.Middleware, psClient)
	}

	var eqpClient *eqp.Client
	if options.ExtendedQuerySync {
		// add eqp middleware
		eqpClient = eqp.NewClient()
		conn.Middleware = append(conn.Middleware, eqpClient)
	}

	return &pooledClient{
		pooledConn: makeConn(
			conn,
		),
		ps:  psClient,
		eqp: eqpClient,
	}
}

func (T *pooledClient) GetEQP() *eqp.Client {
	return T.eqp
}

func (T *pooledClient) GetPS() *ps.Client {
	return T.ps
}
