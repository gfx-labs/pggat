package kitchen

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"go.uber.org/zap"

	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool"
	"gfx.cafe/gfx/pggat/lib/util/maps"
	"gfx.cafe/gfx/pggat/lib/util/slices"
)

type Chef struct {
	config Config

	byName map[string]*Recipe
	byConn map[fed.Conn]*Recipe
	order  []*Recipe
	mu     sync.Mutex
}

func MakeChef(config Config) Chef {
	return Chef{
		config: config,
	}
}

func NewChef(config Config) *Chef {
	chef := MakeChef(config)
	return &chef
}

// Learn will add a recipe to the kitchen. Returns initial removed and added conns
func (T *Chef) Learn(name string, recipe *pool.Recipe) (removed []fed.Conn, added []fed.Conn) {
	n := recipe.AllocateInitial()
	added = make([]fed.Conn, 0, n)
	for i := 0; i < n; i++ {
		conn, err := recipe.Dial()
		if err != nil {
			// free remaining, failed to dial initial :(
			T.config.Logger.Error("failed to dial server", zap.Error(err))
			for j := i; j < n; j++ {
				recipe.Free()
			}
			break
		}

		added = append(added, conn)
	}

	T.mu.Lock()
	defer T.mu.Unlock()

	removed = T.forget(name)

	r := NewRecipe(recipe, added)

	if T.byName == nil {
		T.byName = make(map[string]*Recipe)
	}
	T.byName[name] = r

	if T.byConn == nil {
		T.byConn = make(map[fed.Conn]*Recipe)
	}
	for _, conn := range added {
		T.byConn[conn] = r
	}

	T.order = append(T.order, r)

	return
}

func (T *Chef) forget(name string) []fed.Conn {
	r, ok := T.byName[name]
	if !ok {
		return nil
	}
	delete(T.byName, name)

	conns := make([]fed.Conn, 0, len(r.conns))

	for conn := range r.conns {
		conns = append(conns, conn)
		_ = conn.Close()
		r.recipe.Free()

		delete(T.byConn, conn)
		delete(r.conns, conn)
	}

	T.order = slices.Remove(T.order, r)

	return conns
}

// Forget will remove a recipe from the kitchen. All conn made with the recipe will be closed. Returns conns made with
// recipe.
func (T *Chef) Forget(name string) []fed.Conn {
	T.mu.Lock()
	defer T.mu.Unlock()

	return T.forget(name)
}

func (T *Chef) Empty() bool {
	T.mu.Lock()
	defer T.mu.Unlock()

	return len(T.byName) == 0
}

func (T *Chef) cook(r *Recipe) (fed.Conn, error) {
	T.mu.Unlock()
	defer T.mu.Lock()

	return r.recipe.Dial()
}

func (T *Chef) score(r *Recipe) error {
	now := time.Now()

	r.ratings = slices.Resize(r.ratings, len(T.config.Critics))

	fast := true
	for i := range T.config.Critics {
		if now.After(r.ratings[i].Expiration) {
			fast = false
			break
		}
	}

	if fast {
		// score is just old, nothing changes
		return nil
	}

	critics := make([]pool.Critic, len(T.config.Critics))
	ratings := make([]Rating, len(T.config.Critics))

	total := 0

	for i, critic := range T.config.Critics {
		if now.Before(r.ratings[i].Expiration) {
			total += r.ratings[i].Score
			continue
		}

		critics[i] = critic
	}

	err := func() error {
		T.mu.Unlock()
		defer T.mu.Lock()

		conn, err := r.recipe.Dial()
		if err != nil {
			return err
		}
		defer func() {
			_ = conn.Close()
		}()

		for i, critic := range critics {
			if critic == nil {
				continue
			}

			var score int
			var validity time.Duration
			score, validity, err = critic.Taste(conn)
			if err != nil {
				return err
			}

			ratings[i] = Rating{
				Expiration: time.Now().Add(validity),
				Score:      score,
			}

			total += score
		}

		return nil
	}()
	if err != nil {
		return err
	}

	// update total score
	r.score = total

	// update ratings
	r.ratings = slices.Resize(r.ratings, len(T.config.Critics))

	for i, critic := range critics {
		if critic == nil {
			continue
		}

		r.ratings[i] = ratings[i]
	}

	return nil
}

// Cook will cook the best recipe
func (T *Chef) Cook() (fed.Conn, error) {
	T.mu.Lock()
	defer T.mu.Unlock()

	for _, r := range T.byName {
		if err := T.score(r); err != nil {
			r.score = math.MaxInt
			T.config.Logger.Error("failed to score recipe", zap.Error(err))
			continue
		}
	}

	sort.Slice(T.order, func(i, j int) bool {
		a := T.order[i]
		b := T.order[j]
		// sort by priority first
		if a.Rating() < b.Rating() {
			return true
		}
		if a.Rating() > b.Rating() {
			return false
		}
		// then sort by number of conns
		return len(a.conns) < len(b.conns)
	})

	for i, r := range T.order {
		if !r.recipe.Allocate() {
			continue
		}

		conn, err := T.cook(r)
		if err == nil {
			if r.conns == nil {
				r.conns = make(map[fed.Conn]struct{})
			}
			r.conns[conn] = struct{}{}

			if T.byConn == nil {
				T.byConn = make(map[fed.Conn]*Recipe)
			}
			T.byConn[conn] = r

			return conn, nil
		}

		T.config.Logger.Error("failed to dial server", zap.Error(err))

		r.recipe.Free()

		if i == len(T.order)-1 {
			// return last error
			return nil, err
		}
	}

	var fields = make([]zap.Field, 0, len(T.byName))
	for name, recipe := range T.byName {
		fields = append(fields, zap.String(name, fmt.Sprintf("%d / %d", len(recipe.conns), recipe.recipe.MaxConnections)))
	}

	T.config.Logger.Warn("no available recipes to scale up pool", fields...)
	return nil, ErrNoRecipes
}

// Burn forcefully closes conn and escorts it out of the kitchen.
func (T *Chef) Burn(conn fed.Conn) {
	T.mu.Lock()
	defer T.mu.Unlock()

	r, ok := T.byConn[conn]
	if !ok {
		return
	}
	r.recipe.Free()
	_ = conn.Close()

	delete(T.byConn, conn)
	delete(r.conns, conn)
}

// Ignite tries to Burn conn. If successful, conn is closed and returns true
func (T *Chef) Ignite(conn fed.Conn) bool {
	T.mu.Lock()
	defer T.mu.Unlock()

	r, ok := T.byConn[conn]
	if !ok {
		return false
	}
	if !r.recipe.TryFree() {
		return false
	}
	_ = conn.Close()

	delete(T.byConn, conn)
	delete(r.conns, conn)
	return true
}

func (T *Chef) Cancel(conn fed.Conn) {
	T.mu.Lock()
	defer T.mu.Unlock()

	r, ok := T.byConn[conn]
	if !ok {
		return
	}

	r.recipe.Cancel(conn.BackendKey())
}

func (T *Chef) Close() {
	T.mu.Lock()
	defer T.mu.Unlock()

	maps.Clear(T.byName)
	T.order = T.order[:0]
	for conn, r := range T.byConn {
		r.recipe.Free()
		_ = conn.Close()

		delete(T.byConn, conn)
		delete(r.conns, conn)
	}
}
