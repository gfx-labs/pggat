package pgbouncer_spilo

import (
	"fmt"

	"github.com/caddyserver/caddy/v2"

	"gfx.cafe/gfx/pggat/lib/gat/handlers/pgbouncer"

	"gfx.cafe/gfx/pggat/lib/gat"

	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

func init() {
	caddy.RegisterModule((*Module)(nil))
}

type Module struct {
	Config

	pgbouncer.Module `json:"-"`
}

func (*Module) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "pggat.handlers.pgbouncer_spilo",
		New: func() caddy.Module {
			return new(Module)
		},
	}
}

func (T *Module) Provision(ctx caddy.Context) error {
	pgb := pgbouncer.Default
	if pgb.Databases == nil {
		pgb.Databases = make(map[string]pgbouncer.Database)
	}
	pgb.Databases["*"] = pgbouncer.Database{
		Host:     T.Host,
		Port:     T.Port,
		AuthUser: T.User,
	}
	pgb.PgBouncer.PoolMode = pgbouncer.PoolMode(T.Mode)
	pgb.PgBouncer.AuthType = "md5"
	pgb.PgBouncer.AuthFile = pgbouncer.AuthFile{
		T.User: T.Password,
	}
	pgb.PgBouncer.AdminUsers = []string{T.User}
	pgb.PgBouncer.AuthQuery = fmt.Sprintf("SELECT * FROM %s.user_lookup($1)", T.Schema)
	pgb.PgBouncer.LogFile = "/var/log/pgbouncer/pgbouncer.log"
	pgb.PgBouncer.PidFile = "/var/run/pgbouncer/pgbouncer.pid"

	pgb.PgBouncer.ServerTLSSSLMode = bounce.SSLModeRequire
	pgb.PgBouncer.ServerTLSCaFile = "/etc/ssl/certs/pgbouncer.crt"
	pgb.PgBouncer.ServerTLSProtocols = []pgbouncer.TLSProtocol{
		pgbouncer.TLSProtocolSecure,
	}
	pgb.PgBouncer.ClientTLSSSLMode = bounce.SSLModeRequire
	pgb.PgBouncer.ClientTLSKeyFile = "/etc/ssl/certs/pgbouncer.key"
	pgb.PgBouncer.ClientTLSCertFile = "/etc/ssl/certs/pgbouncer.crt"

	pgb.PgBouncer.LogConnections = 0
	pgb.PgBouncer.LogDisconnections = 0

	pgb.PgBouncer.DefaultPoolSize = T.DefaultSize
	pgb.PgBouncer.ReservePoolSize = T.ReserveSize
	pgb.PgBouncer.MaxClientConn = T.MaxClientConn
	pgb.PgBouncer.MaxDBConnections = T.MaxDBConn
	pgb.PgBouncer.IdleTransactionTimeout = 600
	pgb.PgBouncer.ServerLoginRetry = 5

	pgb.PgBouncer.IgnoreStartupParameters = []strutil.CIString{
		strutil.MakeCIString("extra_float_digits"),
		strutil.MakeCIString("options"),
	}

	T.Module = pgbouncer.Module{
		Config: pgb,
	}

	return nil
}

var _ gat.Handler = (*Module)(nil)
var _ caddy.Module = (*Module)(nil)
var _ caddy.Provisioner = (*Module)(nil)
