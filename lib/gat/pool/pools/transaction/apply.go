package transaction

import "gfx.cafe/gfx/pggat/lib/gat/pool"

func Apply(options pool.Options) pool.Options {
	options.NewPooler = NewPooler
	options.ParameterStatusSync = pool.ParameterStatusSyncDynamic
	options.ExtendedQuerySync = true
	options.ReleaseAfterTransaction = true
	return options
}
