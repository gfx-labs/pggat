package hybrid

import (
	"github.com/caddyserver/caddy/v2"

	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool"
)

func init() {
	caddy.RegisterModule((*Factory)(nil))
}

type Factory struct {
}

func (T *Factory) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "pggat.handlers.pool.pools.hybrid",
		New: func() caddy.Module {
			return new(Factory)
		},
	}
}

func (T *Factory) NewPool() pool.Pool {
	return new(Pool)
}

var _ pool.PoolFactory = (*Factory)(nil)
var _ caddy.Module = (*Factory)(nil)
