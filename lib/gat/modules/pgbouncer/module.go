package pgbouncer

import (
	"crypto/tls"
	"errors"
	"io"
	"net"
	"strconv"
	"strings"
	"time"

	"tuxpa.in/a/zlog/log"

	"pggat/lib/auth/credentials"
	"pggat/lib/bouncer/backends/v0"
	"pggat/lib/bouncer/frontends/v0"
	"pggat/lib/gat"
	"pggat/lib/gat/metrics"
	"pggat/lib/gat/pool"
	"pggat/lib/gat/pool/pools/session"
	"pggat/lib/gat/pool/pools/transaction"
	"pggat/lib/gat/pool/recipe"
	"pggat/lib/gsql"
	"pggat/lib/util/maps"
	"pggat/lib/util/strutil"
)

type authQueryResult struct {
	Username string `sql:"0"`
	Password string `sql:"1"`
}

type Module struct {
	config Config

	pools maps.TwoKey[string, string, *gat.Pool]
}

func NewModule(config Config) (*Module, error) {
	return &Module{
		config: config,
	}, nil
}

func (T *Module) getPassword(user, database string) (string, bool) {
	// try to get password
	password, ok := T.config.PgBouncer.AuthFile[user]
	if !ok {
		// try to run auth query
		if T.config.PgBouncer.AuthQuery == "" {
			return "", false
		}

		authUser := T.config.Databases[database].AuthUser
		if authUser == "" {
			authUser = T.config.PgBouncer.AuthUser
			if authUser == "" {
				return "", false
			}
		}

		authPool := T.Lookup(authUser, database)
		if authPool == nil {
			return "", false
		}

		var result authQueryResult
		client := new(gsql.Client)
		err := gsql.ExtendedQuery(client, &result, T.config.PgBouncer.AuthQuery, user)
		if err != nil {
			log.Println("auth query failed:", err)
			return "", false
		}
		err = client.Close()
		if err != nil {
			log.Println("auth query failed:", err)
			return "", false
		}
		err = authPool.ServeBot(client)
		if err != nil && !errors.Is(err, io.EOF) {
			log.Println("auth query failed:", err)
			return "", false
		}

		if result.Username != user {
			// user not found
			return "", false
		}

		password = result.Password
	}

	return password, true
}

