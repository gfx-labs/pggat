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

	"github.com/caddyserver/caddy/v2"
	"go.uber.org/zap"

	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/gat/poolers/session"
	"gfx.cafe/gfx/pggat/lib/gat/poolers/transaction"
	"gfx.cafe/gfx/pggat/lib/util/dur"

	"gfx.cafe/gfx/pggat/lib/auth/credentials"
	"gfx.cafe/gfx/pggat/lib/bouncer/backends/v0"
	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/gat/metrics"
	"gfx.cafe/gfx/pggat/lib/gat/pool"
	"gfx.cafe/gfx/pggat/lib/gat/pool/recipe"
	"gfx.cafe/gfx/pggat/lib/gsql"
	"gfx.cafe/gfx/pggat/lib/util/maps"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

type authQueryResult struct {
	Username string `sql:"0"`
	Password string `sql:"1"`
}

func init() {
	caddy.RegisterModule((*Module)(nil))
}

type Module struct {
	ConfigFile string `json:"config"`
	Config     Config `json:"-"`

	pools maps.TwoKey[string, string, *gat.Pool]
	mu    sync.RWMutex

	log *zap.Logger
}

func (*Module) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "pggat.providers.pgbouncer",
		New: func() caddy.Module {
			return new(Module)
		},
	}
}

func (T *Module) Provision(ctx caddy.Context) error {
	T.log = ctx.Logger()

	var err error
	T.Config, err = Load(T.ConfigFile)
	return err
}

func (T *Module) Cleanup() error {
	T.mu.Lock()
	defer T.mu.Unlock()

	T.pools.Range(func(user string, database string, p *gat.Pool) bool {
		p.Close()
		T.pools.Delete(user, database)
		return true
	})

	return nil
}

func (T *Module) getPassword(user, database string) (string, bool) {
	// try to get password
	password, ok := T.Config.PgBouncer.AuthFile[user]
	if !ok {
		// try to run auth query
		if T.Config.PgBouncer.AuthQuery == "" {
			return "", false
		}

		authUser := T.Config.Databases[database].AuthUser
		if authUser == "" {
			authUser = T.Config.PgBouncer.AuthUser
			if authUser == "" {
				return "", false
			}
		}

		authPool := T.lookup(authUser, database)
		if authPool == nil {
			return "", false
		}

		var result authQueryResult
		client := new(gsql.Client)
		err := gsql.ExtendedQuery(client, &result, T.Config.PgBouncer.AuthQuery, user)
		if err != nil {
			T.log.Warn("auth query failed", zap.Error(err))
			return "", false
		}
		err = client.Close()
		if err != nil {
			T.log.Warn("auth query failed", zap.Error(err))
			return "", false
		}
		err = authPool.ServeBot(client)
		if err != nil && !errors.Is(err, io.EOF) {
			T.log.Warn("auth query failed", zap.Error(err))
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
	db, ok := T.Config.Databases[database]
	if !ok {
		// try wildcard
		db, ok = T.Config.Databases["*"]
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

	configUser := T.Config.Users[user]

	poolMode := db.PoolMode
	if poolMode == "" {
		poolMode = configUser.PoolMode
		if poolMode == "" {
			poolMode = T.Config.PgBouncer.PoolMode
		}
	}

	trackedParameters := append([]strutil.CIString{
		strutil.MakeCIString("client_encoding"),
		strutil.MakeCIString("datestyle"),
		strutil.MakeCIString("timezone"),
		strutil.MakeCIString("standard_conforming_strings"),
		strutil.MakeCIString("application_name"),
	}, T.Config.PgBouncer.TrackExtraParameters...)

	serverLoginRetry := dur.Duration(T.Config.PgBouncer.ServerLoginRetry * float64(time.Second))

	poolOptions := pool.Options{
		Credentials: creds,
		ManagementOptions: pool.ManagementOptions{
			TrackedParameters:          trackedParameters,
			ServerResetQuery:           T.Config.PgBouncer.ServerResetQuery,
			ServerIdleTimeout:          dur.Duration(T.Config.PgBouncer.ServerIdleTimeout * float64(time.Second)),
			ServerReconnectInitialTime: serverLoginRetry,
		},
		Logger: T.log,
	}

	switch poolMode {
	case PoolModeSession:
		poolOptions.PoolingOptions = session.PoolingOptions
	case PoolModeTransaction:
		if T.Config.PgBouncer.ServerResetQueryAlways == 0 {
			poolOptions.ServerResetQuery = ""
		}
		poolOptions.PoolingOptions = transaction.PoolingOptions
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
		SSLMode: T.Config.PgBouncer.ServerTLSSSLMode,
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
		recipeOptions.MinConnections = T.Config.PgBouncer.MinPoolSize
	}
	if recipeOptions.MaxConnections == 0 {
		recipeOptions.MaxConnections = T.Config.PgBouncer.MaxDBConnections
	}
	r := recipe.NewRecipe(recipeOptions)

	p.AddRecipe("pgbouncer", r)

	return p
}

func (T *Module) lookup(user, database string) *gat.Pool {
	p, _ := T.pools.Load(user, database)
	if p != nil {
		return p
	}

	// try to create pool
	return T.tryCreate(user, database)
}

func (T *Module) Lookup(conn fed.Conn) *gat.Pool {
	return T.lookup(conn.User(), conn.Database())
}

func (T *Module) ReadMetrics(metrics *metrics.Pools) {
	T.mu.RLock()
	defer T.mu.RUnlock()
	T.pools.Range(func(_ string, _ string, p *gat.Pool) bool {
		p.ReadMetrics(&metrics.Pool)
		return true
	})
}

var _ gat.Provider = (*Module)(nil)
var _ caddy.Module = (*Module)(nil)
var _ caddy.Provisioner = (*Module)(nil)
var _ caddy.CleanerUpper = (*Module)(nil)
