package pool

import (
	"context"
	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/gat/metrics"
)

type Pool interface {
	// AddRecipe will add the recipe to the pool for use. The pool should delete any existing recipes with the same name
	// and scale the recipe to min.
	AddRecipe(name string, recipe *Recipe)
	// RemoveRecipe will remove a recipe and disconnect all servers created by that recipe.
	RemoveRecipe(name string)

	Serve(ctx context.Context, conn *fed.Conn) error

	Cancel(ctx context.Context, key fed.BackendKey)
	ReadMetrics(m *metrics.Pool)
	Close(ctx context.Context)
}

type ReplicaPool interface {
	Pool

	AddReplicaRecipe(name string, recipe *Recipe)
	RemoveReplicaRecipe(name string)
}

type PoolFactory interface {
	NewPool() Pool
}
