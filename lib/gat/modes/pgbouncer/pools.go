package pgbouncer

import (
	"crypto/tls"
	"errors"
	"net"
	"strconv"
	"strings"
	"time"

	"tuxpa.in/a/zlog/log"

	"pggat/lib/auth/credentials"
	"pggat/lib/bouncer/backends/v0"
	"pggat/lib/gat"
	"pggat/lib/gat/metrics"
	"pggat/lib/gat/pool"
	"pggat/lib/gat/pool/dialer"
	"pggat/lib/gat/pool/pools/session"
	"pggat/lib/gat/pool/pools/transaction"
	"pggat/lib/gat/pool/recipe"
	"pggat/lib/gsql"
	"pggat/lib/util/maps"
	"pggat/lib/util/strutil"
)

type authQueryResult struct {
	Username string  `sql:"0"`
	Password *string `sql:"1"`
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
		client := new(gsql.Client)
		err := gsql.ExtendedQuery(client, &result, T.Config.PgBouncer.AuthQuery, user)
		if err != nil {
			log.Println("auth query failed:", err)
			return nil
		}
		err = client.Close()
		if err != nil {
			log.Println("auth query failed:", err)
			return nil
		}
		err = authPool.Serve(client, nil, [8]byte{})
		if err != nil && !errors.Is(err, net.ErrClosed) {
			log.Println("auth query failed:", err)
			return nil
		}

		if result.Username != user {
			// user not found
			return nil
		}

		if result.Password != nil {
			password = *result.Password
		}
	}

	creds := credentials.FromString(user, password)

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

	var d dialer.Dialer

	dbCreds := creds
	if db.Password != "" {
		// lookup password
		dbCreds = credentials.FromString(user, db.Password)
	}

	acceptOptions := backends.AcceptOptions{
		SSLMode: T.Config.PgBouncer.ServerTLSSSLMode,
		SSLConfig: &tls.Config{
			InsecureSkipVerify: true, // TODO(garet)
		},
		Credentials:       dbCreds,
		Database:          backendDatabase,
		StartupParameters: db.StartupParameters,
	}

	if db.Host == "" || strings.HasPrefix(db.Host, "/") {
		// connect over unix socket
		dir := db.Host
		port := db.Port
		if !strings.HasPrefix(dir, "/") {
			dir = dir + "/"
		}

		if port == 0 {
			port = 5432
		}

		dir = dir + ".s.PGSQL." + strconv.Itoa(port)

		d = dialer.Net{
			Network:       "unix",
			Address:       dir,
			AcceptOptions: acceptOptions,
		}
	} else {
		var address string
		if db.Port == 0 {
			address = net.JoinHostPort(db.Host, "5432")
		} else {
			address = net.JoinHostPort(db.Host, strconv.Itoa(db.Port))
		}

		// connect over tcp
		d = dialer.Net{
			Network:       "tcp",
			Address:       address,
			AcceptOptions: acceptOptions,
		}
	}

	recipeOptions := recipe.Options{
		Dialer:         d,
		MinConnections: db.MinPoolSize,
		MaxConnections: db.MaxDBConnections,
	}
	if recipeOptions.MinConnections == 0 {
		recipeOptions.MinConnections = T.Config.PgBouncer.MinPoolSize
	}
	if recipeOptions.MaxConnections == 0 {
		recipeOptions.MaxConnections = T.Config.PgBouncer.MaxDBConnections
	}
	r := recipe.NewRecipe(recipeOptions)

	p.AddRecipe("pgbouncer", r)

	return p
}

func (T *Pools) ReadMetrics(metrics *metrics.Pools) {
	T.pools.Range(func(_ poolKey, p *pool.Pool) bool {
		p.ReadMetrics(&metrics.Pool)
		return true
	})
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
