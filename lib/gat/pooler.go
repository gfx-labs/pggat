package gat

import (
	"net"

	"pggat2/lib/auth"
	"pggat2/lib/bouncer"
	"pggat2/lib/bouncer/frontends/v0"
	"pggat2/lib/middleware/interceptor"
	"pggat2/lib/middleware/middlewares/unterminate"
	"pggat2/lib/util/maps"
	"pggat2/lib/util/slices"
	"pggat2/lib/util/strutil"
	"pggat2/lib/zap"
)

type Pooler struct {
	config PoolerConfig

	// key -> pool for cancellation
	keys maps.RWLocked[[8]byte, *Pool]

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

func (T *Pooler) AddUser(user *User) {
	T.users.Store(user.GetCredentials().GetUsername(), user)
}

func (T *Pooler) RemoveUser(name string) {
	T.users.Delete(name)
}

func (T *Pooler) GetUser(name string) *User {
	user, _ := T.users.Load(name)
	return user
}

func (T *Pooler) GetUserCredentials(user, database string) auth.Credentials {
	u := T.GetUser(user)
	if u == nil {
		return nil
	}
	d := u.GetPool(database)
	if d == nil {
		return nil
	}
	return u.GetCredentials()
}

func (T *Pooler) Cancel(key [8]byte) {
	pool, ok := T.keys.Load(key)
	if !ok {
		return
	}

	pool.Cancel(key)
}

func (T *Pooler) IsStartupParameterAllowed(parameter strutil.CIString) bool {
	return slices.Contains(T.config.AllowedStartupParameters, parameter)
}

func (T *Pooler) Serve(client zap.ReadWriter) {
	defer func() {
		_ = client.Close()
	}()

	client = interceptor.NewInterceptor(
		client,
		unterminate.Unterminate,
	)

	conn, err := frontends.Accept(
		client,
		frontends.AcceptOptions{
			Pooler:                T,
			AllowedStartupOptions: T.config.AllowedStartupParameters,
		},
	)
	if err != nil {
		return
	}

	user := T.GetUser(conn.User)
	if user == nil {
		return
	}

	pool := user.GetPool(conn.Database)
	if pool == nil {
		return
	}

	T.keys.Store(conn.BackendKey, pool)
	defer T.keys.Delete(conn.BackendKey)

	pool.Serve(conn)
}

func (T *Pooler) ListenAndServe(listener net.Listener) error {
	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		go T.Serve(zap.WrapNetConn(conn))
	}
}

var _ bouncer.Pooler = (*Pooler)(nil)
