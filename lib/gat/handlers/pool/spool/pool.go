package spool

import (
	"context"
	"gfx.cafe/gfx/pggat/lib/fed/middlewares/tracing"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"gfx.cafe/gfx/pggat/lib/bouncer/backends/v0"
	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/fed/middlewares/eqp"
	"gfx.cafe/gfx/pggat/lib/fed/middlewares/ps"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool/spool/kitchen"
	"gfx.cafe/gfx/pggat/lib/gat/metrics"
	"gfx.cafe/gfx/pggat/lib/util/maps"
)

type Pool struct {
	config Config
	pooler pool.Pooler

	closed chan struct{}

	chef kitchen.Chef

	serversByID   map[uuid.UUID]*Server
	serversByConn map[*fed.Conn]*Server
	mu            sync.RWMutex

	tracer trace.Tracer
}

// MakePool will create a new pool with config. ScaleLoop must be called if this is used instead of NewPool
func MakePool(config Config) Pool {
	pooler := config.PoolerFactory.NewPooler()
	return Pool{
		config: config,
		pooler: pooler,

		closed: make(chan struct{}),

		chef: kitchen.MakeChef(kitchen.Config{
			Critics: config.Critics,
			Logger:  config.Logger,
		}),
		tracer: otel.Tracer("spool", trace.WithInstrumentationAttributes(
			attribute.String("component", "gfx.cafe/gfx/pggat/lib/gat/handlers/pool/spool/pool.go"),
		)),
	}
}

func NewPool(ctx context.Context, config Config) *Pool {
	p := MakePool(config)
	go p.ScaleLoop(ctx)
	return &p
}

func (T *Pool) addServer(conn *fed.Conn) {
	if T.config.UsePacketTracing {
		conn.Middleware = append(
			conn.Middleware,
			tracing.NewPacketTrace())
	}

	if T.config.UseOtelTracing {
		conn.Middleware = append(
			conn.Middleware,
			tracing.NewOtelTrace())
	}

	if T.config.UsePS {
		conn.Middleware = append(
			conn.Middleware,
			ps.NewServer(conn.InitialParameters),
		)
	}

	if T.config.UseEQP {
		conn.Middleware = append(
			conn.Middleware,
			eqp.NewServer(),
		)
	}

	server := NewServer(conn)

	if T.serversByID == nil {
		T.serversByID = make(map[uuid.UUID]*Server)
	}
	T.serversByID[server.ID] = server

	if T.serversByConn == nil {
		T.serversByConn = make(map[*fed.Conn]*Server)
	}
	T.serversByConn[server.Conn] = server

	T.pooler.AddServer(server.ID)
}

func (T *Pool) removeServer(conn *fed.Conn) {
	server, ok := T.serversByConn[conn]
	if !ok {
		return
	}
	delete(T.serversByConn, conn)
	delete(T.serversByID, server.ID)
	T.pooler.DeleteServer(server.ID)
}

func (T *Pool) AddRecipe(ctx context.Context, name string, recipe *pool.Recipe) {
	removed, added := T.chef.Learn(ctx, name, recipe)
	if len(removed) == 0 && len(added) == 0 {
		return
	}

	T.mu.Lock()
	defer T.mu.Unlock()

	for _, server := range removed {
		T.removeServer(server)
	}

	for _, server := range added {
		T.addServer(server)
	}
}

func (T *Pool) RemoveRecipe(ctx context.Context, name string) {
	servers := T.chef.Forget(ctx, name)
	if len(servers) == 0 {
		return
	}

	T.mu.Lock()
	defer T.mu.Unlock()

	for _, server := range servers {
		T.removeServer(server)
	}
}

func (T *Pool) Empty() bool {
	return T.chef.Empty()
}

func (T *Pool) ScaleUp(ctx context.Context) error {
	server, err := T.chef.Cook(ctx)
	if err != nil {
		return err
	}

	T.mu.Lock()
	defer T.mu.Unlock()

	T.addServer(server)

	return nil
}

func (T *Pool) ScaleDown(ctx context.Context, now time.Time) time.Duration {
	T.mu.Lock()
	defer T.mu.Unlock()

	m := T.config.IdleTimeout

	for _, s := range T.serversByID {
		since, state, _ := s.GetState()

		if state != metrics.ConnStateIdle {
			continue
		}

		idle := now.Sub(since)
		if idle > T.config.IdleTimeout {
			// try to free
			if T.chef.Ignite(ctx, s.Conn) {
				delete(T.serversByID, s.ID)
				delete(T.serversByConn, s.Conn)
				T.pooler.DeleteServer(s.ID)
			}
		} else {
			until := T.config.IdleTimeout - idle
			if idle < m {
				m = until
			}
		}
	}

	return m
}

