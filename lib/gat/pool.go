package gat

import (
	"log"
	"sync"
	"time"

	"github.com/google/uuid"

	"pggat2/lib/util/maps"
	"pggat2/lib/util/maths"
	"pggat2/lib/util/strutil"
	"pggat2/lib/zap"
)

type Context struct {
	OnWait chan<- struct{}
}

type RawPool interface {
	Serve(ctx *Context, client zap.ReadWriter, startupParameters map[strutil.CIString]string)

	AddServer(server zap.ReadWriter, startupParameters map[strutil.CIString]string) uuid.UUID
	GetServer(id uuid.UUID) zap.ReadWriter
	RemoveServer(id uuid.UUID) zap.ReadWriter

	ScaleDown(amount int) (remaining int)
	IdleSince() time.Time
}

type BaseRawPoolConfig struct {
	TrackedParameters []strutil.CIString
}

type PoolRecipe struct {
	removed bool
	servers []uuid.UUID
	mu      sync.Mutex

	r Recipe
}

type Pool struct {
	config PoolConfig

	recipes maps.RWLocked[string, *PoolRecipe]

	ctx Context
	raw RawPool
}

type PoolConfig struct {
	// IdleTimeout determines how long idle servers are kept in the pool
	IdleTimeout time.Duration
}

func NewPool(raw RawPool, config PoolConfig) *Pool {
	onWait := make(chan struct{})
	pool := &Pool{
		config: config,
		ctx: Context{
			OnWait: onWait,
		},
		raw: raw,
	}

	go func() {
		for range onWait {
			pool.ScaleUp(1)
		}
	}()

	if config.IdleTimeout != 0 {
		go func() {
			for {
				var wait time.Duration

				now := time.Now()
				idle := pool.IdleSince()
				for now.Sub(idle) > config.IdleTimeout {
					if idle == (time.Time{}) {
						break
					}
					pool.ScaleDown(1)
					idle = pool.IdleSince()
				}

				if idle == (time.Time{}) {
					wait = config.IdleTimeout
				} else {
					wait = now.Sub(idle.Add(config.IdleTimeout))
				}

				time.Sleep(wait)
			}
		}()
	}

	return pool
}

func (T *Pool) _tryAddServers(recipe *PoolRecipe, amount int) (remaining int) {
	remaining = amount

	if recipe.removed {
		return
	}

	j := 0
	for i := 0; i < len(recipe.servers); i++ {
		if T.raw.GetServer(recipe.servers[i]) != nil {
			recipe.servers[j] = recipe.servers[i]
			j++
		}
	}
	recipe.servers = recipe.servers[:j]

	var max = amount
	maxConnections := recipe.r.GetMaxConnections()
	if maxConnections != 0 {
		max = maths.Min(maxConnections-j, max)
	}
	for i := 0; i < max; i++ {
		conn, ps, err := recipe.r.Connect()
		if err != nil {
			log.Printf("error connecting to server: %v", err)
			continue
		}

		id := T.raw.AddServer(conn, ps)
		recipe.servers = append(recipe.servers, id)
		remaining--
	}

	return
}

func (T *Pool) tryAddServers(recipe *PoolRecipe, amount int) (remaining int) {
	recipe.mu.Lock()
	defer recipe.mu.Unlock()

	return T._tryAddServers(recipe, amount)
}

func (T *Pool) addRecipe(recipe *PoolRecipe) {
	recipe.mu.Lock()
	defer recipe.mu.Unlock()

	recipe.removed = false
	min := recipe.r.GetMinConnections() - len(recipe.servers)
	T._tryAddServers(recipe, min)
}

func (T *Pool) removeRecipe(recipe *PoolRecipe) {
	recipe.mu.Lock()
	defer recipe.mu.Unlock()

	recipe.removed = true
	for _, id := range recipe.servers {
		if conn := T.raw.RemoveServer(id); conn != nil {
			_ = conn.Close()
		}
	}

	recipe.servers = recipe.servers[:0]
}

func (T *Pool) ScaleUp(amount int) (remaining int) {
	remaining = amount
	T.recipes.Range(func(_ string, r *PoolRecipe) bool {
		remaining = T.tryAddServers(r, remaining)
		return remaining != 0
	})
	return remaining
}

func (T *Pool) ScaleDown(amount int) (remaining int) {
	return T.raw.ScaleDown(amount)
}

func (T *Pool) IdleSince() time.Time {
	return T.raw.IdleSince()
}

func (T *Pool) AddRecipe(name string, recipe Recipe) {
	r := &PoolRecipe{
		r: recipe,
	}
	T.addRecipe(r)
	if old, ok := T.recipes.Swap(name, r); ok {
		T.removeRecipe(old)
	}
}

func (T *Pool) RemoveRecipe(name string) {
	if r, ok := T.recipes.LoadAndDelete(name); ok {
		T.removeRecipe(r)
	}
}

func (T *Pool) Serve(client zap.ReadWriter, startupParameters map[strutil.CIString]string) {
	T.raw.Serve(&T.ctx, client, startupParameters)
}
