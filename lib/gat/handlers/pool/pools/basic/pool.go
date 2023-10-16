package basic

import (
	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool/spool"
	"gfx.cafe/gfx/pggat/lib/gat/metrics"
)

type Pool struct {
	config Config

	servers spool.Pool
}

func NewPool(config Config) *Pool {
	p := &Pool{
		config:  config,
		servers: spool.MakePool(config.Spool()),
	}
	go p.servers.ScaleLoop()
	return p
}

func (T *Pool) AddRecipe(name string, recipe *pool.Recipe) {
	T.servers.AddRecipe(name, recipe)
}

func (T *Pool) RemoveRecipe(name string) {
	T.servers.RemoveRecipe(name)
}

func (T *Pool) Serve(conn *fed.Conn) error {
	// TODO implement me
	panic("implement me")
}

func (T *Pool) Cancel(key fed.BackendKey) {
	// TODO implement me
	panic("implement me")
}

func (T *Pool) ReadMetrics(m *metrics.Pool) {
	T.servers.ReadMetrics(m)

	// TODO(garet) read client metrics
}

func (T *Pool) Close() {
	T.servers.Close()
}

var _ pool.Pool = (*Pool)(nil)
