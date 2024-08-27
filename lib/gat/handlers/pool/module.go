package pool

import (
	"context"
	"encoding/json"

	"github.com/caddyserver/caddy/v2"

	"gfx.cafe/gfx/pggat/lib/bouncer/frontends/v0"
	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/gat/metrics"
	"gfx.cafe/gfx/pggat/lib/util/decorator"
)

func init() {
	caddy.RegisterModule((*Module)(nil))
}

type Module struct {
	noCopy decorator.NoCopy

	Pool   json.RawMessage `json:"pool" caddy:"namespace=pggat.handlers.pool.pools inline_key=pool"`
	Recipe Recipe          `json:"recipe"`

	pool Pool
}

func (T *Module) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "pggat.handlers.pool",
		New: func() caddy.Module {
			return new(Module)
		},
	}
}

func (T *Module) Provision(ctx caddy.Context) error {
	raw, err := ctx.LoadModule(T, "Pool")
	if err != nil {
		return err
	}
	T.pool = raw.(PoolFactory).NewPool(ctx)

	if err = T.Recipe.Provision(ctx); err != nil {
		return err
	}

	T.pool.AddRecipe(ctx, "recipe", &T.Recipe)
	return nil
}

func (T *Module) Handle(gat.Router) gat.Router {
	return gat.RouterFunc(func(ctx context.Context, c *fed.Conn) error {
		if err := frontends.Authenticate(ctx, c, nil); err != nil {
			return err
		}

		return T.pool.Serve(ctx, c)
	})
}

func (T *Module) ReadMetrics(ctx context.Context, metrics *metrics.Handler) {
	T.pool.ReadMetrics(ctx, &metrics.Pool)
}

func (T *Module) Cancel(ctx context.Context, key fed.BackendKey) {
	T.pool.Cancel(ctx, key)
}

var _ gat.Handler = (*Module)(nil)
var _ gat.MetricsHandler = (*Module)(nil)
var _ gat.CancellableHandler = (*Module)(nil)
var _ caddy.Module = (*Module)(nil)
var _ caddy.Provisioner = (*Module)(nil)
