package basic

import (
	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool"
	"gfx.cafe/gfx/pggat/lib/gat/metrics"
)

type Pool struct {
	config Config
}

func NewPool(config Config) *Pool {
	return &Pool{
		config: config,
	}
}

func (T *Pool) AddRecipe(name string, recipe *pool.Recipe) {
	// TODO implement me
	panic("implement me")
}

func (T *Pool) RemoveRecipe(name string) {
	// TODO implement me
	panic("implement me")
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
	// TODO implement me
	panic("implement me")
}

func (T *Pool) Close() {
	// TODO implement me
	panic("implement me")
}

var _ pool.Pool = (*Pool)(nil)
