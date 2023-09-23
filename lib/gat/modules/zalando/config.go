package zalando

import (
	"errors"

	"gfx.cafe/util/go/gun"
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
