package discovery

import (
	"crypto/tls"
	"fmt"
	"sync"
	"time"

	"github.com/caddyserver/caddy/v2"
	"go.uber.org/zap"

	"gfx.cafe/gfx/pggat/lib/auth"
	"gfx.cafe/gfx/pggat/lib/auth/credentials"
	"gfx.cafe/gfx/pggat/lib/bouncer/frontends/v0"
	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool"
	"gfx.cafe/gfx/pggat/lib/gat/metrics"
	"gfx.cafe/gfx/pggat/lib/util/maps"
	"gfx.cafe/gfx/pggat/lib/util/slices"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

func init() {
	caddy.RegisterModule((*Module)(nil))
}

type poolAndCredentials struct {
	pool  pool.Pool
	creds auth.Credentials
}

type Module struct {
	Config

	discoverer Discoverer

	poolFactory pool.PoolFactory

	sslConfig *tls.Config

	serverStartupParameters map[strutil.CIString]string

	closed chan struct{}

	// this is fine to have no locking because it is only accessed by discoverLoop
	clusters map[string]Cluster
	creds    map[User]auth.Credentials

	pools   maps.TwoKey[string, string, poolAndCredentials]
	poolsMu sync.RWMutex

	log *zap.Logger
}

func (*Module) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "pggat.handlers.discovery",
		New: func() caddy.Module {
			return new(Module)
		},
	}
}

func (T *Module) Provision(ctx caddy.Context) error {
	T.log = ctx.Logger()

	if T.Discoverer != nil {
		val, err := ctx.LoadModule(T, "Discoverer")
		if err != nil {
			return fmt.Errorf("loading discoverer module: %v", err)
		}
		T.discoverer = val.(Discoverer)
	}
	if T.Pool != nil {
		val, err := ctx.LoadModule(T, "Pool")
		if err != nil {
			return fmt.Errorf("loading pooler module: %v", err)
		}
		T.poolFactory = val.(pool.PoolFactory)
	}
	if T.ServerSSL != nil {
		val, err := ctx.LoadModule(T, "ServerSSL")
		if err != nil {
			return fmt.Errorf("loading ssl module: %v", err)
		}
		ssl := val.(gat.SSLClient)
		T.sslConfig = ssl.ClientTLSConfig()
	}
	T.serverStartupParameters = make(map[strutil.CIString]string, len(T.ServerStartupParameters))
	for key, value := range T.ServerStartupParameters {
		T.serverStartupParameters[strutil.MakeCIString(key)] = value
	}

	if T.closed != nil {
		return nil
	}
	T.closed = make(chan struct{})

	if err := T.reconcile(); err != nil {
		return err
	}
	go T.discoverLoop()

	return nil
}

func (T *Module) Cleanup() error {
	if T.closed == nil {
		return nil
	}
	close(T.closed)
	T.closed = nil

	T.poolsMu.Lock()
	defer T.poolsMu.Unlock()
	T.pools.Range(func(user string, database string, p poolAndCredentials) bool {
		p.pool.Close()
		T.pools.Delete(user, database)
		return true
	})
	return nil
}

func (T *Module) added(cluster Cluster) {
	if prev, ok := T.clusters[cluster.ID]; ok {
		T.updated(prev, cluster)
		return
	}
	if T.clusters == nil {
		T.clusters = make(map[string]Cluster)
	}
	T.clusters[cluster.ID] = cluster

	for _, user := range cluster.Users {
		T.addUser(cluster.Primary, cluster.Replicas, cluster.Databases, user)
	}
}

