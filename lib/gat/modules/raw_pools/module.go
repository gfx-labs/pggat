package raw_pools

import (
	"sync"

	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/gat/metrics"
	"gfx.cafe/gfx/pggat/lib/gat/pool"
	"gfx.cafe/gfx/pggat/lib/util/maps"
)

type Module struct {
	pools maps.TwoKey[string, string, *pool.Pool]
	mu    sync.RWMutex
}

func (T *Module) GatModule() {}

func (T *Module) Add(user, database string, p *pool.Pool) {
	T.mu.Lock()
	defer T.mu.Unlock()

	T.pools.Store(user, database, p)
}

func (T *Module) Remove(user, database string) {
	T.mu.Lock()
	defer T.mu.Unlock()

	T.pools.Delete(user, database)
}

func (T *Module) Lookup(user, database string) *gat.Pool {
	T.mu.RLock()
	defer T.mu.RUnlock()

	p, _ := T.pools.Load(user, database)
	return p
}

func (T *Module) ReadMetrics(metrics *metrics.Pools) {
	T.mu.RLock()
	defer T.mu.RUnlock()

	T.pools.Range(func(_ string, _ string, p *pool.Pool) bool {
		p.ReadMetrics(&metrics.Pool)
		return true
	})
}

var _ gat.Module = (*Module)(nil)
var _ gat.Provider = (*Module)(nil)
