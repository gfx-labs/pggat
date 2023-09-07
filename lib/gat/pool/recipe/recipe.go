package recipe

import (
	"sync"

	"pggat2/lib/bouncer/backends/v0"
	"pggat2/lib/fed"
)

type Recipe struct {
	options Options

	count int
	mu    sync.Mutex
}

func NewRecipe(options Options) *Recipe {
	return &Recipe{
		options: options,
	}
}

func (T *Recipe) Allocate() bool {
	T.mu.Lock()
	defer T.mu.Unlock()

	if T.options.MaxConnections != 0 {
		if T.count >= T.options.MaxConnections {
			return false
		}
	}

	T.count++
	return true
}

func (T *Recipe) TryFree() bool {
	T.mu.Lock()
	defer T.mu.Unlock()

	if T.count <= T.options.MinConnections {
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

func (T *Recipe) Dial() (fed.Conn, backends.AcceptParams, error) {
	return T.options.Dialer.Dial()
}
