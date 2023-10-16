package lifo

import (
	"github.com/caddyserver/caddy/v2"

	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool"
)

type Factory struct {
}

func (T *Factory) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "pggat.handlers.pool.poolers.lifo",
		New: func() caddy.Module {
			return new(Factory)
		},
	}
}

func (T *Factory) NewPooler() pool.Pooler {
	return new(Pooler)
}

var _ pool.PoolerFactory = (*Factory)(nil)
var _ caddy.Module = (*Factory)(nil)
