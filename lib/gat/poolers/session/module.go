package session

import (
	"github.com/caddyserver/caddy/v2"
	"go.uber.org/zap"

	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/gat/pool"
)

var PoolingOptions = pool.PoolingConfig{
	NewPooler:               NewPooler,
	ReleaseAfterTransaction: false,
	ParameterStatusSync:     pool.ParameterStatusSyncInitial,
	ExtendedQuerySync:       false,
}

func init() {
	caddy.RegisterModule((*Module)(nil))
}

type Module struct {
	pool.ManagementConfig

	log *zap.Logger
}

func (T *Module) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "pggat.poolers.session",
		New: func() caddy.Module {
			return new(Module)
		},
	}
}

func (T *Module) Provision(ctx caddy.Context) error {
	T.log = ctx.Logger()
	return nil
}

func (T *Module) NewPool() *gat.Pool {
	return pool.NewPool(pool.Config{
		PoolingConfig: PoolingOptions,

		ManagementConfig: T.ManagementConfig,

		Logger: T.log,
	})
}

var _ gat.Pooler = (*Module)(nil)
var _ caddy.Module = (*Module)(nil)
var _ caddy.Provisioner = (*Module)(nil)
