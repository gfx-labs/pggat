package spool

import (
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

	oven kitchen.Oven

	serversByID   map[uuid.UUID]*Server
	serversByConn map[*fed.Conn]*Server
	mu            sync.RWMutex
}

// MakePool will create a new pool with config. ScaleLoop must be called if this is used instead of NewPool
func MakePool(config Config) Pool {
	pooler := config.PoolerFactory.NewPooler()
	return Pool{
		config: config,
		pooler: pooler,

		closed: make(chan struct{}),

		oven: kitchen.MakeOven(kitchen.Config{
			Critics: config.Critics,
			Logger:  config.Logger,
		}),
	}
}

func NewPool(config Config) *Pool {
	p := MakePool(config)
	go p.ScaleLoop()
	return &p
}

func (T *Pool) addServer(conn *fed.Conn) {
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
}

func (T *Pool) AddRecipe(name string, recipe *pool.Recipe) {
	removed, added := T.oven.Learn(name, recipe)
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

func (T *Pool) RemoveRecipe(name string) {
	servers := T.oven.Forget(name)
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
	return T.oven.Empty()
}

func (T *Pool) ScaleUp() error {
	server, err := T.oven.Cook()
	if err != nil {
		return err
	}

	T.mu.Lock()
	defer T.mu.Unlock()

	T.addServer(server)

	return nil
}

func (T *Pool) ScaleDown(now time.Time) time.Duration {
	T.mu.Lock()
	defer T.mu.Unlock()

	var m time.Duration

	for _, s := range T.serversByID {
		since, state, _ := s.GetState()

		if state != metrics.ConnStateIdle {
			continue
		}

		idle := now.Sub(since)
		if idle > T.config.IdleTimeout {
			// try to free
			if T.oven.Ignite(s.Conn) {
				delete(T.serversByID, s.ID)
				delete(T.serversByConn, s.Conn)
			}
		} else if idle > m {
			m = idle
		}
	}

	return T.config.IdleTimeout - m
}

func (T *Pool) ScaleLoop() {
	var idle *time.Timer
	defer func() {
		if idle != nil {
			idle.Stop()
		}
	}()
	var idleC <-chan time.Time
	if T.config.IdleTimeout != 0 {
		idle = time.NewTimer(T.config.IdleTimeout)
		idleC = idle.C
	}

	var backoff *time.Timer
	defer func() {
		if backoff != nil {
			backoff.Stop()
		}
	}()
	var backoffC <-chan time.Time
	var backoffNext time.Duration

	for {
		var pending <-chan struct{}
		if backoffNext == 0 {
			pending = T.pooler.Waiting()
		}

		select {
		case <-T.closed:
			return
		case <-backoffC:
			// scale up
			if err := T.ScaleUp(); err == nil {
				backoffNext = 0
				continue
			}

			backoffNext *= 2
			if T.config.ReconnectMaxTime != 0 && backoffNext > T.config.ReconnectMaxTime {
				backoffNext = T.config.ReconnectMaxTime
			}
			backoff.Reset(backoffNext)
		case <-pending:
			// scale up
			ok := true
			for T.pooler.Waiters() > 0 {
				if err := T.ScaleUp(); err != nil {
					ok = false
					break
				}
			}
			if ok {
				continue
			}

			// backoff
			backoffNext = T.config.ReconnectInitialTime
			if backoffNext != 0 {
				if backoff == nil {
					backoff = time.NewTimer(backoffNext)
					backoffC = backoff.C
				} else {
					backoff.Reset(backoffNext)
				}
			}
		case now := <-idleC:
			// scale down
			idle.Reset(T.ScaleDown(now))
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
		serverID := T.pooler.Acquire(client)
		if serverID == uuid.Nil {
			return nil
		}

		T.mu.RLock()
		c, ok := T.serversByID[serverID]
		T.mu.RUnlock()

		if !ok {
			T.pooler.DeleteServer(serverID)
			continue
		}

		return c
	}
}

func (T *Pool) Release(server *Server) {
	if T.config.ResetQuery != "" {
		server.SetState(metrics.ConnStateRunningResetQuery, uuid.Nil)

		if err, _ := backends.QueryString(server.Conn, nil, T.config.ResetQuery); err != nil {
			T.config.Logger.Error("failed to run reset query", zap.Error(err))
			T.RemoveServer(server)
			return
		}
	}

	T.pooler.Release(server.ID)

	server.SetState(metrics.ConnStateIdle, uuid.Nil)
}

func (T *Pool) RemoveServer(server *Server) {
	T.oven.Burn(server.Conn)

	T.mu.Lock()
	defer T.mu.Unlock()

	delete(T.serversByID, server.ID)
	delete(T.serversByConn, server.Conn)
}

func (T *Pool) Cancel(server *Server) {
	T.oven.Cancel(server.Conn)
}

func (T *Pool) ReadMetrics(m *metrics.Pool) {
	T.mu.RLock()
	defer T.mu.RUnlock()

	if m.Servers == nil {
		m.Servers = make(map[uuid.UUID]metrics.Conn)
	}
	for _, server := range T.serversByID {
		var s metrics.Conn
		server.ReadMetrics(&s)
		m.Servers[server.ID] = s
	}
}

func (T *Pool) Close() {
	close(T.closed)

	T.oven.Close()
	T.pooler.Close()

	T.mu.Lock()
	defer T.mu.Unlock()
	maps.Clear(T.serversByID)
	maps.Clear(T.serversByConn)
}
