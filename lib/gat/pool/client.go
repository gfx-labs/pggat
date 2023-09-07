package pool

import (
	"sync/atomic"

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
	id uuid.UUID

	conn fed.Conn

	ps  *ps.Client
	eqp *eqp.Client

	initialParameters map[strutil.CIString]string
	backendKey        [8]byte

	transactionCount atomic.Int64
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
		id:         uuid.New(),
		conn:       conn,
		ps:         psClient,
		eqp:        eqpClient,
		backendKey: backendKey,
	}
}

func (T *Client) GetID() uuid.UUID {
	return T.id
}

func (T *Client) GetConn() fed.Conn {
	return T.conn
}

func (T *Client) GetEQP() *eqp.Client {
	return T.eqp
}

func (T *Client) GetPS() *ps.Client {
	return T.ps
}

func (T *Client) TransactionComplete() {
	T.transactionCount.Add(1)
}

func (T *Client) GetInitialParameters() map[strutil.CIString]string {
	return T.initialParameters
}

func (T *Client) SetState(state State, peer uuid.UUID) {

}
