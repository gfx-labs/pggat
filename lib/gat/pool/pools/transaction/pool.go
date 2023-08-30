package transaction

import "pggat2/lib/gat/pool"

func NewPool(options pool.Options) *pool.Pool {
	options.Pooler = new(Pooler)
	options.ParameterStatusSync = pool.ParameterStatusSyncDynamic
	options.ExtendedQuerySync = true
	return pool.NewPool(options)
}
