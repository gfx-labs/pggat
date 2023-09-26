package pgbouncer

import (
	"crypto/tls"
	"errors"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"tuxpa.in/a/zlog/log"

	"gfx.cafe/gfx/pggat/lib/auth/credentials"
	"gfx.cafe/gfx/pggat/lib/bouncer/backends/v0"
	"gfx.cafe/gfx/pggat/lib/bouncer/frontends/v0"
	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/gat/metrics"
	"gfx.cafe/gfx/pggat/lib/gat/modules/net_listener"
	"gfx.cafe/gfx/pggat/lib/gat/pool"
	"gfx.cafe/gfx/pggat/lib/gat/pool/pools/session"
	"gfx.cafe/gfx/pggat/lib/gat/pool/pools/transaction"
	"gfx.cafe/gfx/pggat/lib/gat/pool/recipe"
	"gfx.cafe/gfx/pggat/lib/gsql"
	"gfx.cafe/gfx/pggat/lib/util/maps"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

type authQueryResult struct {
	Username string `sql:"0"`
	Password string `sql:"1"`
}

type Module struct {
	Config

	pools maps.TwoKey[string, string, *gat.Pool]
	mu    sync.RWMutex

	tcpListener  net_listener.Module
	unixListener net_listener.Module
}

func (T *Module) Start() error {
	trackedParameters := append([]strutil.CIString{
		strutil.MakeCIString("client_encoding"),
		strutil.MakeCIString("datestyle"),
		strutil.MakeCIString("timezone"),
		strutil.MakeCIString("standard_conforming_strings"),
		strutil.MakeCIString("application_name"),
	}, T.PgBouncer.TrackExtraParameters...)

	allowedStartupParameters := append(trackedParameters, T.PgBouncer.IgnoreStartupParameters...)
	var sslConfig *tls.Config
	if T.PgBouncer.ClientTLSCertFile != "" && T.PgBouncer.ClientTLSKeyFile != "" {
		certificate, err := tls.LoadX509KeyPair(T.PgBouncer.ClientTLSCertFile, T.PgBouncer.ClientTLSKeyFile)
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
		SSLRequired:           T.PgBouncer.ClientTLSSSLMode.IsRequired(),
		SSLConfig:             sslConfig,
		AllowedStartupOptions: allowedStartupParameters,
	}

	if T.PgBouncer.ListenAddr != "" {
		listenAddr := T.PgBouncer.ListenAddr
		if listenAddr == "*" {
			listenAddr = ""
		}

		listen := net.JoinHostPort(listenAddr, strconv.Itoa(T.PgBouncer.ListenPort))

		T.tcpListener = net_listener.Module{
			Config: net_listener.Config{
				Network:       "tcp",
				Address:       listen,
				AcceptOptions: acceptOptions,
			},
		}
		if err := T.tcpListener.Start(); err != nil {
			return err
		}
	}

	// listen on unix socket
	dir := T.PgBouncer.UnixSocketDir
	port := T.PgBouncer.ListenPort

	if !strings.HasSuffix(dir, "/") {
		dir = dir + "/"
	}
	dir = dir + ".s.PGSQL." + strconv.Itoa(port)

	T.unixListener = net_listener.Module{
		Config: net_listener.Config{
			Network:       "unix",
			Address:       dir,
			AcceptOptions: acceptOptions,
		},
	}
	if err := T.unixListener.Start(); err != nil {
		return err
	}

	return nil
}

func (T *Module) Stop() error {
	var err error
	if T.PgBouncer.ListenAddr != "" {
		if err2 := T.tcpListener.Stop(); err2 != nil {
			err = err2
		}
	}

	if err2 := T.unixListener.Stop(); err2 != nil {
		err = err2
	}

	T.mu.Lock()
	defer T.mu.Unlock()
	T.pools.Range(func(user string, database string, p *gat.Pool) bool {
		p.Close()
		T.pools.Delete(user, database)
		return true
	})

	return err
}

func (T *Module) getPassword(user, database string) (string, bool) {
	// try to get password
	password, ok := T.PgBouncer.AuthFile[user]
	if !ok {
		// try to run auth query
		if T.PgBouncer.AuthQuery == "" {
			return "", false
		}

		authUser := T.Databases[database].AuthUser
		if authUser == "" {
			authUser = T.PgBouncer.AuthUser
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
		err := gsql.ExtendedQuery(client, &result, T.PgBouncer.AuthQuery, user)
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
	db, ok := T.Databases[database]
	if !ok {
		// try wildcard
		db, ok = T.Databases["*"]
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

	configUser := T.Users[user]

	poolMode := db.PoolMode
	if poolMode == "" {
		poolMode = configUser.PoolMode
		if poolMode == "" {
			poolMode = T.PgBouncer.PoolMode
		}
	}

	trackedParameters := append([]strutil.CIString{
		strutil.MakeCIString("client_encoding"),
		strutil.MakeCIString("datestyle"),
		strutil.MakeCIString("timezone"),
		strutil.MakeCIString("standard_conforming_strings"),
		strutil.MakeCIString("application_name"),
	}, T.PgBouncer.TrackExtraParameters...)

	serverLoginRetry := time.Duration(T.PgBouncer.ServerLoginRetry * float64(time.Second))

	poolOptions := pool.Options{
		Credentials:                creds,
		TrackedParameters:          trackedParameters,
		ServerResetQuery:           T.PgBouncer.ServerResetQuery,
		ServerIdleTimeout:          time.Duration(T.PgBouncer.ServerIdleTimeout * float64(time.Second)),
		ServerReconnectInitialTime: serverLoginRetry,
	}

	switch poolMode {
	case PoolModeSession:
		poolOptions = session.Apply(poolOptions)
	case PoolModeTransaction:
		if T.PgBouncer.ServerResetQueryAlways == 0 {
			poolOptions.ServerResetQuery = ""
		}
		poolOptions = transaction.Apply(poolOptions)
	default:
		return nil
	}
	p := pool.NewPool(poolOptions)

	T.mu.Lock()
	defer T.mu.Unlock()
	T.pools.Store(user, database, p)

	var d recipe.Dialer

	serverCreds := creds
	if db.Password != "" {
		// lookup password
		serverCreds = credentials.FromString(user, db.Password)
	}

	acceptOptions := backends.AcceptOptions{
		SSLMode: T.PgBouncer.ServerTLSSSLMode,
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
		recipeOptions.MinConnections = T.PgBouncer.MinPoolSize
	}
	if recipeOptions.MaxConnections == 0 {
		recipeOptions.MaxConnections = T.PgBouncer.MaxDBConnections
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
	T.mu.RLock()
	defer T.mu.RUnlock()
	T.pools.Range(func(_ string, _ string, p *gat.Pool) bool {
		p.ReadMetrics(&metrics.Pool)
		return true
	})
}

func (T *Module) Accept() []<-chan gat.AcceptedConn {
	var accept []<-chan gat.AcceptedConn
	if T.PgBouncer.ListenAddr != "" {
		accept = append(accept, T.tcpListener.Accept()...)
	}
	accept = append(accept, T.unixListener.Accept()...)
	return accept
}

func (T *Module) GatModule() {}

var _ gat.Module = (*Module)(nil)
var _ gat.Provider = (*Module)(nil)
var _ gat.Listener = (*Module)(nil)
var _ gat.Starter = (*Module)(nil)
var _ gat.Stopper = (*Module)(nil)
