package gat

import (
	"errors"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"

	"pggat2/lib/bouncer/backends/v0"
	"pggat2/lib/util/maps"
	"pggat2/lib/util/maths"
	"pggat2/lib/zap"
)

type Context struct {
	OnWait chan<- struct{}
}

type RawPool interface {
	Serve(ctx *Context, client zap.ReadWriter)

	AddServer(server zap.ReadWriter) uuid.UUID
	GetServer(id uuid.UUID) zap.ReadWriter
	RemoveServer(id uuid.UUID) zap.ReadWriter

	ScaleDown(amount int) (remaining int)
	IdleSince() time.Time
}

type PoolRecipe struct {
	removed bool
	servers []uuid.UUID
	mu      sync.Mutex

	r Recipe
}

func (T *PoolRecipe) connect() (zap.ReadWriter, error) {
	rw, err := T.r.Connect()
	if err != nil {
		return nil, err
	}

	err2 := backends.Accept(rw, T.r.GetUser(), T.r.GetPassword(), T.r.GetDatabase())
	if err2 != nil {
		return nil, errors.New(err2.Message())
	}

	return rw, nil
}

type Pool struct {
	recipes maps.RWLocked[string, *PoolRecipe]

	ctx Context
	raw RawPool
}

func NewPool(raw RawPool, idleTimeout time.Duration) *Pool {
	onWait := make(chan struct{})
	pool := &Pool{
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

	go func() {
		for {
			var wait time.Duration

			now := time.Now()
			idle := pool.IdleSince()
			for now.Sub(idle) > idleTimeout {
				if idle == (time.Time{}) {
					break
				}
				pool.ScaleDown(1)
				idle = pool.IdleSince()
			}

			if idle == (time.Time{}) {
				wait = idleTimeout
			} else {
				wait = now.Sub(idle.Add(idleTimeout))
			}

			time.Sleep(wait)
		}
	}()

	return pool
}

func (T *Pool) tryAddServers(recipe *PoolRecipe, amount int) (remaining int) {
	recipe.mu.Lock()
	defer recipe.mu.Unlock()

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

	max := maths.Min(recipe.r.GetMaxConnections()-j, amount)
	for i := 0; i < max; i++ {
		conn, err := recipe.connect()
		if err != nil {
			log.Printf("error connecting to server: %v", err)
			continue
		}

		id := T.raw.AddServer(conn)
		recipe.servers = append(recipe.servers, id)
		remaining--
	}

	return
}

func (T *Pool) addRecipe(recipe *PoolRecipe) {
	recipe.mu.Lock()
	defer recipe.mu.Unlock()

	recipe.removed = false
	min := recipe.r.GetMinConnections() - len(recipe.servers)
	for i := 0; i < min; i++ {
		conn, err := recipe.connect()
		if err != nil {
			log.Printf("error connecting to server: %v", err)
			continue
		}

		id := T.raw.AddServer(conn)
		recipe.servers = append(recipe.servers, id)
	}
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

func (T *Pool) Serve(client zap.ReadWriter) {
	T.raw.Serve(&T.ctx, client)
}
