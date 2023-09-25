package zalando

import (
	"fmt"

	"gfx.cafe/gfx/pggat/lib/bouncer"
	"gfx.cafe/gfx/pggat/lib/gat/modules/pgbouncer"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

func NewModule(config Config) (*pgbouncer.Module, error) {
	pgb := pgbouncer.Default
	if pgb.Databases == nil {
		pgb.Databases = make(map[string]pgbouncer.Database)
	}
	pgb.Databases["*"] = pgbouncer.Database{
		Host:     config.PGHost,
		Port:     config.PGPort,
		AuthUser: config.PGUser,
	}
	pgb.PgBouncer.PoolMode = pgbouncer.PoolMode(config.PoolerMode)
	pgb.PgBouncer.ListenPort = config.PoolerPort
	pgb.PgBouncer.ListenAddr = "*"
	pgb.PgBouncer.AuthType = "md5"
	pgb.PgBouncer.AuthFile = pgbouncer.AuthFile{
		config.PGUser: config.PGPassword,
	}
	pgb.PgBouncer.AdminUsers = []string{config.PGUser}
	pgb.PgBouncer.AuthQuery = fmt.Sprintf("SELECT * FROM %s.user_lookup($1)", config.PGSchema)
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

	pgb.PgBouncer.DefaultPoolSize = config.PoolerDefaultSize
	pgb.PgBouncer.ReservePoolSize = config.PoolerReserveSize
	pgb.PgBouncer.MaxClientConn = config.PoolerMaxClientConn
	pgb.PgBouncer.MaxDBConnections = config.PoolerMaxDBConn
	pgb.PgBouncer.IdleTransactionTimeout = 600
	pgb.PgBouncer.ServerLoginRetry = 5

	pgb.PgBouncer.IgnoreStartupParameters = []strutil.CIString{
		strutil.MakeCIString("extra_float_digits"),
		strutil.MakeCIString("options"),
	}

	return pgbouncer.NewModule(pgb)
}
