package pool_handler

import (
	"crypto/tls"
	"fmt"
	"strings"

	"github.com/caddyserver/caddy/v2"

	"gfx.cafe/gfx/pggat/lib/auth/credentials"
	"gfx.cafe/gfx/pggat/lib/bouncer/frontends/v0"
	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/gat/metrics"
	"gfx.cafe/gfx/pggat/lib/gat/pool"
	"gfx.cafe/gfx/pggat/lib/gat/pool/pools/basic"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

func init() {
	caddy.RegisterModule((*Module)(nil))
}

type Module struct {
	Config

	pool *basic.Pool
}

func (*Module) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "pggat.handlers.pool",
		New: func() caddy.Module {
			return new(Module)
		},
	}
}

func (T *Module) Provision(ctx caddy.Context) error {
	val, err := ctx.LoadModule(T, "Pooler")
	if err != nil {
		return fmt.Errorf("loading pooler module: %v", err)
	}
	pooler := val.(gat.Pooler)

	var sslConfig *tls.Config
	if T.ServerSSL != nil {
		val, err = ctx.LoadModule(T, "ServerSSL")
		if err != nil {
			return fmt.Errorf("loading ssl module: %v", err)
		}
		ssl := val.(gat.SSLClient)
		sslConfig = ssl.ClientTLSConfig()
	}

	creds := credentials.FromString(T.ServerUsername, T.ServerPassword)
	startupParameters := make(map[strutil.CIString]string, len(T.ServerStartupParameters))
	for key, value := range T.ServerStartupParameters {
		startupParameters[strutil.MakeCIString(key)] = value
	}

	var network string
	if strings.HasPrefix(T.ServerAddress, "/") {
		network = "unix"
	} else {
		network = "tcp"
	}

	d := pool.Dialer{
		Network:           network,
		Address:           T.ServerAddress,
		SSLMode:           T.ServerSSLMode,
		SSLConfig:         sslConfig,
		Username:          T.ServerUsername,
		Credentials:       creds,
		Database:          T.ServerDatabase,
		StartupParameters: startupParameters,
	}

	T.pool = pooler.NewPool()
	T.pool.AddRecipe("pool", pool.NewRecipe(pool.RecipeConfig{
		Dialer: d,
	}))

	return nil
}

func (T *Module) Cleanup() error {
	T.pool.Close()
	return nil
}

func (T *Module) Handle(conn *fed.Conn) error {
	if err := frontends.Authenticate(conn, nil); err != nil {
		return err
	}

	return T.pool.Serve(conn)
}

func (T *Module) Cancel(key fed.BackendKey) {
	T.pool.Cancel(key)
}

func (T *Module) ReadMetrics(metrics *metrics.Handler) {
	T.pool.ReadMetrics(&metrics.Pool)
}

var _ gat.Handler = (*Module)(nil)
var _ gat.MetricsHandler = (*Module)(nil)
var _ gat.CancellableHandler = (*Module)(nil)
var _ caddy.Module = (*Module)(nil)
var _ caddy.Provisioner = (*Module)(nil)
var _ caddy.CleanerUpper = (*Module)(nil)
