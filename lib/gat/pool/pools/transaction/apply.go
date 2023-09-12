package transaction

import "pggat/lib/gat/pool"

func Apply(options pool.Options) pool.Options {
	options.Pooler = new(Pooler)
	options.ParameterStatusSync = pool.ParameterStatusSyncDynamic
	options.ExtendedQuerySync = true
	return options
}
