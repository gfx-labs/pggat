package gat

import (
	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/gat/metrics"
)

// Provider provides pool to the server
type Provider interface {
	Lookup(conn *fed.Conn) *Pool
	ReadMetrics(metrics *metrics.Pools)
}
