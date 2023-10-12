package pool

import (
	"log"
	"sync"
	"sync/atomic"
	"time"

	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/pool/recipe"
)

type backendRecipe struct {
	weight atomic.Int64
	recipe *recipe.Recipe

	servers []*fed.Conn // TODO(garet)
	killed  bool
}

type Backend struct {
	config BackendConfig
	pooler Pooler

	closed chan struct{}

	scaleUp chan struct{}
	waiters atomic.Int64

	recipes map[string]*backendRecipe
	mu      sync.RWMutex
}

func NewBackend(config BackendConfig) *Backend {
	b := &Backend{
		config: config,
		pooler: config.NewPooler(),

		closed: make(chan struct{}),

		scaleUp: make(chan struct{}, 1),
	}

	go b.scaleLoop()

	return b
}

func (T *Backend) addRecipe(name string, r *recipe.Recipe) {
	if T.recipes == nil {
		T.recipes = make(map[string]*backendRecipe)
	}
	T.recipes[name] = &backendRecipe{
		recipe: r,
	}
}

func (T *Backend) AddRecipe(name string, r *recipe.Recipe) {
	T.mu.Lock()
	defer T.mu.Unlock()

	T.removeRecipe(name)
	T.addRecipe(name, r)
}

func (T *Backend) removeRecipe(name string) {
	r, ok := T.recipes[name]
	if !ok {
		return
	}
	delete(T.recipes, name)

	r.killed = true
	for _, server := range r.servers {
		_ = server.Close()
		r.recipe.Free()
	}
	r.servers = r.servers[:0]
}

func (T *Backend) RemoveRecipe(name string) {
	T.mu.Lock()
	defer T.mu.Unlock()

	T.removeRecipe(name)
}

func (T *Backend) ScaleUp() error {
	var target *backendRecipe
	func() {
		T.mu.RLock()
		defer T.mu.RUnlock()

		var targetWeight int64 = -1
		for _, r := range T.recipes {
			if target == nil {
				target = r
				continue
			}
			weight := r.weight.Load()
			if weight > targetWeight {
				target = r
				targetWeight = weight
				continue
			}
		}

		if !target.recipe.Allocate() {
			return
		}
		target.weight.Add(1)
	}()

	if target == nil {
		return ErrNoScalableRecipe
	}

	server, err := target.recipe.Dial()
	if err != nil {
		target.recipe.Free()
		target.weight.Add(-1)
		return err
	}

	T.mu.Lock()
	defer T.mu.Unlock()

	if target.killed {
		target.recipe.Free()
		target.weight.Add(-1)
		return nil
	}

	target.servers = append(target.servers, server)
	return nil
}

// ScaleDown attempts to scale down the pool. Returns when the next scale down should happen
func (T *Backend) ScaleDown() time.Duration {
	// TODO(garet)
	return T.config.IdleTimeout
}

func (T *Backend) scaleLoop() {
	var scaleUpBackoffDuration time.Duration
	var scaleUpBackoff *time.Timer

	var idleTimeout *time.Timer
	if T.config.IdleTimeout != 0 {
		idleTimeout = time.NewTimer(T.config.IdleTimeout)
	}

	defer func() {
		if scaleUpBackoff != nil {
			scaleUpBackoff.Stop()
		}
		if idleTimeout != nil {
			idleTimeout.Stop()
		}
	}()

	for {
		var scaleUpBackoffC <-chan time.Time
		var scaleUpC <-chan struct{}
		if scaleUpBackoffDuration != 0 {
			scaleUpBackoffC = scaleUpBackoff.C
		} else {
			scaleUpC = T.scaleUp
		}

		var idleTimeoutC <-chan time.Time
		if idleTimeout != nil {
			idleTimeoutC = idleTimeout.C
		}

		select {
		case <-scaleUpC:
			if err := T.ScaleUp(); err != nil {
				log.Printf("error scaling up: %v", err)

				scaleUpBackoffDuration = T.config.ReconnectInitialTime
				if scaleUpBackoffDuration != 0 {
					if scaleUpBackoff == nil {
						scaleUpBackoff = time.NewTimer(scaleUpBackoffDuration)
					} else {
						scaleUpBackoff.Reset(scaleUpBackoffDuration)
					}
				}
			}
		case <-scaleUpBackoffC:
			if err := T.ScaleUp(); err != nil {
				log.Printf("error scaling up: %v", err)

				scaleUpBackoffDuration *= 2
				scaleUpBackoff.Reset(scaleUpBackoffDuration)
			} else {
				scaleUpBackoffDuration = 0
			}
		case <-idleTimeoutC:
			idleTimeout.Reset(T.ScaleDown())
		case <-T.closed:
			return
		}
	}
}

func (T *Backend) Close() {
	close(T.closed)
}
