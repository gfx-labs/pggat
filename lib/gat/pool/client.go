package pool

import (
	"github.com/google/uuid"

	"pggat2/lib/fed"
	"pggat2/lib/middleware"
	"pggat2/lib/middleware/interceptor"
	"pggat2/lib/middleware/middlewares/eqp"
	"pggat2/lib/middleware/middlewares/ps"
	"pggat2/lib/middleware/middlewares/unterminate"
	"pggat2/lib/util/strutil"
)

type Client struct {
	Conn

	ps  *ps.Client
	eqp *eqp.Client
}

func NewClient(
	options Options,
	conn fed.Conn,
	initialParameters map[strutil.CIString]string,
	backendKey [8]byte,
) *Client {
	middlewares := []middleware.Middleware{
		unterminate.Unterminate,
	}

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

	return &Client{
		Conn: MakeConn(
			uuid.New(),
			conn,
			initialParameters,
			backendKey,
		),
		ps:  psClient,
		eqp: eqpClient,
	}
}

func (T *Client) GetEQP() *eqp.Client {
	return T.eqp
}

func (T *Client) GetPS() *ps.Client {
	return T.ps
}
