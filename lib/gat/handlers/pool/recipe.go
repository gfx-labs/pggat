package pool

import (
	"sync"
)

type Recipe struct {
	Dialer

	Priority int `json:"priority,omitempty"`

	MinConnections int `json:"min_connections,omitempty"`

	// MaxConnections is the max number of active server connections for this recipe.
	// 0 = unlimited
	MaxConnections int `json:"max_connections,omitempty"`

	count int
	mu    sync.Mutex
}

func (T *Recipe) AllocateInitial() int {
	T.mu.Lock()
	defer T.mu.Unlock()

	if T.count >= T.MinConnections {
		return 0
	}

	amount := T.MinConnections - T.count
	T.count = T.MinConnections

	return amount
}

func (T *Recipe) Allocate() bool {
	T.mu.Lock()
	defer T.mu.Unlock()

	if T.MaxConnections != 0 {
		if T.count >= T.MaxConnections {
			return false
		}
	}

	T.count++
	return true
}

func (T *Recipe) TryFree() bool {
	T.mu.Lock()
	defer T.mu.Unlock()

	if T.count <= T.MinConnections {
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