func (T *Module) updated(prev, next Cluster) {
	T.clusters[next.ID] = next

	// primary endpoints
	if prev.Primary != next.Primary {
		T.replacePrimary(prev.Users, prev.Databases, next.Primary)
	}

	// replica endpoints
	if len(prev.Replicas) != 0 && len(next.Replicas) == 0 {
		T.removeReplicas(prev.Replicas, prev.Users, prev.Databases)
	} else if len(prev.Replicas) == 0 && len(next.Replicas) != 0 {
		T.addReplicas(next.Replicas, prev.Users, prev.Databases)
	} else {
		// change # of replicas

		for id, nextReplica := range next.Replicas {
			T.addReplica(prev.Users, prev.Databases, id, nextReplica)
		}
		for id := range prev.Replicas {
			_, ok := next.Replicas[id]
			if ok {
				continue // already handled
			}

			T.removeReplica(prev.Users, prev.Databases, id)
		}
	}

	for _, nextUser := range next.Users {
		// test if prevUser exists
		var prevUser User
		var ok bool

		for _, u := range prev.Users {
			if u.Username == nextUser.Username {
				prevUser = u
				ok = true
				break
			}
		}

		if !ok {
			T.addUser(next.Primary, next.Replicas, prev.Databases, nextUser)
		} else if nextUser.Password != prevUser.Password {
			T.removeUser(next.Replicas, prev.Databases, nextUser.Username)
			T.addUser(next.Primary, next.Replicas, prev.Databases, nextUser)
		}
	}
outer:
	for _, prevUser := range prev.Users {
		for _, u := range next.Users {
			if u.Username == prevUser.Username {
				continue outer
			}
		}

		T.removeUser(next.Replicas, prev.Databases, prevUser.Username)
	}

	for _, nextDatabase := range next.Databases {
		if !slices.Contains(prev.Databases, nextDatabase) {
			T.addDatabase(next.Primary, next.Replicas, next.Users, nextDatabase)
		}
	}
	for _, prevDatabase := range prev.Databases {
		if !slices.Contains(next.Databases, prevDatabase) {
			T.removeDatabase(next.Replicas, next.Users, prevDatabase)
		}
	}
}

func (T *Module) addPrimaryNode(user User, database string, primary Node) {
	p := T.getOrAddPool(user, database)

	d := pool.Recipe{
		Dialer: pool.Dialer{
			Address:     primary.Address,
			Username:    user.Username,
			Credentials: p.creds,
			Database:    database,
			SSLMode:     T.ServerSSLMode,
			SSLConfig:   T.sslConfig,
			Parameters:  T.serverStartupParameters,
		},
		Priority: primary.Priority,
	}
	p.pool.AddRecipe("primary", &d)
}

func (T *Module) removePrimaryNode(username, database string) {
	T.removePool(username, database)
}

func (T *Module) addReplicaNodes(user User, database string, replicas map[string]Node) {
	p := T.getOrAddPool(user, database)

	if rp, ok := p.pool.(pool.ReplicaPool); ok {
		for id, replica := range replicas {
			d := pool.Recipe{
				Dialer: pool.Dialer{
					Address:     replica.Address,
					Username:    user.Username,
					Credentials: p.creds,
					Database:    database,
					SSLMode:     T.ServerSSLMode,
					SSLConfig:   T.sslConfig,
					Parameters:  T.serverStartupParameters,
				},
				Priority: replica.Priority,
			}
			rp.AddReplicaRecipe(id, &d)
		}
		return
	}

	rp := T.getOrAddReplicaPool(user, database)
	for id, replica := range replicas {
		d := pool.Recipe{
			Dialer: pool.Dialer{
				Address:     replica.Address,
				Username:    user.Username,
				Credentials: p.creds,
				Database:    database,
				SSLMode:     T.ServerSSLMode,
				SSLConfig:   T.sslConfig,
				Parameters:  T.serverStartupParameters,
			},
			Priority: replica.Priority,
		}
		rp.pool.AddRecipe(id, &d)
	}
}

func (T *Module) removeReplicaNodes(username string, database string, replicas map[string]Node) {
	p, ok := T.getPool(username, database)
	if !ok {
		return
	}

	// remove endpoints from replica pool
	if rp, ok := p.pool.(pool.ReplicaPool); ok {
		for key := range replicas {
			rp.RemoveReplicaRecipe(key)
		}
		return
	}

	// remove replica pool
	T.removeReplicaPool(username, database)
}

