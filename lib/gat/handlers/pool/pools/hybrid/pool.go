package hybrid

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

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

	tracer trace.Tracer
}

func NewPool(ctx context.Context, config Config) *Pool {
	c := config.Spool()

	p := &Pool{
		config: config,

		primary: spool.MakePool(c),
		replica: spool.MakePool(c),
		tracer: otel.Tracer("hybrid-pool", trace.WithInstrumentationAttributes(
			attribute.String("component", "gfx.cafe/gfx/pggat/lib/gat/handlers/pool/pools/hybrid/pool.go"),
		)),
	}
	go p.primary.ScaleLoop(ctx)
	go p.replica.ScaleLoop(ctx)
	return p
}

func (T *Pool) AddReplicaRecipe(ctx context.Context, name string, recipe *pool.Recipe) {
	T.replica.AddRecipe(ctx, name, recipe)
}

func (T *Pool) RemoveReplicaRecipe(ctx context.Context, name string) {
	T.replica.RemoveRecipe(ctx, name)
}

func (T *Pool) AddRecipe(ctx context.Context, name string, recipe *pool.Recipe) {
	T.primary.AddRecipe(ctx, name, recipe)
}

func (T *Pool) RemoveRecipe(ctx context.Context, name string) {
	T.primary.RemoveRecipe(ctx, name)
}

func (T *Pool) Pair(ctx context.Context, client *Client, server *spool.Server) (err, serverErr error) {
	ctx, span := T.tracer.Start(ctx, "Pair")
	defer span.End()

	// returning 2 errors is questionable
	err, serverErr = T.pair(ctx, client, server)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else if serverErr != nil {
		span.RecordError(serverErr)
		span.SetStatus(codes.Error, serverErr.Error())
	}

	return
}

func (T *Pool) pair(ctx context.Context, client *Client, server *spool.Server) (err, serverErr error) {
	client.SetState(metrics.ConnStatePairing, server, true)
	server.SetState(metrics.ConnStatePairing, client.ID)

	err, serverErr = ps.Sync(ctx, T.config.TrackedParameters, client.Conn, server.Conn)

	if err != nil || serverErr != nil {
		return
	}

	serverErr = eqp.Sync(ctx, client.Conn, server.Conn)

	if serverErr != nil {
		return
	}

	client.SetState(metrics.ConnStateActive, server, true)
	server.SetState(metrics.ConnStateActive, client.ID)
	return
}

