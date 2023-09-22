package gat

import (
	"pggat/lib/gat/metrics"
	"pggat/lib/gat/pool"
)

type Pools interface {
	Lookup(user, database string) *pool.Pool

	ReadMetrics(metrics *metrics.Pools)
}
