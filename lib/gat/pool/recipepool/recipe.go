package recipepool

import (
	"log"

	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/fed/middlewares/eqp"
	"gfx.cafe/gfx/pggat/lib/fed/middlewares/ps"
	"gfx.cafe/gfx/pggat/lib/gat/pool"
	"gfx.cafe/gfx/pggat/lib/util/slices"
)

type Recipe struct {
	parameterStatusSync pool.ParameterStatusSync
	extendedQuerySync   bool

	r *pool.Recipe

	servers []*pool.Conn
}

func NewRecipe(parameterStatusSync pool.ParameterStatusSync, extendedQuerySync bool, r *pool.Recipe) *Recipe {
	s := &Recipe{
		parameterStatusSync: parameterStatusSync,
		extendedQuerySync:   extendedQuerySync,

		r: r,
	}
	s.init()
	return s
}

func (T *Recipe) init() {
	count := T.r.AllocateInitial()
	T.servers = make([]*pool.Conn, 0, count)

	for i := 0; i < count; i++ {
		conn := T.dial()
		if conn == nil {
			T.r.Free()
		}
		T.servers = append(T.servers, conn)
	}
}

func (T *Recipe) dial() *pool.Conn {
	conn, err := T.r.Dial()
	if err != nil {
		// TODO(garet) use proper logger
		log.Printf("failed to dial server: %v", err)
		return nil
	}

	if T.extendedQuerySync {
		conn.Middleware = append(
			conn.Middleware,
			eqp.NewServer(),
		)
	}

	if T.parameterStatusSync == pool.ParameterStatusSyncDynamic {
		conn.Middleware = append(
			conn.Middleware,
			ps.NewServer(conn.InitialParameters),
		)
	}

	return pool.NewConn(conn)
}

func (T *Recipe) Dial() *pool.Conn {
	if !T.r.Allocate() {
		return nil
	}

	c := T.dial()
	if c == nil {
		T.r.Free()
	}
	return c
}

func (T *Recipe) Cancel(key fed.BackendKey) {
	T.r.Cancel(key)
}

func (T *Recipe) TryRemoveServer(server *pool.Conn) bool {
	idx := slices.Index(T.servers, server)
	if idx == -1 {
		return false
	}
	if !T.r.TryFree() {
		return false
	}
	T.servers = slices.DeleteIndex(T.servers, idx)
	return true
}

func (T *Recipe) RemoveServer(server *pool.Conn) {
	idx := slices.Index(T.servers, server)
	if idx == -1 {
		return
	}
	T.servers = slices.DeleteIndex(T.servers, idx)
	T.r.Free()
}
