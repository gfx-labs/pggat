package session

import (
	"pggat/lib/gat/pool"
)

func Apply(options pool.Options) pool.Options {
	options.Pooler = new(Pooler)
	options.ParameterStatusSync = pool.ParameterStatusSyncInitial
	options.ExtendedQuerySync = false
	return options
}
