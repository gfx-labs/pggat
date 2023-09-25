package session

import (
	"gfx.cafe/gfx/pggat/lib/gat/pool"
)

func Apply(options pool.Options) pool.Options {
	options.NewPooler = NewPooler
	options.ParameterStatusSync = pool.ParameterStatusSyncInitial
	options.ExtendedQuerySync = false
	return options
}
