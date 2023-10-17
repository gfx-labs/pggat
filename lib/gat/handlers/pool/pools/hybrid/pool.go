package hybrid

import (
	"log"
	"time"

	"github.com/google/uuid"

	"gfx.cafe/gfx/pggat/lib/bouncer/bouncers/v2"
	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/fed/middlewares/unterminate"
	packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool/poolers/rob"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool/spool"
	"gfx.cafe/gfx/pggat/lib/gat/metrics"
)

type Pool struct {
	primary spool.Pool
	replica spool.Pool
}

func NewPool() *Pool {
	config := spool.Config{
		PoolerFactory:        new(rob.Factory),
		UsePS:                true,
		UseEQP:               true,
		IdleTimeout:          5 * time.Minute,
		ReconnectInitialTime: 5 * time.Second,
		ReconnectMaxTime:     1 * time.Minute,
	}

	p := &Pool{
		primary: spool.MakePool(config),
		replica: spool.MakePool(config),
	}
	go p.primary.ScaleLoop()
	go p.replica.ScaleLoop()
	return p
}

func (T *Pool) AddReplicaRecipe(name string, recipe *pool.Recipe) {
	T.replica.AddRecipe(name, recipe)
}

func (T *Pool) RemoveReplicaRecipe(name string) {
	T.replica.RemoveRecipe(name)
}

func (T *Pool) AddRecipe(name string, recipe *pool.Recipe) {
	T.primary.AddRecipe(name, recipe)
}

func (T *Pool) RemoveRecipe(name string) {
	T.primary.RemoveRecipe(name)
}

func (T *Pool) Serve(conn *fed.Conn) error {
	m := NewMiddleware()
	conn.Middleware = append(conn.Middleware, unterminate.Unterminate, m)

	id := uuid.New()
	T.primary.AddClient(id)
	defer T.primary.RemoveClient(id)
	T.replica.AddClient(id)
	defer T.replica.RemoveClient(id)

	var err, serverErr error

	var server *spool.Server
	defer func() {
		if server != nil {
			if serverErr != nil {
				T.replica.RemoveServer(server)
			} else {
				T.replica.Release(server)
			}
			server = nil
		}
	}()

	if !conn.Ready {
		// TODO(garet) pair
		enc := packets.ParameterStatus{
			Key:   "client_encoding",
			Value: "UTF-8",
		}
		if err = conn.WritePacket(&enc); err != nil {
			return err
		}

		p := packets.ReadyForQuery('I')
		if err = conn.WritePacket(&p); err != nil {
			return err
		}

		conn.Ready = true
	}

	for {
		if server != nil {
			T.replica.Release(server)
			server = nil
		}

		var packet fed.Packet
		packet, err = conn.ReadPacket(true)
		if err != nil {
			return err
		}

		server = T.replica.Acquire(id)
		if server == nil {
			return pool.ErrClosed
		}

		// TODO(garet) pair
		err, serverErr = bouncers.Bounce(conn, server.Conn, packet)
		if serverErr != nil {
			return serverErr
		}
		if err != nil {
			if err == (ErrReadOnly{}) {
				log.Printf("READ ONLY DETECTED :)")
				log.Printf("buffered: %v", m.buf.Full())
			}
			return err
		}

		m.Reset()
	}
}

func (T *Pool) Cancel(key fed.BackendKey) {
	// TODO implement me
	panic("implement me")
}

func (T *Pool) ReadMetrics(m *metrics.Pool) {
	T.primary.ReadMetrics(m)
	T.replica.ReadMetrics(m)
}

func (T *Pool) Close() {
	T.primary.Close()
	T.replica.Close()
}

var _ pool.Pool = (*Pool)(nil)
var _ pool.ReplicaPool = (*Pool)(nil)
