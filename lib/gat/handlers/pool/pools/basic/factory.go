package basic

import (
	"fmt"

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

	if T.RawCritics != nil {
		raw, err := ctx.LoadModule(T, "RawCritics")
		if err != nil {
			return fmt.Errorf("loading critic module: %v", err)
		}

		val := raw.([]any)
		T.Critics = make([]pool.Critic, 0, len(val))
		for _, vv := range val {
			T.Critics = append(T.Critics, vv.(pool.Critic))
		}
	}

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
