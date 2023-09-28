package transaction

import (
	"github.com/caddyserver/caddy/v2"
	"go.uber.org/zap"

	"gfx.cafe/gfx/pggat/lib/auth"
	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/gat/pool"
)

var PoolingOptions = pool.PoolingOptions{
	NewPooler:               NewPooler,
	ReleaseAfterTransaction: true,
	ParameterStatusSync:     pool.ParameterStatusSyncDynamic,
	ExtendedQuerySync:       true,
}

func init() {
	caddy.RegisterModule((*Pool)(nil))
}

type Pool struct {
	pool.ManagementOptions

	log *zap.Logger
}

func (T *Pool) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "pggat.poolers.transaction",
		New: func() caddy.Module {
			return new(Pool)
		},
	}
}

func (T *Pool) Provision(ctx caddy.Context) error {
	T.log = ctx.Logger()
	return nil
}

func (T *Pool) NewPool(creds auth.Credentials) *gat.Pool {
	return pool.NewPool(pool.Options{
		Credentials: creds,

		PoolingOptions: PoolingOptions,

		ManagementOptions: T.ManagementOptions,

		Logger: T.log,
	})
}

var _ gat.Pooler = (*Pool)(nil)
var _ caddy.Module = (*Pool)(nil)
