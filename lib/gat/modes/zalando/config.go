package zalando

import (
	"errors"
	"fmt"

	"gfx.cafe/util/go/gun"

	"pggat2/lib/bouncer"
	"pggat2/lib/gat/modes/pgbouncer"
	"pggat2/lib/util/strutil"
)

type Config struct {
	PGHost              string `env:"PGHOST"`
	PGPort              int    `env:"PGPORT"`
	PGUser              string `env:"PGUSER"`
	PGSchema            string `env:"PGSCHEMA"`
	PGPassword          string `env:"PGPASSWORD"`
	PoolerPort          int    `env:"CONNECTION_POOLER_PORT"`
	PoolerMode          string `env:"CONNECTION_POOLER_MODE"`
	PoolerDefaultSize   int    `env:"CONNECTION_POOLER_DEFAULT_SIZE"`
	PoolerMinSize       int    `env:"CONNECTION_POOLER_MIN_SIZE"`
	PoolerReserveSize   int    `env:"CONNECTION_POOLER_RESERVE_SIZE"`
	PoolerMaxClientConn int    `env:"CONNECTION_POOLER_MAX_CLIENT_CONN"`
	PoolerMaxDBConn     int    `env:"CONNECTION_POOLER_MAX_DB_CONN"`
}

func Load() (Config, error) {
	var conf Config
	gun.Load(&conf)
	if conf.PoolerMode == "" {
		return Config{}, errors.New("expected pooler mode")
	}

	return conf, nil
}

func (T *Config) ListenAndServe() error {
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
		Users: map[string]string{
			T.PGUser: T.PGPassword,
		},
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

	return pgb.ListenAndServe()
}
