package session

import (
	"pggat2/lib/gat/pool"
)

func NewPool(options pool.Options) *pool.Pool {
	options.Pooler = new(Pooler)
	options.ParameterStatusSync = pool.ParameterStatusSyncInitial
	options.ExtendedQuerySync = false
	return pool.NewPool(options)
}
