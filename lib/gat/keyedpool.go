package gat

import (
	"pggat/lib/gat/pool"
	"pggat/lib/util/maps"
)

type KeyedPools struct {
	Pools

	keys maps.RWLocked[[8]byte, *pool.Pool]
}

func NewKeyedPools(pools Pools) *KeyedPools {
	return &KeyedPools{
		Pools: pools,
	}
}

func (T *KeyedPools) RegisterKey(key [8]byte, user, database string) {
	p := T.Lookup(user, database)
	if p == nil {
		return
	}
	T.keys.Store(key, p)
}

func (T *KeyedPools) UnregisterKey(key [8]byte) {
	T.keys.Delete(key)
}

func (T *KeyedPools) LookupKey(key [8]byte) *pool.Pool {
	p, _ := T.keys.Load(key)
	return p
}
