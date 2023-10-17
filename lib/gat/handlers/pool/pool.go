package pool

import (
	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/gat/metrics"
)

type Pool interface {
	AddRecipe(name string, recipe *Recipe)
	RemoveRecipe(name string)

	Serve(conn *fed.Conn) error

	Cancel(key fed.BackendKey)
	ReadMetrics(m *metrics.Pool)
	Close()
}

type ReplicaPool interface {
	Pool

	AddReplicaRecipe(name string, recipe *Recipe)
	RemoveReplicaRecipe(name string)
}

type PoolFactory interface {
	NewPool() Pool
}
