package gat

import (
	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/gat/metrics"
)

type Pool interface {
	Serve(conn *fed.Conn) error

	Cancel(key fed.BackendKey)
	ReadMetrics(m *metrics.Pool)
	Close()
}
