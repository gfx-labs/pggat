package gat

import (
	"log"
	"sync"

	"github.com/google/uuid"

	"pggat2/lib/bouncer/backends/v0"
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
}

type recipeWithConns struct {
	recipe Recipe

	conns []uuid.UUID
	mu    sync.Mutex
}

func (T *recipeWithConns) scaleUp(pool *Pool, currentScale int) bool {
	if currentScale >= T.recipe.GetMaxConnections() {
		return false
	}

	T.mu.Unlock()
	conn, err := T.recipe.Connect()
	if err != nil {
		log.Printf("Failed to connect: %v", err)
		T.mu.Lock()
		return false
	}
	err2 := backends.Accept(conn, T.recipe.GetUser(), T.recipe.GetPassword(), T.recipe.GetDatabase())
	if err2 != nil {
		_ = conn.Close()
		log.Printf("Failed to connect: %v", err2)
		T.mu.Lock()
		return false
	}

	id := pool.raw.AddServer(conn)
	T.mu.Lock()
	T.conns = append(T.conns, id)
	return true
}

func (T *recipeWithConns) scaleDown(pool *Pool, currentScale int) bool {
	if currentScale <= T.recipe.GetMinConnections() {
		return false
	}

	if len(T.conns) == 0 {
		// none to close
		return false
	}

	id := T.conns[len(T.conns)-1]
	conn := pool.raw.RemoveServer(id)
	if conn != nil {
		_ = conn.Close()
	}
	T.conns = T.conns[:len(T.conns)-1]
	return true
}

func (T *recipeWithConns) scale(pool *Pool, currentScale int, amount int) int {
	if amount > 0 {
		for amount > 0 {
			if T.scaleUp(pool, currentScale) {
				amount--
				currentScale++
			} else {
				break
			}
		}
	} else {
		for amount < 0 {
			if T.scaleDown(pool, currentScale) {
				amount++
				currentScale--
			} else {
				break
			}
		}
	}
	return amount
}

func (T *recipeWithConns) currentScale(pool *Pool) int {
	i := 0
	for j := 0; j < len(T.conns); j++ {
		if pool.raw.GetServer(T.conns[j]) != nil {
			T.conns[i] = T.conns[j]
			i++
		}
	}

	T.conns = T.conns[:i]
	return i
}

func (T *recipeWithConns) CurrentScale(pool *Pool) int {
	T.mu.Lock()
	defer T.mu.Unlock()

	return T.currentScale(pool)
}

func (T *recipeWithConns) Scale(pool *Pool, amount int) int {
	T.mu.Lock()
	defer T.mu.Unlock()

	currentScale := T.currentScale(pool)
	return T.scale(pool, currentScale, amount)
}

func (T *recipeWithConns) SetScale(pool *Pool, scale int) {
	T.mu.Lock()
	defer T.mu.Unlock()

	target := maths.Clamp(scale, T.recipe.GetMinConnections(), T.recipe.GetMaxConnections())
	currentScale := T.currentScale(pool)
	target -= currentScale

	T.scale(pool, currentScale, target)
}

func (T *recipeWithConns) Added(pool *Pool) {
	T.SetScale(pool, 0)
}

func (T *recipeWithConns) Removed(pool *Pool) {
	T.mu.Lock()
	defer T.mu.Unlock()

	for _, conn := range T.conns {
		pool.raw.RemoveServer(conn)
	}

	T.conns = T.conns[:0]
}

type Pool struct {
	recipes map[string]*recipeWithConns
	mu      sync.Mutex

	ctx Context

	raw RawPool
}

func NewPool(rawPool RawPool) *Pool {
	onWait := make(chan struct{})

	p := &Pool{
		ctx: Context{
			OnWait: onWait,
		},
		raw: rawPool,
	}

	go func() {
		for {
			_, ok := <-onWait
			if !ok {
				break
			}

			p.Scale(1)
		}
	}()

	return p
}

func (T *Pool) Serve(client zap.ReadWriter) {
	T.raw.Serve(&T.ctx, client)
}

func (T *Pool) CurrentScale() int {
	T.mu.Lock()
	recipes := make([]string, 0, len(T.recipes))
	for recipe := range T.recipes {
		recipes = append(recipes, recipe)
	}
	T.mu.Unlock()

	scale := 0
	for _, recipe := range recipes {
		scale += T.recipes[recipe].CurrentScale(T)
	}
	return scale
}

func (T *Pool) Scale(amount int) {
	T.mu.Lock()
	recipes := make([]string, 0, len(T.recipes))
	for recipe := range T.recipes {
		recipes = append(recipes, recipe)
	}
	T.mu.Unlock()

outer:
	for len(recipes) > 0 {
		j := 0
		for i := 0; i < len(recipes); i++ {
			recipe := recipes[i]
			if amount > 0 {
				if T.recipes[recipe].Scale(T, 1) == 0 {
					amount--
					recipes[j] = recipes[i]
					j++
				}
			} else if amount < 0 {
				if T.recipes[recipe].Scale(T, -1) == 0 {
					amount++
					recipes[j] = recipes[i]
					j++
				}
			} else {
				break outer
			}
		}
		recipes = recipes[:j]
	}
}

func (T *Pool) AddRecipe(name string, recipe Recipe) {
	r := &recipeWithConns{
		recipe: recipe,
	}
	r.Added(T)

	T.mu.Lock()
	old, ok := T.recipes[name]
	if T.recipes == nil {
		T.recipes = make(map[string]*recipeWithConns)
	}
	T.recipes[name] = r
	T.mu.Unlock()

	if ok {
		old.Removed(T)
	}
}

func (T *Pool) RemoveRecipe(name string) {
	T.mu.Lock()
	r, ok := T.recipes[name]
	delete(T.recipes, name)
	T.mu.Unlock()

	if ok {
		r.Removed(T)
	}
}
