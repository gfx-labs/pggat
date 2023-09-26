package zalando

import (
	"errors"

	"gfx.cafe/util/go/gun"
)

type Config struct {
	PGHost              string `env:"PGHOST" json:"pg_host"`
	PGPort              int    `env:"PGPORT" json:"pg_port"`
	PGUser              string `env:"PGUSER" json:"pg_user"`
	PGSchema            string `env:"PGSCHEMA" json:"pg_schema"`
	PGPassword          string `env:"PGPASSWORD" json:"pg_password"`
	PoolerPort          int    `env:"CONNECTION_POOLER_PORT" json:"pooler_port"`
	PoolerMode          string `env:"CONNECTION_POOLER_MODE" json:"pooler_mode"`
	PoolerDefaultSize   int    `env:"CONNECTION_POOLER_DEFAULT_SIZE" json:"pooler_default_size"`
	PoolerMinSize       int    `env:"CONNECTION_POOLER_MIN_SIZE" json:"pooler_min_size"`
	PoolerReserveSize   int    `env:"CONNECTION_POOLER_RESERVE_SIZE" json:"pooler_reserve_size"`
	PoolerMaxClientConn int    `env:"CONNECTION_POOLER_MAX_CLIENT_CONN" json:"pooler_max_client_conn"`
	PoolerMaxDBConn     int    `env:"CONNECTION_POOLER_MAX_DB_CONN" json:"pooler_max_db_conn"`
}

func Load() (Config, error) {
	var conf Config
	gun.Load(&conf)
	if conf.PoolerMode == "" {
		return Config{}, errors.New("expected pooler mode")
	}

	return conf, nil
}
