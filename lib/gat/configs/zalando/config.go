package zalando

import (
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"

	"gfx.cafe/util/go/gun"

	"pggat2/lib/auth/credentials"
	"pggat2/lib/gat"
	"pggat2/lib/gat/pools/session"
	"pggat2/lib/gat/pools/transaction"
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
	pooler := gat.NewPooler(gat.PoolerConfig{})

	creds := credentials.Cleartext{
		Username: T.PGUser,
		Password: T.PGPassword,
	}

	user := gat.NewUser(creds)
	pooler.AddUser(user)

	var rawPool gat.RawPool
	if T.PoolerMode == "transaction" {
		rawPool = transaction.NewPool(transaction.Config{})
	} else {
		rawPool = session.NewPool(session.Config{})
	}

	pool := gat.NewPool(rawPool, gat.PoolConfig{})
	user.AddPool("test", pool)

	pool.AddRecipe("zalando", gat.TCPRecipe{
		Address:        net.JoinHostPort(T.PGHost, strconv.Itoa(T.PGPort)),
		Credentials:    creds,
		MinConnections: T.PoolerMinSize,
		MaxConnections: T.PoolerMaxDBConn,
		Database:       "test",
	})

	var bank flip.Bank

	bank.Queue(func() error {
		listen := fmt.Sprintf(":%d", T.PoolerPort)

		listener, err := net.Listen("tcp", listen)
		if err != nil {
			return err
		}

		log.Println("listening on", listen)

		return pooler.ListenAndServe(listener)
	})

	return bank.Wait()
}
