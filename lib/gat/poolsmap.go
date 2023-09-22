package gat

import (
	"pggat/lib/gat/metrics"
	"pggat/lib/gat/pool"
	"pggat/lib/util/maps"
)

type mapKey struct {
	User     string
	Database string
}

type PoolsMap struct {
	pools maps.RWLocked[mapKey, *pool.Pool]
}

func (T *PoolsMap) Add(user, database string, pool *pool.Pool) {
	T.pools.Store(mapKey{
		User:     user,
		Database: database,
	}, pool)
}

func (T *PoolsMap) Remove(user, database string) *pool.Pool {
	p, _ := T.pools.LoadAndDelete(mapKey{
		User:     user,
		Database: database,
	})
	return p
}

func (T *PoolsMap) Lookup(user, database string) *pool.Pool {
	p, _ := T.pools.Load(mapKey{
		User:     user,
		Database: database,
	})
	return p
}

func (T *PoolsMap) ReadMetrics(metrics *metrics.Pools) {
	T.pools.Range(func(_ mapKey, p *pool.Pool) bool {
		p.ReadMetrics(&metrics.Pool)
		return true
	})
}