func (T *Pool) PairPrimary(ctx context.Context, client *Client, psc *ps.Client, eqpc *eqp.Client, server *spool.Server) error {
	server.SetState(metrics.ConnStatePairing, client.ID)

	if err := ps.SyncMiddleware(ctx, T.config.TrackedParameters, psc, server.Conn); err != nil {
		return err
	}

	if err := eqp.SyncMiddleware(ctx, eqpc, server.Conn); err != nil {
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

func (T *Pool) serveRW(ctx context.Context, l prom.PoolHybridLabels, conn *fed.Conn) error {
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
				T.primary.RemoveServer(ctx, primary)
			} else {
				T.primary.Release(ctx, primary)
			}
			primary = nil
		}
		if replica != nil {
			if serverErr != nil {
				T.replica.RemoveServer(ctx, replica)
			} else {
				T.replica.Release(ctx, replica)
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

			err, serverErr = T.Pair(ctx, client, replica)
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

			err, serverErr = T.Pair(ctx, client, primary)
			if serverErr != nil {
				return serverErr
			}
			if err != nil {
				return err
			}
		}

		p := packets.ReadyForQuery('I')
		if err = conn.WritePacket(ctx, &p); err != nil {
			return err
		}

		conn.Ready = true
	}

	for {
		if primary != nil {
			T.primary.Release(ctx, primary)
			primary = nil
		}
		if replica != nil {
			T.replica.Release(ctx, replica)
			replica = nil
		}
		client.SetState(metrics.ConnStateIdle, nil, false)

		var packet fed.Packet
		packet, err = conn.ReadPacket(ctx, true)
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

			err, serverErr = T.Pair(ctx, client, replica)
			dur := time.Since(start)

			psi.Set(ctx, psa)
			eqpi.Set(ctx, eqpa)

			if err == nil && serverErr == nil {
				prom.OperationHybrid.Acquire(l.ToOperation("replica")).Observe(float64(dur) / float64(time.Millisecond))
				start := time.Now()
				err, serverErr = bouncers.Bounce(ctx, conn, replica.Conn, packet)
				if serverErr == nil {
					dur := time.Since(start)
					prom.OperationHybrid.Execution(l.ToOperation("replica")).Observe(float64(dur) / float64(time.Millisecond))
				}
			}
			if serverErr != nil {
				return fmt.Errorf("server error: %w", serverErr)
			} else {
				replica.TransactionComplete()
			}

			// fallback to primary
			if err == (ErrReadOnly{}) {
				m.Primary()

				T.replica.Release(ctx, replica)
				replica = nil

				packet, err = conn.ReadPacket(ctx, true)
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

				serverErr = T.PairPrimary(ctx, client, psi, eqpi, primary)
				dur := time.Since(start)

				if serverErr == nil {
					prom.OperationHybrid.Acquire(l.ToOperation("primary")).Observe(float64(dur) / float64(time.Millisecond))
					start := time.Now()
					err, serverErr = bouncers.Bounce(ctx, conn, primary.Conn, packet)
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

			packet, err = conn.ReadPacket(ctx, true)
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

			err, serverErr = T.Pair(ctx, client, primary)

			dur := time.Since(start)

			if err == nil && serverErr == nil {
				prom.OperationHybrid.Acquire(l.ToOperation("primary")).Observe(float64(dur) / float64(time.Millisecond))
				start := time.Now()
				err, serverErr = bouncers.Bounce(ctx, conn, primary.Conn, packet)
				if serverErr == nil {
					dur := time.Since(start)
					prom.OperationHybrid.Execution(l.ToOperation("primary")).Observe(float64(dur) / float64(time.Millisecond))
				}
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

func (T *Pool) serveOnly(ctx context.Context, l prom.PoolHybridLabels, conn *fed.Conn, write bool) error {
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
				sp.RemoveServer(ctx, server)
			} else {
				sp.Release(ctx, server)
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

		err, serverErr = T.Pair(ctx, client, server)
		if serverErr != nil {
			return serverErr
		}
		if err != nil {
			return err
		}

		p := packets.ReadyForQuery('I')
		if err = conn.WritePacket(ctx, &p); err != nil {
			return err
		}

		conn.Ready = true
	}

	var opL prom.OperationHybridLabels
	if write {
		opL = l.ToOperation("primary")
	} else {
		opL = l.ToOperation("replica")
	}

	for {
		if server != nil {
			sp.Release(ctx, server)
			server = nil
		}
		client.SetState(metrics.ConnStateIdle, nil, true)

		var packet fed.Packet
		packet, err = conn.ReadPacket(ctx, true)
		if err != nil {
			return err
		}

		client.SetState(metrics.ConnStateAwaitingServer, nil, true)

		start := time.Now()
		server = sp.Acquire(client.ID)
		if server == nil {
			return pool.ErrFailedToAcquirePeer
		}
		err, serverErr = T.Pair(ctx, client, server)
		dur := time.Since(start)
		if err == nil && serverErr == nil {
			prom.OperationHybrid.Acquire(opL).Observe(float64(dur) / float64(time.Millisecond))
			start := time.Now()
			err, serverErr = bouncers.Bounce(ctx, conn, server.Conn, packet)
			if serverErr == nil {
				dur := time.Since(start)
				prom.OperationHybrid.Execution(opL).Observe(float64(dur) / float64(time.Millisecond))
			}
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

func (T *Pool) Serve(ctx context.Context, conn *fed.Conn) error {
	ctx, span := T.tracer.Start(ctx, "Serve")
	defer span.End()

	// returning 2 errors is questionable
	err := T.serve(ctx, conn)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	return err
}

func (T *Pool) serve(ctx context.Context, conn *fed.Conn) error {
	labels := prom.PoolHybridLabels{
		Database: conn.Database,
		User:     conn.User,
	}
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
		return T.serveOnly(ctx, labels, conn, false)
	case "wo":
		return T.serveOnly(ctx, labels, conn, true)
	case "rw":
		return T.serveRW(ctx, labels, conn)
	default:
		panic("impossible")
	}
}

func (T *Pool) Cancel(ctx context.Context, key fed.BackendKey) {
	ctx, span := T.tracer.Start(ctx, "Cancel")
	defer span.End()

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
		T.replica.Cancel(ctx, peer)
	} else {
		T.primary.Cancel(ctx, peer)
	}
}

func (T *Pool) ReadMetrics(ctx context.Context, m *metrics.Pool) {
	ctx, span := T.tracer.Start(ctx, "ReadMetrics")
	defer span.End()

	T.primary.ReadMetrics(ctx, m)
	T.replica.ReadMetrics(ctx, m)

	T.mu.RLock()
	defer T.mu.RUnlock()

	if m.Clients == nil {
		m.Clients = make(map[uuid.UUID]metrics.Conn)
	}
	for _, client := range T.clients {
		var c metrics.Conn
		client.ReadMetrics(ctx, &c)
		m.Clients[client.ID] = c
	}
}

func (T *Pool) Close(ctx context.Context) {
	ctx, span := T.tracer.Start(ctx, "Close")
	defer span.End()

	T.primary.Close(ctx)
	T.replica.Close(ctx)
}

var (
	_ pool.Pool        = (*Pool)(nil)
	_ pool.ReplicaPool = (*Pool)(nil)
)
