package transaction

import "pggat/lib/gat/pool"

func Apply(options pool.Options) pool.Options {
	options.Pooler = NewPooler()
	options.ParameterStatusSync = pool.ParameterStatusSyncDynamic
	options.ExtendedQuerySync = true
	options.ReleaseAfterTransaction = true
	return options
}
