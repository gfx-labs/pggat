package pool

import (
	"sync"

	"gfx.cafe/gfx/pggat/lib/fed"
)

type Recipe struct {
	config RecipeConfig

	count int
	mu    sync.Mutex
}

type RecipeConfig struct {
	Dialer Dialer

	MinConnections int
	// MaxConnections is the max number of active server connections for this recipe.
	// 0 = unlimited
	MaxConnections int
}

func NewRecipe(config RecipeConfig) *Recipe {
	return &Recipe{
		config: config,
	}
}

func (T *Recipe) AllocateInitial() int {
	T.mu.Lock()
	defer T.mu.Unlock()

	if T.count >= T.config.MinConnections {
		return 0
	}

	amount := T.config.MinConnections - T.count
	T.count = T.config.MinConnections

	return amount
}

func (T *Recipe) Allocate() bool {
	T.mu.Lock()
	defer T.mu.Unlock()

	if T.config.MaxConnections != 0 {
		if T.count >= T.config.MaxConnections {
			return false
		}
	}

	T.count++
	return true
}

func (T *Recipe) TryFree() bool {
	T.mu.Lock()
	defer T.mu.Unlock()

	if T.count <= T.config.MinConnections {
		return false
	}

	T.count--
	return true
}

func (T *Recipe) Free() {
	T.mu.Lock()
	defer T.mu.Unlock()

	T.count--
}

func (T *Recipe) Dial() (*fed.Conn, error) {
	return T.config.Dialer.Dial()
}

func (T *Recipe) Cancel(key fed.BackendKey) {
	T.config.Dialer.Cancel(key)
}
