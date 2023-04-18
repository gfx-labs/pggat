package session

import "gfx.cafe/gfx/pggat/lib/gat2/pool"

type Pool struct{}

var _ pool.Pool = (*Pool)(nil)
