package pool

import (
	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/middleware"
	"gfx.cafe/gfx/pggat/lib/middleware/interceptor"
	"gfx.cafe/gfx/pggat/lib/middleware/middlewares/eqp"
	"gfx.cafe/gfx/pggat/lib/middleware/middlewares/ps"
	"gfx.cafe/gfx/pggat/lib/middleware/middlewares/unterminate"
)

type pooledClient struct {
	pooledConn

	ps  *ps.Client
	eqp *eqp.Client
}

func newClient(
	options Options,
	conn fed.Conn,
	backendKey [8]byte,
) *pooledClient {
	middlewares := []middleware.Middleware{
		unterminate.Unterminate,
	}

	initialParameters := conn.InitialParameters()

	var psClient *ps.Client
	if options.ParameterStatusSync == ParameterStatusSyncDynamic {
		// add ps middleware
		psClient = ps.NewClient(initialParameters)
		middlewares = append(middlewares, psClient)
	}

	var eqpClient *eqp.Client
	if options.ExtendedQuerySync {
		// add eqp middleware
		eqpClient = eqp.NewClient()
		middlewares = append(middlewares, eqpClient)
	}

	conn = interceptor.NewInterceptor(
		conn,
		middlewares...,
	)

	return &pooledClient{
		pooledConn: makeConn(
			conn,
			initialParameters,
			backendKey,
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