func (T *Pool) ScaleLoop(ctx context.Context) {
	idle := new(time.Timer)
	if T.config.IdleTimeout != 0 {
		idle = time.NewTimer(T.config.IdleTimeout)
		defer idle.Stop()
	}

	backoff := time.NewTimer(0)
	<-backoff.C
	defer backoff.Stop()

	var backoffAmount time.Duration

	for {
		select {
		case <-T.closed:
			return
		case <-backoff.C:
			if backoffAmount == 0 {
				continue
			}

			ok := true
			for T.pooler.Waiters() > 0 {
				if err := T.ScaleUp(ctx); err != nil {
					ok = false
					break
				}
			}

			if ok {
				backoffAmount = 0
				continue
			}

			// increase backoff
			backoffAmount = min(T.config.ReconnectMaxTime, 2*backoffAmount)
			if backoffAmount != 0 {
				backoff.Reset(backoffAmount)
			}
		case <-T.pooler.Waiting():
			if backoffAmount != 0 {
				// already backing off
				continue
			}

			ok := true
			for T.pooler.Waiters() > 0 {
				if err := T.ScaleUp(ctx); err != nil {
					ok = false
					break
				}
			}

			if ok {
				continue
			}

			// start backoff
			backoffAmount = T.config.ReconnectInitialTime
			if backoffAmount != 0 {
				backoff.Reset(backoffAmount)
			}
		case now := <-idle.C:
			// scale down
			idle.Reset(T.ScaleDown(ctx, now))
		}
	}
}

func (T *Pool) AddClient(client uuid.UUID) {
	T.pooler.AddClient(client)
}

func (T *Pool) RemoveClient(client uuid.UUID) {
	T.pooler.DeleteClient(client)
}

func (T *Pool) Acquire(client uuid.UUID) *Server {
	for {
		serverID := T.pooler.Acquire(client, T.config.AcquireTimeout)
		if serverID == uuid.Nil {
			return nil
		}

		T.mu.RLock()
		c, ok := T.serversByID[serverID]
		if ok {
			c.SetState(metrics.ConnStatePairing, client)
		}
		T.mu.RUnlock()

		if !ok {
			T.pooler.DeleteServer(serverID)
			continue
		}

		return c
	}
}

func (T *Pool) Release(ctx context.Context, server *Server) {
	if T.config.ResetQuery != "" {
		server.SetState(metrics.ConnStateRunningResetQuery, uuid.Nil)

		if err, _ := backends.QueryString(ctx, server.Conn, nil, T.config.ResetQuery); err != nil {
			T.config.Logger.Error("failed to run reset query", zap.Error(err))
			T.RemoveServer(ctx, server)
			return
		}
	}

	T.pooler.Release(server.ID)

	server.SetState(metrics.ConnStateIdle, uuid.Nil)
}

func (T *Pool) RemoveServer(ctx context.Context, server *Server) {
	T.chef.Burn(ctx, server.Conn)
	T.pooler.DeleteServer(server.ID)

	T.mu.Lock()
	defer T.mu.Unlock()

	delete(T.serversByID, server.ID)
	delete(T.serversByConn, server.Conn)
	T.pooler.DeleteServer(server.ID)
}

func (T *Pool) Cancel(ctx context.Context, server *Server) {
	T.chef.Cancel(ctx, server.Conn)
}

func (T *Pool) ReadMetrics(ctx context.Context, m *metrics.Pool) {
	T.mu.RLock()
	defer T.mu.RUnlock()

	if m.Servers == nil {
		m.Servers = make(map[uuid.UUID]metrics.Conn)
	}
	for _, server := range T.serversByID {
		var s metrics.Conn
		server.ReadMetrics(ctx, &s)
		m.Servers[server.ID] = s
	}
}

func (T *Pool) Close(ctx context.Context) {
	close(T.closed)

	T.chef.Close(ctx)
	T.pooler.Close()

	T.mu.Lock()
	defer T.mu.Unlock()
	maps.Clear(T.serversByID)
	maps.Clear(T.serversByConn)
}
