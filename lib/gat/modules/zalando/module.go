package zalando

import (
	"fmt"

	"gfx.cafe/gfx/pggat/lib/bouncer"
	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/gat/modules/pgbouncer"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

type Module struct {
	Config

	pgbouncer.Module `json:"-"`
}

func (T *Module) Start() error {
	pgb := pgbouncer.Default
	if pgb.Databases == nil {
		pgb.Databases = make(map[string]pgbouncer.Database)
	}
	pgb.Databases["*"] = pgbouncer.Database{
		Host:     T.PGHost,
		Port:     T.PGPort,
		AuthUser: T.PGUser,
	}
	pgb.PgBouncer.PoolMode = pgbouncer.PoolMode(T.PoolerMode)
	pgb.PgBouncer.ListenPort = T.PoolerPort
	pgb.PgBouncer.ListenAddr = "*"
	pgb.PgBouncer.AuthType = "md5"
	pgb.PgBouncer.AuthFile = pgbouncer.AuthFile{
		T.PGUser: T.PGPassword,
	}
	pgb.PgBouncer.AdminUsers = []string{T.PGUser}
	pgb.PgBouncer.AuthQuery = fmt.Sprintf("SELECT * FROM %s.user_lookup($1)", T.PGSchema)
	pgb.PgBouncer.LogFile = "/var/log/pgbouncer/pgbouncer.log"
	pgb.PgBouncer.PidFile = "/var/run/pgbouncer/pgbouncer.pid"

	pgb.PgBouncer.ServerTLSSSLMode = bouncer.SSLModeRequire
	pgb.PgBouncer.ServerTLSCaFile = "/etc/ssl/certs/pgbouncer.crt"
	pgb.PgBouncer.ServerTLSProtocols = []pgbouncer.TLSProtocol{
		pgbouncer.TLSProtocolSecure,
	}
	pgb.PgBouncer.ClientTLSSSLMode = bouncer.SSLModeRequire
	pgb.PgBouncer.ClientTLSKeyFile = "/etc/ssl/certs/pgbouncer.key"
	pgb.PgBouncer.ClientTLSCertFile = "/etc/ssl/certs/pgbouncer.crt"

	pgb.PgBouncer.LogConnections = 0
	pgb.PgBouncer.LogDisconnections = 0

	pgb.PgBouncer.DefaultPoolSize = T.PoolerDefaultSize
	pgb.PgBouncer.ReservePoolSize = T.PoolerReserveSize
	pgb.PgBouncer.MaxClientConn = T.PoolerMaxClientConn
	pgb.PgBouncer.MaxDBConnections = T.PoolerMaxDBConn
	pgb.PgBouncer.IdleTransactionTimeout = 600
	pgb.PgBouncer.ServerLoginRetry = 5

	pgb.PgBouncer.IgnoreStartupParameters = []strutil.CIString{
		strutil.MakeCIString("extra_float_digits"),
		strutil.MakeCIString("options"),
	}

	T.Module = pgbouncer.Module{
		Config: pgb,
	}

	return T.Module.Start()
}

var _ gat.Starter = (*Module)(nil)
