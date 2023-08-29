package zalando

import (
	"errors"
	"fmt"
	"net"
	"strconv"

	"tuxpa.in/a/zlog/log"

	"gfx.cafe/util/go/gun"

	"pggat2/lib/bouncer/backends/v0"
	"pggat2/lib/bouncer/frontends/v0"
	"pggat2/lib/gat/pool/pools/session"

	"pggat2/lib/auth/credentials"
	"pggat2/lib/gat"
	"pggat2/lib/gat/pool"
	"pggat2/lib/util/flip"
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
	pools := new(gat.PoolsMap)

	creds := credentials.Cleartext{
		Username: T.PGUser,
		Password: T.PGPassword,
	}

	var p *pool.Pool
	if T.PoolerMode == "transaction" {
		panic("transaction mode not implemented yet")
		// TODO(garet) p = transaction.NewPool(pool.Options{})
	} else {
		p = session.NewPool(pool.Options{})
	}

	pools.Add(T.PGUser, "test", p)

	p.AddRecipe("zalando", pool.Recipe{
		Dialer: pool.NetDialer{
			Network: "tcp",
			Address: net.JoinHostPort(T.PGHost, strconv.Itoa(T.PGPort)),
			AcceptOptions: backends.AcceptOptions{
				Credentials: creds,
				Database:    "test",
			},
		},
		MinConnections: T.PoolerMinSize,
		MaxConnections: T.PoolerMaxDBConn,
	})

	var bank flip.Bank

	bank.Queue(func() error {
		listen := fmt.Sprintf(":%d", T.PoolerPort)

		log.Printf("listening on %s", listen)

		return gat.ListenAndServe("tcp", listen, frontends.AcceptOptions{}, pools)
	})

	return bank.Wait()
}
