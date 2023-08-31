package pgbouncer

import (
	"crypto/tls"
	"net"
	"strconv"
	"time"

	"tuxpa.in/a/zlog/log"

	"pggat2/lib/auth/credentials"
	"pggat2/lib/bouncer/backends/v0"
	"pggat2/lib/fed"
	"pggat2/lib/gat"
	"pggat2/lib/gat/pool"
	"pggat2/lib/gat/pool/pools/session"
	"pggat2/lib/gat/pool/pools/transaction"
	"pggat2/lib/psql"
	"pggat2/lib/util/maps"
	"pggat2/lib/util/strutil"
)

type authQueryResult struct {
	Username string  `ini:"usename"`
	Password *string `ini:"passwd"`
}

type poolKey struct {
	User     string
	Database string
}

type Pools struct {
	Config *Config

	pools maps.RWLocked[poolKey, *pool.Pool]
	keys  maps.RWLocked[[8]byte, *pool.Pool]
}

func NewPools(config *Config) (*Pools, error) {
	pools := &Pools{
		Config: config,
	}

	return pools, nil
}

func (T *Pools) Lookup(user, database string) *pool.Pool {
	key := poolKey{
		User:     user,
		Database: database,
	}
	p, _ := T.pools.Load(key)
	if p != nil {
		return p
	}

	// create pool
	db, ok := T.Config.Databases[database]
	if !ok {
		// try wildcard
		db, ok = T.Config.Databases["*"]
		if !ok {
			return nil
		}
	}

	password, ok := T.Config.PgBouncer.AuthFile.Users[user]
	if !ok {
		// try auth query
		authUser := db.AuthUser
		if authUser == "" {
			authUser = T.Config.PgBouncer.AuthUser
		}

		if authUser == "" {
			// user not present in auth file
			return nil
		}

		if T.Config.PgBouncer.AuthQuery == "" {
			// no auth query
			return nil
		}

		// auth user should be in auth file
		if authUser == user {
			return nil
		}

		authPool := T.Lookup(authUser, database)
		if authPool == nil {
			return nil
		}

		var result authQueryResult
		err := authPool.Do(func(server fed.Conn) error {
			return psql.Query(server, &result, T.Config.PgBouncer.AuthQuery, user)
		})
		if err != nil {
			log.Println("auth query failed:", err)
			return nil
		}

		if result.Password != nil {
			password = *result.Password
		}
	}

	creds := credentials.Cleartext{
		Username: user,
		Password: password, // TODO(garet) md5 and sasl
	}

	backendDatabase := db.DBName
	if backendDatabase == "" {
		backendDatabase = database
	}

	configUser := T.Config.Users[user]

	poolMode := db.PoolMode
	if poolMode == "" {
		poolMode = configUser.PoolMode
	}
	if poolMode == "" {
		poolMode = T.Config.PgBouncer.PoolMode
	}

	trackedParameters := append([]strutil.CIString{
		strutil.MakeCIString("client_encoding"),
		strutil.MakeCIString("datestyle"),
		strutil.MakeCIString("timezone"),
		strutil.MakeCIString("standard_conforming_strings"),
		strutil.MakeCIString("application_name"),
	}, T.Config.PgBouncer.TrackExtraParameters...)

	poolOptions := pool.Options{
		Credentials:       creds,
		TrackedParameters: trackedParameters,
		ServerResetQuery:  T.Config.PgBouncer.ServerResetQuery,
		ServerIdleTimeout: time.Duration(T.Config.PgBouncer.ServerIdleTimeout * float64(time.Second)),
	}

	switch poolMode {
	case PoolModeSession:
		p = session.NewPool(poolOptions)
	case PoolModeTransaction:
		if T.Config.PgBouncer.ServerResetQueryAlways == 0 {
			poolOptions.ServerResetQuery = ""
		}
		p = transaction.NewPool(poolOptions)
	default:
		return nil
	}

	T.pools.Store(poolKey{
		User:     user,
		Database: database,
	}, p)

	if db.Host == "" {
		// connect over unix socket
		// TODO(garet)
	} else {
		var address string
		if db.Port == 0 {
			address = net.JoinHostPort(db.Host, "5432")
		} else {
			address = net.JoinHostPort(db.Host, strconv.Itoa(db.Port))
		}

		creds := creds
		if db.Password != "" {
			// lookup password
			creds.Password = db.Password
		}

		// connect over tcp
		dialer := pool.NetDialer{
			Network: "tcp",
			Address: address,
			AcceptOptions: backends.AcceptOptions{
				SSLMode: T.Config.PgBouncer.ServerTLSSSLMode,
				SSLConfig: &tls.Config{
					InsecureSkipVerify: true, // TODO(garet)
				},
				Credentials:       creds,
				Database:          backendDatabase,
				StartupParameters: db.StartupParameters,
			},
		}
		recipe := pool.Recipe{
			Dialer:         dialer,
			MinConnections: db.MinPoolSize,
			MaxConnections: db.MaxDBConnections,
		}
		if recipe.MinConnections == 0 {
			recipe.MinConnections = T.Config.PgBouncer.MinPoolSize
		}
		if recipe.MaxConnections == 0 {
			recipe.MaxConnections = T.Config.PgBouncer.MaxDBConnections
		}

		p.AddRecipe("pgbouncer", recipe)
	}

	return p
}

func (T *Pools) RegisterKey(key [8]byte, user, database string) {
	p := T.Lookup(user, database)
	if p == nil {
		return
	}
	T.keys.Store(key, p)
}

func (T *Pools) UnregisterKey(key [8]byte) {
	T.keys.Delete(key)
}

func (T *Pools) LookupKey(key [8]byte) *pool.Pool {
	p, _ := T.keys.Load(key)
	return p
}

var _ gat.Pools = (*Pools)(nil)