func (T *Module) addReplicaNode(user User, database string, id string, replica Node) {
	p := T.getOrAddPool(user, database)

	d := pool.Recipe{
		Dialer: pool.Dialer{
			Address:     replica.Address,
			Username:    user.Username,
			Credentials: p.creds,
			Database:    database,
			SSLMode:     T.ServerSSLMode,
			SSLConfig:   T.sslConfig,
			Parameters:  T.serverStartupParameters,
		},
		Priority: replica.Priority,
	}

	if rp, ok := p.pool.(pool.ReplicaPool); ok {
		rp.AddReplicaRecipe(id, &d)
		return
	}

	rp := T.getOrAddReplicaPool(user, database)
	rp.pool.AddRecipe(id, &d)
}

func (T *Module) removeReplicaNode(username string, database string, id string) {
	p, ok := T.getPool(username, database)
	if !ok {
		return
	}

	// remove endpoints from replica pool
	if rp, ok := p.pool.(pool.ReplicaPool); ok {
		rp.RemoveReplicaRecipe(id)
		return
	}

	// remove replica pool
	rp, ok := T.getReplicaPool(username, database)
	if !ok {
		return
	}
	rp.pool.RemoveRecipe(id)
}

// replacePrimary replaces the primary endpoint.
func (T *Module) replacePrimary(users []User, databases []string, primary Node) {
	for _, user := range users {
		for _, database := range databases {
			T.addPrimaryNode(user, database, primary)
		}
	}
}

// addReplicas adds multiple replicas. Other replicas must not exist.
func (T *Module) addReplicas(replicas map[string]Node, users []User, databases []string) {
	for _, user := range users {
		for _, database := range databases {
			T.addReplicaNodes(user, database, replicas)
		}
	}
}

// removeReplicas removes all replicas.
func (T *Module) removeReplicas(replicas map[string]Node, users []User, databases []string) {
	for _, user := range users {
		for _, database := range databases {
			T.removeReplicaNodes(user.Username, database, replicas)
		}
	}
}

// addReplica adds a single replica.
func (T *Module) addReplica(users []User, databases []string, id string, replica Node) {
	for _, user := range users {
		for _, database := range databases {
			T.addReplicaNode(user, database, id, replica)
		}
	}
}

// removeReplica removes a single replica.
func (T *Module) removeReplica(users []User, databases []string, id string) {
	for _, user := range users {
		for _, database := range databases {
			T.removeReplicaNode(user.Username, database, id)
		}
	}
}

// addUser adds a new user.
func (T *Module) addUser(primary Node, replicas map[string]Node, databases []string, user User) {
	for _, database := range databases {
		T.addPrimaryNode(user, database, primary)
		T.addReplicaNodes(user, database, replicas)
	}
}

// removeUser removes a user.
func (T *Module) removeUser(replicas map[string]Node, databases []string, username string) {
	for _, database := range databases {
		T.removeReplicaNodes(username, database, replicas)
		T.removePrimaryNode(username, database)
	}
}

// addDatabase adds a new database.
func (T *Module) addDatabase(primary Node, replicas map[string]Node, users []User, database string) {
	for _, user := range users {
		T.addPrimaryNode(user, database, primary)
		T.addReplicaNodes(user, database, replicas)
	}
}

// removeDatabase removes a single database.
func (T *Module) removeDatabase(replicas map[string]Node, users []User, database string) {
	for _, user := range users {
		T.removeReplicaNodes(user.Username, database, replicas)
		T.removePrimaryNode(user.Username, database)
	}
}

func (T *Module) removed(id string) {
	cluster, ok := T.clusters[id]
	if !ok {
		return
	}
	delete(T.clusters, id)

	for _, database := range cluster.Databases {
		T.removeDatabase(cluster.Replicas, cluster.Users, database)
	}
}

