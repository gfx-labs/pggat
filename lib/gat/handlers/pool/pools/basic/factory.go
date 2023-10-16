package basic

import (
	"github.com/caddyserver/caddy/v2"

	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool"
)

func init() {
	caddy.RegisterModule((*Factory)(nil))
}

type Factory struct {
	Config
}

func (T *Factory) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "pggat.handlers.pool.pools.basic",
		New: func() caddy.Module {
			return new(Factory)
		},
	}
}

func (T *Factory) Provision(ctx caddy.Context) error {
	T.Logger = ctx.Logger()

	raw, err := ctx.LoadModule(T, "RawPoolerFactory")
	if err != nil {
		return err
	}

	T.PoolerFactory = raw.(pool.PoolerFactory)
	return nil
}

func (T *Factory) NewPool() pool.Pool {
	return NewPool(T.Config)
}

var _ pool.PoolFactory = (*Factory)(nil)
var _ caddy.Module = (*Factory)(nil)
var _ caddy.Provisioner = (*Factory)(nil)
