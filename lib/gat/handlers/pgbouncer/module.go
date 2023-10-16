package pgbouncer

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/caddyserver/caddy/v2"
	"go.uber.org/zap"

	"gfx.cafe/gfx/pggat/lib/auth"
	"gfx.cafe/gfx/pggat/lib/bouncer/frontends/v0"
	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool"
	"gfx.cafe/gfx/pggat/lib/perror"
	"gfx.cafe/gfx/pggat/lib/util/dur"
	"gfx.cafe/gfx/pggat/lib/util/flip"
	"gfx.cafe/gfx/pggat/lib/util/slices"

	"gfx.cafe/gfx/pggat/lib/auth/credentials"
	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/gat/metrics"
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

type poolAndCredentials struct {
	pool  pool.Pool
	creds auth.Credentials
}

type Module struct {
	ConfigFile string `json:"config"`
	Config     Config `json:"-"`

	pools maps.TwoKey[string, string, poolAndCredentials]
	mu    sync.RWMutex

	log *zap.Logger
}

func (*Module) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "pggat.handlers.pgbouncer",
		New: func() caddy.Module {
			return new(Module)
		},
	}
}

func (T *Module) Provision(ctx caddy.Context) error {
	T.log = ctx.Logger()

	if T.ConfigFile != "" {
		var err error
		T.Config, err = Load(T.ConfigFile)
		return err
	}
	return nil
}

func (T *Module) Cleanup() error {
	T.mu.Lock()
	defer T.mu.Unlock()

	T.pools.Range(func(user string, database string, p poolAndCredentials) bool {
		p.pool.Close()
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

		authPool, ok := T.lookup(authUser, database)
		if !ok {
			return "", false
		}

		var result authQueryResult

		var b flip.Bank

		inward, outward := gsql.NewPair()
		b.Queue(func() error {
			if err := gsql.ExtendedQuery(inward, &result, T.Config.PgBouncer.AuthQuery, user); err != nil {
				return err
			}
			return inward.Close()
		})

		b.Queue(func() error {
			err := authPool.pool.Serve(outward)
			if err != nil && !errors.Is(err, io.EOF) {
				return err
			}
			return nil
		})

		if err := b.Wait(); err != nil {
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

func (T *Module) tryCreate(user, database string) (poolAndCredentials, bool) {
	db, ok := T.Config.Databases[database]
	if !ok {
		// try wildcard
		db, ok = T.Config.Databases["*"]
		if !ok {
			return poolAndCredentials{}, false
		}
	}

	// try to get password
	password, ok := T.getPassword(user, database)
	if !ok {
		return poolAndCredentials{}, false
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

	var p poolAndCredentials
	p.creds = creds

	switch poolMode {
	case PoolModeSession:
		_ = trackedParameters
		_ = serverLoginRetry
		// TODO(garet) create session pool
	case PoolModeTransaction:
		// TODO(garet) create transaction pool
	default:
		return poolAndCredentials{}, false
	}

	T.mu.Lock()
	defer T.mu.Unlock()
	T.pools.Store(user, database, p)

	serverCreds := creds
	if db.Password != "" {
		// lookup password
		serverCreds = credentials.FromString(user, db.Password)
	}

	dialer := pool.Dialer{
		SSLMode: T.Config.PgBouncer.ServerTLSSSLMode,
		SSLConfig: &tls.Config{
			InsecureSkipVerify: true, // TODO(garet)
		},
		Username:    user,
		Credentials: serverCreds,
		Database:    serverDatabase,
		Parameters:  db.StartupParameters,
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

		dialer.Address = dir
	} else {
		var address string
		if db.Port == 0 {
			address = net.JoinHostPort(db.Host, "5432")
		} else {
			address = net.JoinHostPort(db.Host, strconv.Itoa(db.Port))
		}

		// connect over tcp
		dialer.Address = address
	}

	r := pool.Recipe{
		Dialer:         dialer,
		MinConnections: db.MinPoolSize,
		MaxConnections: db.MaxDBConnections,
	}
	if r.MinConnections == 0 {
		r.MinConnections = T.Config.PgBouncer.MinPoolSize
	}
	if r.MaxConnections == 0 {
		r.MaxConnections = T.Config.PgBouncer.MaxDBConnections
	}

	p.pool.AddRecipe("pgbouncer", &r)

	return p, true
}

func (T *Module) lookup(user, database string) (poolAndCredentials, bool) {
	p, ok := T.pools.Load(user, database)
	if ok {
		return p, true
	}

	// try to create pool
	return T.tryCreate(user, database)
}

func (T *Module) Handle(conn *fed.Conn) error {
	// check ssl
	if T.Config.PgBouncer.ClientTLSSSLMode.IsRequired() {
		if !conn.SSL {
			return perror.New(
				perror.FATAL,
				perror.InvalidPassword,
				"SSL is required",
			)
		}
	}

	// check startup parameters
	for key := range conn.InitialParameters {
		if slices.Contains([]strutil.CIString{
			strutil.MakeCIString("client_encoding"),
			strutil.MakeCIString("datestyle"),
			strutil.MakeCIString("timezone"),
			strutil.MakeCIString("standard_conforming_strings"),
			strutil.MakeCIString("application_name"),
		}, key) {
			continue
		}
		if slices.Contains(T.Config.PgBouncer.TrackExtraParameters, key) {
			continue
		}
		if slices.Contains(T.Config.PgBouncer.IgnoreStartupParameters, key) {
			continue
		}

		return perror.New(
			perror.FATAL,
			perror.FeatureNotSupported,
			fmt.Sprintf(`Startup parameter "%s" is not supported`, key.String()),
		)
	}

	p, ok := T.lookup(conn.User, conn.Database)
	if !ok {
		return nil
	}

	if err := frontends.Authenticate(conn, p.creds); err != nil {
		return err
	}

	return p.pool.Serve(conn)
}

func (T *Module) ReadMetrics(metrics *metrics.Handler) {
	T.mu.RLock()
	defer T.mu.RUnlock()
	T.pools.Range(func(_ string, _ string, p poolAndCredentials) bool {
		p.pool.ReadMetrics(&metrics.Pool)
		return true
	})
}

func (T *Module) Cancel(key fed.BackendKey) {
	T.mu.RLock()
	defer T.mu.RUnlock()
	T.pools.Range(func(_ string, _ string, p poolAndCredentials) bool {
		p.pool.Cancel(key)
		return true
	})
}

var _ gat.Handler = (*Module)(nil)
var _ gat.MetricsHandler = (*Module)(nil)
var _ gat.CancellableHandler = (*Module)(nil)
var _ caddy.Module = (*Module)(nil)
var _ caddy.Provisioner = (*Module)(nil)
var _ caddy.CleanerUpper = (*Module)(nil)
