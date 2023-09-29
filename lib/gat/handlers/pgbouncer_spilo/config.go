package pgbouncer_spilo

import (
	"strconv"

	"github.com/caddyserver/caddy/v2/caddyconfig"

	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/gat/ssl/servers/x509_key_pair"
)

type Config struct {
	Host                string `json:"host" env:"PGHOST"`
	Port                int    `json:"port" env:"PGPORT"`
	User                string `json:"user" env:"PGUSER"`
	Schema              string `json:"schema" env:"PGSCHEMA"`
	Password            string `json:"password" env:"PGPASSWORD"`
	PoolerPort          int    `json:"pooler_port" env:"CONNECTION_POOLER_PORT"`
	PoolerMode          string `json:"pooler_mode" env:"CONNECTION_POOLER_MODE"`
	PoolerDefaultSize   int    `json:"pooler_default_size" env:"CONNECTION_POOLER_DEFAULT_SIZE"`
	PoolerMinSize       int    `json:"pooler_min_size" env:"CONNECTION_POOLER_MIN_SIZE"`
	PoolerReserveSize   int    `json:"pooler_reserve_size" env:"CONNECTION_POOLER_RESERVE_SIZE"`
	PoolerMaxClientConn int    `json:"pooler_max_client_conn" env:"CONNECTION_POOLER_MAX_CLIENT_CONN"`
	PoolerMaxDBConn     int    `json:"pooler_max_db_conn" env:"CONNECTION_POOLER_MAX_DB_CONN"`
}

func (T Config) Listen() []gat.ListenerConfig {
	ssl := caddyconfig.JSONModuleObject(
		x509_key_pair.Server{
			CertFile: "/etc/ssl/certs/pgbouncer.crt",
			KeyFile:  "/etc/ssl/certs/pgbouncer.key",
		},
		"provider",
		"x509_key_pair",
		nil,
	)

	return []gat.ListenerConfig{
		{
			Address: ":" + strconv.Itoa(T.PoolerPort),
			SSL:     ssl,
		},
	}
}
