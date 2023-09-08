package transaction

import "pggat/lib/gat/pool"

func NewPool(options pool.Options) *pool.Pool {
	options.Pooler = new(Pooler)
	options.ParameterStatusSync = pool.ParameterStatusSyncDynamic
	options.ExtendedQuerySync = true
	return pool.NewPool(options)
}
