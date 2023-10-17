package hybrid

import (
	"github.com/google/uuid"

	"gfx.cafe/gfx/pggat/lib/bouncer/bouncers/v2"
	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/fed/middlewares/unterminate"
	packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool/spool"
	"gfx.cafe/gfx/pggat/lib/gat/metrics"
)

type Pool struct {
	primary spool.Pool
	replica spool.Pool
}

func NewPool(config Config) *Pool {
	c := config.Spool()

	p := &Pool{
		primary: spool.MakePool(c),
		replica: spool.MakePool(c),
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

	var primary, replica *spool.Server
	defer func() {
		if primary != nil {
			if serverErr != nil {
				T.primary.RemoveServer(primary)
			} else {
				T.primary.Release(primary)
			}
			primary = nil
		}
		if replica != nil {
			if serverErr != nil {
				T.replica.RemoveServer(replica)
			} else {
				T.replica.Release(replica)
			}
			replica = nil
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
		if primary != nil {
			T.primary.Release(primary)
			primary = nil
		}
		if replica != nil {
			T.replica.Release(replica)
			replica = nil
		}

		var packet fed.Packet
		packet, err = conn.ReadPacket(true)
		if err != nil {
			return err
		}

		replica = T.replica.Acquire(id)
		if replica == nil {
			return pool.ErrClosed
		}

		// TODO(garet) pair

		err, serverErr = bouncers.Bounce(conn, replica.Conn, packet)
		if serverErr != nil {
			return serverErr
		}
		if err == (ErrReadOnly{}) {
			m.Primary()

			// release replica
			if replica != nil {
				T.replica.Release(replica)
				replica = nil
			}

			packet, err = conn.ReadPacket(true)
			if err != nil {
				return err
			}

			// acquire primary
			primary = T.primary.Acquire(id)
			if primary == nil {
				return pool.ErrClosed
			}

			// TODO(garet) get primary in the same state replica was

			err, serverErr = bouncers.Bounce(conn, primary.Conn, packet)
			if serverErr != nil {
				return serverErr
			}
		}
		if err != nil {
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
