package recipepool

import (
	"sync"
	"time"

	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/gat/pool/serverpool"

	"gfx.cafe/gfx/pggat/lib/gat/metrics"
	"gfx.cafe/gfx/pggat/lib/gat/pool"
	"gfx.cafe/gfx/pggat/lib/util/slices"
)

type Pool struct {
	config Config

	servers serverpool.Pool

	recipes          map[string]*Recipe
	recipeScaleOrder slices.Sorted[string]
	mu               sync.RWMutex
}

func MakePool(config Config) Pool {
	return Pool{
		config: config,

		servers: serverpool.MakePool(config.Config),
	}
}

func (T *Pool) scaleUpL0() (string, *Recipe) {
	T.mu.RLock()
	defer T.mu.RUnlock()

	for _, name := range T.recipeScaleOrder {
		r := T.recipes[name]
		if r.r.Allocate() {
			return name, r
		}
	}

	return "", nil
}

func (T *Pool) scaleUpL1(name string, r *Recipe) *pool.Conn {
	if r == nil {
		return nil
	}

	return T.scaleUpL2(name, r, r.Dial())
}

func (T *Pool) scaleUpL2(name string, r *Recipe, conn *pool.Conn) *pool.Conn {
	if conn == nil {
		r.r.Free()
		return nil
	}

	T.mu.Lock()
	defer T.mu.Unlock()
	if T.recipes[name] != r {
		// recipe was removed
		r.r.Free()
		return nil
	}

	r.servers = append(r.servers, conn)
	// update order
	T.recipeScaleOrder.Update(slices.Index(T.recipeScaleOrder, name), func(n string) int {
		return len(T.recipes[n].servers)
	})
	return nil
}

// ScaleUp will attempt to allocate a new server connection. Returns whether the operation was successful.
func (T *Pool) ScaleUp() bool {
	conn := T.scaleUpL1(T.scaleUpL0())
	if conn == nil {
		return false
	}

	T.servers.AddServer(conn)
	return true
}

func (T *Pool) ScaleDown(idleFor time.Duration) time.Duration {
	return T.servers.ScaleDown(idleFor)
}

func (T *Pool) AddClient(client *pool.Conn) {
	T.servers.AddClient(client)
}

func (T *Pool) RemoveClient(client *pool.Conn) {
	T.servers.RemoveClient(client)
}

func (T *Pool) removeRecipe(name string) *Recipe {
	r, ok := T.recipes[name]
	if !ok {
		return nil
	}
	delete(T.recipes, name)
	T.recipeScaleOrder = slices.Delete(T.recipeScaleOrder, name)

	return r
}

func (T *Pool) addRecipe(name string, r *Recipe) {
	if T.recipes == nil {
		T.recipes = make(map[string]*Recipe)
	}
	T.recipes[name] = r

	// insert
	T.recipeScaleOrder = T.recipeScaleOrder.Insert(name, func(n string) int {
		return len(T.recipes[name].servers)
	})
}

func (T *Pool) AddRecipe(name string, r *pool.Recipe) {
	added := NewRecipe(T.config.ParameterStatusSync, T.config.ExtendedQuerySync, r)

	for _, server := range added.servers {
		T.servers.AddServer(server)
	}

	var removed *Recipe

	func() {
		T.mu.Lock()
		defer T.mu.Unlock()

		removed = T.removeRecipe(name)
		T.addRecipe(name, added)
	}()

	for _, server := range removed.servers {
		T.servers.RemoveServer(server)
	}
}

func (T *Pool) RemoveRecipe(name string) {
	var removed *Recipe

	func() {
		T.mu.Lock()
		defer T.mu.Unlock()

		removed = T.removeRecipe(name)
	}()

	for _, server := range removed.servers {
		T.servers.RemoveServer(server)
	}
}

func (T *Pool) RemoveServer(server *pool.Conn) {
	T.servers.RemoveServer(server)

	// update recipe
	T.mu.Lock()
	defer T.mu.Unlock()
	r, ok := T.recipes[server.Recipe]
	if !ok {
		return
	}
	r.RemoveServer(server)
}

func (T *Pool) Acquire(client *pool.Conn, mode gat.SyncMode) (server *pool.Conn) {
	return T.servers.Acquire(client, mode)
}

func (T *Pool) Release(server *pool.Conn) {
	T.servers.Release(server)
}

func (T *Pool) ReadMetrics(m *metrics.Pool) {
	T.servers.ReadMetrics(m)
}

func (T *Pool) Cancel(server *pool.Conn) {
	T.mu.RLock()
	defer T.mu.RUnlock()

	r := T.recipes[server.Recipe]
	if r == nil {
		return
	}

	r.Cancel(server.Conn.BackendKey)
}

func (T *Pool) Close() {
	T.servers.Close()

	T.mu.Lock()
	defer T.mu.Unlock()
	clear(T.recipes)
}
