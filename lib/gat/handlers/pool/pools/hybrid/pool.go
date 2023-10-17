package hybrid

import (
	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool/spool"
	"gfx.cafe/gfx/pggat/lib/gat/metrics"
)

type Pool struct {
	primary spool.Pool
	replica spool.Pool
}

func (T *Pool) AddReplicaRecipe(name string, recipe *pool.Recipe) {
	T.replica.AddRecipe(name, recipe)
}

func (T *Pool) RemoveReplicaRecipe(name string) {
	T.replica.RemoveRecipe(name)
}

func (T *Pool) AddRecipe(name string, recipe *pool.Recipe) {
	T.primary.AddRecipe(name, recipe)
}

func (T *Pool) RemoveRecipe(name string) {
	T.primary.RemoveRecipe(name)
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
	T.primary.ReadMetrics(m)
	T.replica.ReadMetrics(m)
}

func (T *Pool) Close() {
	T.primary.Close()
	T.replica.Close()
}

var _ pool.Pool = (*Pool)(nil)
var _ pool.ReplicaPool = (*Pool)(nil)
