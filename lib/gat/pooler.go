package gat

import (
	"net"

	"pggat2/lib/auth"
	"pggat2/lib/bouncer/frontends/v0"
	"pggat2/lib/middleware/interceptor"
	"pggat2/lib/middleware/middlewares/unterminate"
	"pggat2/lib/util/maps"
	"pggat2/lib/util/strutil"
	"pggat2/lib/zap"
)

type Pooler struct {
	config PoolerConfig

	users maps.RWLocked[string, *User]
}

type PoolerConfig struct {
	AllowedStartupParameters []strutil.CIString
}

func NewPooler(config PoolerConfig) *Pooler {
	return &Pooler{
		config: config,
	}
}

func (T *Pooler) AddUser(name string, user *User) {
	T.users.Store(name, user)
}

func (T *Pooler) RemoveUser(name string) {
	T.users.Delete(name)
}

func (T *Pooler) GetUser(name string) *User {
	user, _ := T.users.Load(name)
	return user
}

func (T *Pooler) Serve(client zap.ReadWriter) {
	client = interceptor.NewInterceptor(
		client,
		unterminate.Unterminate,
	)

	username, database, startupParameters, err := frontends.Accept(
		client,
		func(username, database string) (auth.Credentials, bool) {
			user := T.GetUser(username)
			if user == nil {
				return nil, false
			}
			pool := user.GetPool(database)
			if pool == nil {
				return nil, false
			}
			return user.GetCredentials(), true
		},
		T.config.AllowedStartupParameters,
	)
	if err != nil {
		_ = client.Close()
		return
	}

	user := T.GetUser(username)
	if user == nil {
		_ = client.Close()
		return
	}

	pool := user.GetPool(database)
	if pool == nil {
		_ = client.Close()
		return
	}

	pool.Serve(client, startupParameters)
}

func (T *Pooler) ListenAndServe(listener net.Listener) error {
	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		go T.Serve(zap.WrapIOReadWriter(conn))
	}
}