func (T *Module) tryCreate(user, database string) *gat.Pool {
	db, ok := T.config.Databases[database]
	if !ok {
		// try wildcard
		db, ok = T.config.Databases["*"]
		if !ok {
			return nil
		}
	}

	// try to get password
	password, ok := T.getPassword(user, database)
	if !ok {
		return nil
	}

	creds := credentials.FromString(user, password)

	serverDatabase := db.DBName
	if serverDatabase == "" {
		serverDatabase = database
	}

	configUser := T.config.Users[user]

	poolMode := db.PoolMode
	if poolMode == "" {
		poolMode = configUser.PoolMode
		if poolMode == "" {
			poolMode = T.config.PgBouncer.PoolMode
		}
	}

	trackedParameters := append([]strutil.CIString{
		strutil.MakeCIString("client_encoding"),
		strutil.MakeCIString("datestyle"),
		strutil.MakeCIString("timezone"),
		strutil.MakeCIString("standard_conforming_strings"),
		strutil.MakeCIString("application_name"),
	}, T.config.PgBouncer.TrackExtraParameters...)

	serverLoginRetry := time.Duration(T.config.PgBouncer.ServerLoginRetry * float64(time.Second))

	poolOptions := pool.Options{
		Credentials:                creds,
		TrackedParameters:          trackedParameters,
		ServerResetQuery:           T.config.PgBouncer.ServerResetQuery,
		ServerIdleTimeout:          time.Duration(T.config.PgBouncer.ServerIdleTimeout * float64(time.Second)),
		ServerReconnectInitialTime: serverLoginRetry,
	}

	switch poolMode {
	case PoolModeSession:
		poolOptions = session.Apply(poolOptions)
	case PoolModeTransaction:
		if T.config.PgBouncer.ServerResetQueryAlways == 0 {
			poolOptions.ServerResetQuery = ""
		}
		poolOptions = transaction.Apply(poolOptions)
	default:
		return nil
	}
	p := pool.NewPool(poolOptions)

	T.pools.Store(user, database, p)

	var d recipe.Dialer

	serverCreds := creds
	if db.Password != "" {
		// lookup password
		serverCreds = credentials.FromString(user, db.Password)
	}

	acceptOptions := backends.AcceptOptions{
		SSLMode: T.config.PgBouncer.ServerTLSSSLMode,
		SSLConfig: &tls.Config{
			InsecureSkipVerify: true, // TODO(garet)
		},
		Username:          user,
		Credentials:       serverCreds,
		Database:          serverDatabase,
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

		d = recipe.Dialer{
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
		d = recipe.Dialer{
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
		recipeOptions.MinConnections = T.config.PgBouncer.MinPoolSize
	}
	if recipeOptions.MaxConnections == 0 {
		recipeOptions.MaxConnections = T.config.PgBouncer.MaxDBConnections
	}
	r := recipe.NewRecipe(recipeOptions)

	p.AddRecipe("pgbouncer", r)

	return p
}

func (T *Module) Lookup(user, database string) *gat.Pool {
	p, _ := T.pools.Load(user, database)
	if p != nil {
		return p
	}

	// try to create pool
	return T.tryCreate(user, database)
}

func (T *Module) ReadMetrics(metrics *metrics.Pools) {
	T.pools.Range(func(_ string, _ string, p *gat.Pool) bool {
		p.ReadMetrics(&metrics.Pool)
		return true
	})
}

func (T *Module) Endpoints() []gat.Endpoint {
	trackedParameters := append([]strutil.CIString{
		strutil.MakeCIString("client_encoding"),
		strutil.MakeCIString("datestyle"),
		strutil.MakeCIString("timezone"),
		strutil.MakeCIString("standard_conforming_strings"),
		strutil.MakeCIString("application_name"),
	}, T.config.PgBouncer.TrackExtraParameters...)

	allowedStartupParameters := append(trackedParameters, T.config.PgBouncer.IgnoreStartupParameters...)
	var sslConfig *tls.Config
	if T.config.PgBouncer.ClientTLSCertFile != "" && T.config.PgBouncer.ClientTLSKeyFile != "" {
		certificate, err := tls.LoadX509KeyPair(T.config.PgBouncer.ClientTLSCertFile, T.config.PgBouncer.ClientTLSKeyFile)
		if err != nil {
			log.Printf("error loading X509 keypair: %v", err)
		} else {
			sslConfig = &tls.Config{
				Certificates: []tls.Certificate{
					certificate,
				},
			}
		}
	}

	acceptOptions := frontends.AcceptOptions{
		SSLRequired:           T.config.PgBouncer.ClientTLSSSLMode.IsRequired(),
		SSLConfig:             sslConfig,
		AllowedStartupOptions: allowedStartupParameters,
	}

	var endpoints []gat.Endpoint

	if T.config.PgBouncer.ListenAddr != "" {
		listenAddr := T.config.PgBouncer.ListenAddr
		if listenAddr == "*" {
			listenAddr = ""
		}

		listen := net.JoinHostPort(listenAddr, strconv.Itoa(T.config.PgBouncer.ListenPort))

		endpoints = append(endpoints, gat.Endpoint{
			Network:       "tcp",
			Address:       listen,
			AcceptOptions: acceptOptions,
		})
	}

	// listen on unix socket
	dir := T.config.PgBouncer.UnixSocketDir
	port := T.config.PgBouncer.ListenPort

	if !strings.HasSuffix(dir, "/") {
		dir = dir + "/"
	}
	dir = dir + ".s.PGSQL." + strconv.Itoa(port)

	endpoints = append(endpoints, gat.Endpoint{
		Network:       "unix",
		Address:       dir,
		AcceptOptions: acceptOptions,
	})

	return endpoints
}

func (T *Module) GatModule() {}

var _ gat.Module = (*Module)(nil)
var _ gat.Provider = (*Module)(nil)
var _ gat.Listener = (*Module)(nil)
