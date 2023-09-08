package gat

import (
	"pggat/lib/gat/metrics"
	"pggat/lib/gat/pool"
	"pggat/lib/util/maps"
)

type Pools interface {
	Lookup(user, database string) *pool.Pool

	ReadMetrics(metrics *metrics.Pools)

	// Key based lookup functions (for cancellation)

	RegisterKey(key [8]byte, user, database string)
	UnregisterKey(key [8]byte)

	LookupKey(key [8]byte) *pool.Pool
}

type mapKey struct {
	User     string
	Database string
}

type PoolsMap struct {
	pools maps.RWLocked[mapKey, *pool.Pool]
	keys  maps.RWLocked[[8]byte, mapKey]
}

func (T *PoolsMap) Add(user, database string, pool *pool.Pool) {
	T.pools.Store(mapKey{
		User:     user,
		Database: database,
	}, pool)
}

func (T *PoolsMap) Remove(user, database string) {
	T.pools.Delete(mapKey{
		User:     user,
		Database: database,
	})
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

// key based lookup funcs

func (T *PoolsMap) RegisterKey(key [8]byte, user, database string) {
	T.keys.Store(key, mapKey{
		User:     user,
		Database: database,
	})
}

func (T *PoolsMap) UnregisterKey(key [8]byte) {
	T.keys.Delete(key)
}

func (T *PoolsMap) LookupKey(key [8]byte) *pool.Pool {
	m, ok := T.keys.Load(key)
	if !ok {
		return nil
	}
	p, ok := T.pools.Load(m)
	if !ok {
		T.keys.Delete(key)
		return nil
	}
	return p
}