func (T *Module) reconcile() error {
	clusters, err := T.discoverer.Clusters()
	if err != nil {
		return err
	}

	for _, cluster := range clusters {
		prev, ok := T.clusters[cluster.ID]
		if !ok {
			T.added(cluster)
		} else {
			T.updated(prev, cluster)
		}
	}

	// remove old clusters
outer:
	for id := range T.clusters {
		for _, cluster := range clusters {
			if cluster.ID == id {
				continue outer
			}
		}
		T.removed(id)
	}

	return nil
}

func (T *Module) discoverLoop() {
	var reconcile <-chan time.Time
	if T.ReconcilePeriod != 0 {
		r := time.NewTicker(time.Duration(T.ReconcilePeriod))
		defer r.Stop()

		reconcile = r.C
	}
	for {
		select {
		case cluster := <-T.discoverer.Added():
			T.added(cluster)
		case id := <-T.discoverer.Removed():
			T.removed(id)
		case <-reconcile:
			err := T.reconcile()
			if err != nil {
				T.log.Warn("failed to reconcile", zap.Error(err))
			}
		}
	}
}

func (T *Module) toReplicaUsername(username string) string {
	return username + "_ro"
}

func (T *Module) toReplicaUser(user User) User {
	return User{
		Username: T.toReplicaUsername(user.Username),
		Password: user.Password,
	}
}

func (T *Module) getCreds(user User) auth.Credentials {
	if creds, ok := T.creds[user]; ok {
		return creds
	}
	if T.creds == nil {
		T.creds = make(map[User]auth.Credentials)
	}
	creds := credentials.FromString(user.Username, user.Password)
	T.creds[user] = creds
	return creds
}

func (T *Module) getOrAddPool(user User, database string) poolAndCredentials {
	T.poolsMu.Lock()
	defer T.poolsMu.Unlock()
	if old, ok := T.pools.Load(user.Username, database); ok {
		return old
	}

	creds := T.getCreds(user)
	p := poolAndCredentials{
		pool:  T.poolFactory.NewPool(),
		creds: creds,
	}
	T.pools.Store(user.Username, database, p)
	T.log.Info("added pool", zap.String("user", user.Username), zap.String("database", database))
	return p
}

func (T *Module) getOrAddReplicaPool(user User, database string) poolAndCredentials {
	return T.getOrAddPool(T.toReplicaUser(user), database)
}

func (T *Module) getPool(user, database string) (poolAndCredentials, bool) {
	T.poolsMu.RLock()
	defer T.poolsMu.RUnlock()
	return T.pools.Load(user, database)
}

func (T *Module) getReplicaPool(user, database string) (poolAndCredentials, bool) {
	return T.getPool(T.toReplicaUsername(user), database)
}

func (T *Module) removePool(user, database string) {
	T.poolsMu.Lock()
	defer T.poolsMu.Unlock()
	p, ok := T.pools.Load(user, database)
	if !ok {
		return
	}
	p.pool.Close()
	T.log.Info("removed pool", zap.String("user", user), zap.String("database", database))
	T.pools.Delete(user, database)
}

func (T *Module) removeReplicaPool(user, database string) {
	T.removePool(T.toReplicaUsername(user), database)
}

func (T *Module) ReadMetrics(metrics *metrics.Handler) {
	T.poolsMu.RLock()
	defer T.poolsMu.RUnlock()
	T.pools.Range(func(_ string, _ string, p poolAndCredentials) bool {
		p.pool.ReadMetrics(&metrics.Pool)
		return true
	})
}

func (T *Module) Handle(conn *fed.Conn) error {
	p, ok := T.getPool(conn.User, conn.Database)
	if !ok {
		return nil
	}

	if err := frontends.Authenticate(conn, p.creds); err != nil {
		return err
	}

	return p.pool.Serve(conn)
}

func (T *Module) Cancel(key fed.BackendKey) {
	T.poolsMu.RLock()
	defer T.poolsMu.RUnlock()
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
