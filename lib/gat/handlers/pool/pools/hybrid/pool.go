package hybrid

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"gfx.cafe/gfx/pggat/lib/bouncer/bouncers/v2"
	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/fed/middlewares/eqp"
	"gfx.cafe/gfx/pggat/lib/fed/middlewares/ps"
	"gfx.cafe/gfx/pggat/lib/fed/middlewares/unterminate"
	packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool/spool"
	"gfx.cafe/gfx/pggat/lib/gat/metrics"
	"gfx.cafe/gfx/pggat/lib/instrumentation/prom"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

type Pool struct {
	config Config

	primary spool.Pool
	replica spool.Pool

	clients map[fed.BackendKey]*Client
	mu      sync.RWMutex
}

func NewPool(config Config) *Pool {
	c := config.Spool()

	p := &Pool{
		config: config,

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

func (T *Pool) Pair(client *Client, server *spool.Server) (err, serverErr error) {
	client.SetState(metrics.ConnStatePairing, server, true)
	server.SetState(metrics.ConnStatePairing, client.ID)

	err, serverErr = ps.Sync(T.config.TrackedParameters, client.Conn, server.Conn)

	if err != nil || serverErr != nil {
		return
	}

	serverErr = eqp.Sync(client.Conn, server.Conn)

	if serverErr != nil {
		return
	}

	client.SetState(metrics.ConnStateActive, server, true)
	server.SetState(metrics.ConnStateActive, client.ID)
	return
}

func (T *Pool) PairPrimary(client *Client, psc *ps.Client, eqpc *eqp.Client, server *spool.Server) error {
	server.SetState(metrics.ConnStatePairing, client.ID)

	if err := ps.SyncMiddleware(T.config.TrackedParameters, psc, server.Conn); err != nil {
		return err
	}

	if err := eqp.SyncMiddleware(eqpc, server.Conn); err != nil {
		return err
	}

	client.SetState(metrics.ConnStateActive, server, false)
	server.SetState(metrics.ConnStateActive, client.ID)
	return nil
}

func (T *Pool) addClient(client *Client) {
	T.mu.Lock()
	defer T.mu.Unlock()

	if T.clients == nil {
		T.clients = make(map[fed.BackendKey]*Client)
	}
	T.clients[client.Conn.BackendKey] = client
}

func (T *Pool) removeClient(client *Client) {
	T.mu.Lock()
	defer T.mu.Unlock()

	delete(T.clients, client.Conn.BackendKey)
}

func (T *Pool) serveRW(l prom.PoolHybridLabels, conn *fed.Conn) error {
	m := NewMiddleware()

	eqpa := eqp.NewClient()
	eqpi := eqp.NewClient()
	psa := ps.NewClient(conn.InitialParameters)
	psi := ps.NewClient(nil)

	conn.Middleware = append(
		conn.Middleware,
		unterminate.Unterminate,
		psa,
		eqpa,
		m,
	)

	client := NewClient(conn)

	T.addClient(client)
	defer T.removeClient(client)

	T.primary.AddClient(client.ID)
	defer T.primary.RemoveClient(client.ID)
	T.replica.AddClient(client.ID)
	defer T.replica.RemoveClient(client.ID)

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
		client.SetState(metrics.ConnStateAwaitingServer, nil, false)

		if !T.replica.Empty() {
			replica = T.replica.Acquire(client.ID)
			if replica == nil {
				return pool.ErrFailedToAcquirePeer
			}

			err, serverErr = T.Pair(client, replica)
			if serverErr != nil {
				return serverErr
			}
			if err != nil {
				return err
			}
		} else {
			// pair with primary instead

			primary = T.primary.Acquire(client.ID)
			if primary == nil {
				return pool.ErrFailedToAcquirePeer
			}

			err, serverErr = T.Pair(client, primary)
			if serverErr != nil {
				return serverErr
			}
			if err != nil {
				return err
			}
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
		client.SetState(metrics.ConnStateIdle, nil, false)

		var packet fed.Packet
		packet, err = conn.ReadPacket(true)
		if err != nil {
			return err
		}

		client.SetState(metrics.ConnStateAwaitingServer, nil, false)

		// try replica first (if it isn't empty)
		if !T.replica.Empty() {
			start := time.Now()
			replica = T.replica.Acquire(client.ID)
			if replica == nil {
				return pool.ErrFailedToAcquirePeer
			}

			err, serverErr = T.Pair(client, replica)
			dur := time.Since(start)

			psi.Set(psa)
			eqpi.Set(eqpa)

			if err == nil && serverErr == nil {
				prom.OperationHybrid.Acquire(l.ToOperation("replica")).Observe(float64(dur) / float64(time.Millisecond))
				start := time.Now()
				err, serverErr = bouncers.Bounce(conn, replica.Conn, packet)
				dur := time.Since(start)
				prom.OperationHybrid.Execution(l.ToOperation("replica")).Observe(float64(dur) / float64(time.Millisecond))
			}
			if serverErr != nil {
				return fmt.Errorf("server error: %w", serverErr)
			} else {
				replica.TransactionComplete()
			}

			// fallback to primary
			if err == (ErrReadOnly{}) {
				prom.OperationHybrid.Miss(l.ToOperation("replica")).Inc()
				m.Primary()

				T.replica.Release(replica)
				replica = nil

				packet, err = conn.ReadPacket(true)
				if err != nil {
					return err
				}

				client.SetState(metrics.ConnStateAwaitingServer, nil, false)

				// acquire primary
				start := time.Now()
				primary = T.primary.Acquire(client.ID)
				if primary == nil {
					return pool.ErrFailedToAcquirePeer
				}

				serverErr = T.PairPrimary(client, psi, eqpi, primary)
				dur := time.Since(start)

				if serverErr == nil {
					prom.OperationHybrid.Acquire(l.ToOperation("primary")).Observe(float64(dur) / float64(time.Millisecond))
					start := time.Now()
					err, serverErr = bouncers.Bounce(conn, primary.Conn, packet)
					dur := time.Since(start)
					prom.OperationHybrid.Execution(l.ToOperation("primary")).Observe(float64(dur) / float64(time.Millisecond))
				}
				if serverErr != nil {
					return fmt.Errorf("server error: %w", serverErr)
				} else {
					primary.TransactionComplete()
				}
			} else {
				prom.OperationHybrid.Hit(l.ToOperation("replica")).Inc()
			}
		} else {
			// straight to primary
			m.Primary()

			packet, err = conn.ReadPacket(true)
			if err != nil {
				return err
			}

			client.SetState(metrics.ConnStateAwaitingServer, nil, false)

			start := time.Now()
			// acquire primary
			primary = T.primary.Acquire(client.ID)
			if primary == nil {
				return pool.ErrFailedToAcquirePeer
			}

			err, serverErr = T.Pair(client, primary)

			dur := time.Since(start)

			if err == nil && serverErr == nil {
				prom.OperationHybrid.Acquire(l.ToOperation("primary")).Observe(float64(dur) / float64(time.Millisecond))
				start := time.Now()
				err, serverErr = bouncers.Bounce(conn, primary.Conn, packet)
				dur := time.Since(start)
				prom.OperationHybrid.Execution(l.ToOperation("primary")).Observe(float64(dur) / float64(time.Millisecond))

			}
			if serverErr != nil {
				return fmt.Errorf("server error: %w", serverErr)
			} else {
				primary.TransactionComplete()
			}
		}
		client.TransactionComplete()
		if err != nil {
			return err
		}

		m.Reset()
	}
}

func (T *Pool) serveOnly(l prom.PoolHybridLabels, conn *fed.Conn, write bool) error {
	var sp *spool.Pool
	if write {
		sp = &T.primary
	} else {
		sp = &T.replica
	}

	conn.Middleware = append(
		conn.Middleware,
		unterminate.Unterminate,
		ps.NewClient(conn.InitialParameters),
		eqp.NewClient(),
	)

	client := NewClient(conn)

	T.addClient(client)
	defer T.removeClient(client)

	sp.AddClient(client.ID)
	defer sp.RemoveClient(client.ID)

	var err, serverErr error

	var server *spool.Server
	defer func() {
		if server != nil {
			if serverErr != nil {
				sp.RemoveServer(server)
			} else {
				sp.Release(server)
			}
			server = nil
		}
	}()

	if !conn.Ready {
		client.SetState(metrics.ConnStateAwaitingServer, nil, true)

		server = sp.Acquire(client.ID)
		if server == nil {
			return pool.ErrFailedToAcquirePeer
		}

		err, serverErr = T.Pair(client, server)
		if serverErr != nil {
			return serverErr
		}
		if err != nil {
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
			sp.Release(server)
			server = nil
		}
		client.SetState(metrics.ConnStateIdle, nil, true)

		var packet fed.Packet
		packet, err = conn.ReadPacket(true)
		if err != nil {
			return err
		}

		client.SetState(metrics.ConnStateAwaitingServer, nil, true)

		server = sp.Acquire(client.ID)
		if server == nil {
			return pool.ErrFailedToAcquirePeer
		}

		err, serverErr = T.Pair(client, server)

		if err == nil && serverErr == nil {
			err, serverErr = bouncers.Bounce(conn, server.Conn, packet)
		}
		if serverErr != nil {
			return fmt.Errorf("server error: %w", serverErr)
		} else {
			server.TransactionComplete()
			client.TransactionComplete()
		}

		if err != nil {
			return err
		}
	}
}

func (T *Pool) Serve(conn *fed.Conn) error {
	labels := prom.PoolHybridLabels{}
	switch conn.InitialParameters[strutil.MakeCIString("hybrid.mode")] {
	case "ro":
		labels.Mode = "ro"
	case "wo":
		labels.Mode = "wo"
	default:
		labels.Mode = "rw"
	}
	prom.PoolHybrid.Accepted(labels).Inc()
	prom.PoolHybrid.Current(labels).Inc()
	defer prom.PoolHybrid.Current(labels).Dec()
	switch labels.Mode {
	case "ro":
		return T.serveOnly(labels, conn, false)
	case "wo":
		return T.serveOnly(labels, conn, true)
	case "rw":
		return T.serveRW(labels, conn)
	default:
		panic("impossible")
	}
}

func (T *Pool) Cancel(key fed.BackendKey) {
	peer, replica := func() (*spool.Server, bool) {
		T.mu.RLock()
		defer T.mu.RUnlock()

		c, ok := T.clients[key]
		if !ok {
			return nil, false
		}

		_, _, peer, replica := c.GetState()
		return peer, replica
	}()

	if peer == nil {
		return
	}

	if replica {
		T.replica.Cancel(peer)
	} else {
		T.primary.Cancel(peer)
	}
}

func (T *Pool) ReadMetrics(m *metrics.Pool) {
	T.primary.ReadMetrics(m)
	T.replica.ReadMetrics(m)

	T.mu.RLock()
	defer T.mu.RUnlock()

	if m.Clients == nil {
		m.Clients = make(map[uuid.UUID]metrics.Conn)
	}
	for _, client := range T.clients {
		var c metrics.Conn
		client.ReadMetrics(&c)
		m.Clients[client.ID] = c
	}
}

func (T *Pool) Close() {
	T.primary.Close()
	T.replica.Close()
}

var _ pool.Pool = (*Pool)(nil)
var _ pool.ReplicaPool = (*Pool)(nil)
