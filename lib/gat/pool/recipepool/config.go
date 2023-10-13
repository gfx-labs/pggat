package recipepool

import (
	"gfx.cafe/gfx/pggat/lib/gat/pool/serverpool"

	"gfx.cafe/gfx/pggat/lib/gat/pool"
)

type Config struct {
	// ParameterStatusSync is the parameter syncing mode
	ParameterStatusSync pool.ParameterStatusSync

	// ExtendedQuerySync controls whether prepared statements and portals should be tracked and synced before use.
	// Use false for lower latency
	// Use true for transaction pooling
	ExtendedQuerySync bool

	serverpool.Config
}
