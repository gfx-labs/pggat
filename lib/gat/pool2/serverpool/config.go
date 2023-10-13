package serverpool

import "gfx.cafe/gfx/pggat/lib/gat/pool"

type Config struct {
	// NewPooler allocates a new pooler
	NewPooler func() pool.Pooler
	// ServerResetQuery is the query to be run before the server is released
	ServerResetQuery string
}
