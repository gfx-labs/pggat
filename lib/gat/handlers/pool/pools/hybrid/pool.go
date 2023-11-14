package hybrid

import (
	"sync"

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

func (T *Pool) serveRW(conn *fed.Conn) error {
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
				return pool.ErrClosed
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
				return pool.ErrClosed
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
			replica = T.replica.Acquire(client.ID)
			if replica == nil {
				return pool.ErrClosed
			}

			err, serverErr = T.Pair(client, replica)

			psi.Set(psa)
			eqpi.Set(eqpa)

			if err == nil && serverErr == nil {
				err, serverErr = bouncers.Bounce(conn, replica.Conn, packet)
			}
			if serverErr != nil {
				return serverErr
			} else {
				replica.TransactionComplete()
			}

			// fallback to primary
			if err == (ErrReadOnly{}) {
				m.Primary()

				T.replica.Release(replica)
				replica = nil

				packet, err = conn.ReadPacket(true)
				if err != nil {
					return err
				}

				client.SetState(metrics.ConnStateAwaitingServer, nil, false)

				// acquire primary
				primary = T.primary.Acquire(client.ID)
				if primary == nil {
					return pool.ErrClosed
				}

				serverErr = T.PairPrimary(client, psi, eqpi, primary)

				if serverErr == nil {
					err, serverErr = bouncers.Bounce(conn, primary.Conn, packet)
				}
				if serverErr != nil {
					return serverErr
				} else {
					primary.TransactionComplete()
				}
			}
		} else {
			// straight to primary
			m.Primary()

			packet, err = conn.ReadPacket(true)
			if err != nil {
				return err
			}

			client.SetState(metrics.ConnStateAwaitingServer, nil, false)

			// acquire primary
			primary = T.primary.Acquire(client.ID)
			if primary == nil {
				return pool.ErrClosed
			}

			err, serverErr = T.Pair(client, primary)

			if serverErr == nil {
				err, serverErr = bouncers.Bounce(conn, primary.Conn, packet)
			}
			if serverErr != nil {
				return serverErr
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

func (T *Pool) serveOnly(conn *fed.Conn, write bool) error {
	conn.Middleware = append(
		conn.Middleware,
		unterminate.Unterminate,
		ps.NewClient(conn.InitialParameters),
		eqp.NewClient(),
	)

	client := NewClient(conn)

	T.addClient(client)
	defer T.removeClient(client)

	if write {
		T.primary.AddClient(client.ID)
		defer T.primary.RemoveClient(client.ID)
	} else {
		T.replica.AddClient(client.ID)
		defer T.replica.RemoveClient(client.ID)
	}

	var err, serverErr error

	var server *spool.Server
	defer func() {
		if server != nil {
			if serverErr != nil {
				if write {
					T.primary.RemoveServer(server)
				} else {
					T.replica.RemoveServer(server)
				}
			} else {
				if write {
					T.primary.Release(server)
				} else {
					T.replica.Release(server)
				}
			}
			server = nil
		}
	}()

	if !conn.Ready {
		client.SetState(metrics.ConnStateAwaitingServer, nil, true)

		if write {
			server = T.primary.Acquire(client.ID)
		} else {
			server = T.replica.Acquire(client.ID)
		}
		if server == nil {
			return pool.ErrClosed
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
			if write {
				T.primary.Release(server)
			} else {
				T.replica.Release(server)
			}
			server = nil
		}
		client.SetState(metrics.ConnStateIdle, nil, true)

		var packet fed.Packet
		packet, err = conn.ReadPacket(true)
		if err != nil {
			return err
		}

		client.SetState(metrics.ConnStateAwaitingServer, nil, true)

		if write {
			server = T.primary.Acquire(client.ID)
		} else {
			server = T.replica.Acquire(client.ID)
		}
		if server == nil {
			return pool.ErrClosed
		}

		err, serverErr = T.Pair(client, server)

		if err == nil && serverErr == nil {
			err, serverErr = bouncers.Bounce(conn, server.Conn, packet)
		}
		if serverErr != nil {
			return serverErr
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
	switch conn.InitialParameters[strutil.MakeCIString("hybrid.mode")] {
	case "ro":
		return T.serveOnly(conn, false)
	case "wo":
		return T.serveOnly(conn, true)
	default:
		return T.serveRW(conn)
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
