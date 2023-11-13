package spool

import (
	"math"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"gfx.cafe/gfx/pggat/lib/bouncer/backends/v0"
	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/fed/middlewares/eqp"
	"gfx.cafe/gfx/pggat/lib/fed/middlewares/ps"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool"
	"gfx.cafe/gfx/pggat/lib/gat/metrics"
	"gfx.cafe/gfx/pggat/lib/util/slices"
)

type Pool struct {
	config Config
	pooler pool.Pooler

	closed chan struct{}

	recipes          map[string]*Recipe
	recipeScaleOrder []*Recipe
	servers          map[uuid.UUID]*Server
	mu               sync.RWMutex
}

// MakePool will create a new pool with config. ScaleLoop must be called if this is used instead of NewPool
func MakePool(config Config) Pool {
	pooler := config.PoolerFactory.NewPooler()
	return Pool{
		config: config,
		pooler: pooler,

		closed: make(chan struct{}),
	}
}

func NewPool(config Config) *Pool {
	p := MakePool(config)
	go p.ScaleLoop()
	return &p
}

func (T *Pool) removeServer(server *Server, deleteFromRecipe, freeFromRecipe bool) {
	delete(T.servers, server.ID)

	r := server.recipe
	if deleteFromRecipe {
		r.Servers = slices.Delete(r.Servers, server)
	}
	if freeFromRecipe {
		r.Recipe.Free()
	}
}

func (T *Pool) scoreRecipe(recipe *Recipe) error {
	T.mu.RUnlock()
	defer T.mu.RLock()

	now := time.Now()

	if len(T.config.Scorers) >= len(recipe.Scores) {
		// we're fine to access scores because we're the only one that can
		recipe.Scores = slices.Resize(recipe.Scores, len(T.config.Scorers))
	}

	var conn *fed.Conn
	defer func() {
		if conn != nil {
			_ = conn.Close()
		}
	}()
	getConn := func() error {
		if conn != nil {
			return nil
		}

		var err error
		conn, err = recipe.Recipe.Dial()
		return err
	}

	for i, scorer := range T.config.Scorers {
		score := recipe.Scores[i]

		if now.Before(score.Expiration) {
			// score still valid
			continue
		}

		var s = math.MaxInt
		var validity = 5 * time.Second
		err := getConn()
		if err == nil {
			s, validity, err = scorer.Score(conn)
		}
		exp := now.Add(validity)
		if err != nil {
			s = math.MaxInt
		}
		recipe.Scores[i] = RecipeScore{
			Score:      s,
			Expiration: exp,
		}

		if err != nil {
			// on error, we don't need to calculate anymore, it will be max
			return err
		}
	}

	return nil
}

func (T *Pool) scoreRecipes() {
	for _, recipe := range T.recipes {
		if err := T.scoreRecipe(recipe); err != nil {
			T.config.Logger.Error("failed to score recipe", zap.Error(err))
		}
	}
}

func (T *Pool) sortRecipes() {
	T.scoreRecipes()

	sort.Slice(T.recipeScaleOrder, func(i, j int) bool {
		a := T.recipeScaleOrder[i]
		b := T.recipeScaleOrder[j]
		// sort by priority first
		if a.Priority() < b.Priority() {
			return true
		}
		if a.Priority() > b.Priority() {
			return false
		}
		// then sort by number of servers
		return len(a.Servers) < len(b.Servers)
	})
}

func (T *Pool) addRecipe(name string, recipe *pool.Recipe) *Recipe {
	r := NewRecipe(name, recipe)

	if T.recipes == nil {
		T.recipes = make(map[string]*Recipe)
	}
	T.recipes[name] = r
	T.recipeScaleOrder = append(T.recipeScaleOrder, r)

	return r
}

func (T *Pool) removeRecipe(name string) {
	r, ok := T.recipes[name]
	if !ok {
		return
	}
	delete(T.recipes, name)
	T.recipeScaleOrder = slices.Delete(T.recipeScaleOrder, r)

	for _, server := range r.Servers {
		T.removeServer(server, false, true)
	}
}

func (T *Pool) AddRecipe(name string, recipe *pool.Recipe) {
	r := func() *Recipe {
		T.mu.Lock()
		defer T.mu.Unlock()

		T.removeRecipe(name)
		return T.addRecipe(name, recipe)
	}()

	c := r.Recipe.AllocateInitial()
	for i := 0; i < c; i++ {
		T.ScaleUpOnce(r)
	}
}

func (T *Pool) RemoveRecipe(name string) {
	T.mu.Lock()
	defer T.mu.Unlock()

	T.removeRecipe(name)
}

func (T *Pool) scaleUpL0() *Recipe {
	T.sortRecipes()
	for _, recipe := range T.recipeScaleOrder {
		if !recipe.Recipe.Allocate() {
			continue
		}
		return recipe
	}
	return nil
}

func (T *Pool) ScaleUpOnce(recipe *Recipe) bool {
	conn, err := recipe.Recipe.Dial()
	if err != nil {
		T.config.Logger.Error("failed to dial server", zap.Error(err))
		recipe.Recipe.Free()
		return false
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

	server := NewServer(recipe, conn)

	T.mu.Lock()
	defer T.mu.Unlock()
	recipe.Servers = append(recipe.Servers, server)
	if T.servers == nil {
		T.servers = make(map[uuid.UUID]*Server)
	}
	T.servers[server.ID] = server

	T.pooler.AddServer(server.ID)

	return true
}

func (T *Pool) ScaleUp() bool {
	r := func() *Recipe {
		T.mu.RLock()
		defer T.mu.RUnlock()

		return T.scaleUpL0()
	}()

	if r == nil {
		T.config.Logger.Warn("no available recipes to scale up pool")
		return false
	}
	return T.ScaleUpOnce(r)
}

func (T *Pool) ScaleDown(now time.Time) time.Duration {
	T.mu.Lock()
	defer T.mu.Unlock()

	var m time.Duration

	for _, s := range T.servers {
		since, state, _ := s.GetState()

		if state != metrics.ConnStateIdle {
			continue
		}

		idle := now.Sub(since)
		if idle > T.config.IdleTimeout {
			// delete
			if s.recipe.Recipe.TryFree() {
				T.removeServer(s, true, false)
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
			if T.ScaleUp() {
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
				if !T.ScaleUp() {
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
		c, ok := T.servers[serverID]
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
	T.mu.Lock()
	defer T.mu.Unlock()

	T.removeServer(server, true, true)
}

func (T *Pool) Cancel(server *Server) {
	server.recipe.Recipe.Cancel(server.Conn.BackendKey)
}

func (T *Pool) ReadMetrics(m *metrics.Pool) {
	T.mu.RLock()
	defer T.mu.RUnlock()

	if m.Servers == nil {
		m.Servers = make(map[uuid.UUID]metrics.Conn)
	}
	for _, server := range T.servers {
		var s metrics.Conn
		server.ReadMetrics(&s)
		m.Servers[server.ID] = s
	}
}

func (T *Pool) Close() {
	close(T.closed)

	T.pooler.Close()
}
