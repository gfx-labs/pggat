package discovery

import (
	"fmt"
	"sync"
	"time"

	"github.com/caddyserver/caddy/v2"
	"go.uber.org/zap"

	"gfx.cafe/gfx/pggat/lib/auth"
	"gfx.cafe/gfx/pggat/lib/auth/credentials"
	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/gat/metrics"
	"gfx.cafe/gfx/pggat/lib/gat/pool"
	"gfx.cafe/gfx/pggat/lib/gat/pool/recipe"
	"gfx.cafe/gfx/pggat/lib/util/maps"
	"gfx.cafe/gfx/pggat/lib/util/slices"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

func init() {
	caddy.RegisterModule((*Module)(nil))
}

type Module struct {
	Config

	discoverer Discoverer

	pooler gat.Pooler
	ssl    gat.SSLClient

	serverStartupParameters map[strutil.CIString]string

	closed chan struct{}

	// this is fine to have no locking because it is only accessed by discoverLoop
	clusters map[string]Cluster

	pools maps.TwoKey[string, string, *pool.Pool]
	mu    sync.RWMutex

	log *zap.Logger
}

func (*Module) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "pggat.providers.discovery",
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
	if T.Pooler != nil {
		val, err := ctx.LoadModule(T, "Pooler")
		if err != nil {
			return fmt.Errorf("loading pooler module: %v", err)
		}
		T.pooler = val.(gat.Pooler)
	}
	if T.ServerSSL != nil {
		val, err := ctx.LoadModule(T, "ServerSSL")
		if err != nil {
			return fmt.Errorf("loading ssl module: %v", err)
		}
		T.ssl = val.(gat.SSLClient)
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

	T.mu.Lock()
	defer T.mu.Unlock()
	T.pools.Range(func(user string, database string, p *pool.Pool) bool {
		p.Close()
		T.pools.Delete(user, database)
		return true
	})
	return nil
}

func (T *Module) replicaUsername(username string) string {
	return username + "_ro"
}

func (T *Module) creds(user User) (primary, replica auth.Credentials) {
	primary = credentials.FromString(user.Username, user.Password)
	replica = credentials.FromString(T.replicaUsername(user.Username), user.Password)
	return
}

func (T *Module) added(cluster Cluster) {
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
		T.removeReplicas(prev.Users, prev.Databases)
	} else if len(prev.Replicas) == 0 && len(next.Replicas) != 0 {
		T.addReplicas(next.Replicas, prev.Users, prev.Databases)
	} else {
		// change # of replicas

		for id, nextReplica := range next.Replicas {
			prevReplica, ok := prev.Replicas[id]
			if !ok {
				T.addReplica(prev.Users, prev.Databases, id, nextReplica)
			} else if prevReplica != nextReplica {
				T.removeReplica(prev.Users, prev.Databases, id)
				T.addReplica(prev.Users, prev.Databases, id, nextReplica)
			}
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

func (T *Module) replacePrimary(users []User, databases []string, endpoint Endpoint) {
	for _, user := range users {
		primaryCreds, _ := T.creds(user)
		for _, database := range databases {
			primary := recipe.Dialer{
				Network:           endpoint.Network,
				Address:           endpoint.Address,
				Username:          user.Username,
				Credentials:       primaryCreds,
				Database:          database,
				SSLMode:           T.ServerSSLMode,
				SSLConfig:         T.ssl.ClientTLSConfig(),
				StartupParameters: T.serverStartupParameters,
			}

			p := T.lookup(user.Username, database)
			if p == nil {
				continue
			}

			p.RemoveRecipe("primary")
			p.AddRecipe("primary", recipe.NewRecipe(recipe.Config{
				Dialer: primary,
			}))
		}
	}
}

func (T *Module) addReplicas(replicas map[string]Endpoint, users []User, databases []string) {
	for _, user := range users {
		replicaUsername := T.replicaUsername(user.Username)
		primaryCreds, replicaCreds := T.creds(user)
		for _, database := range databases {
			replicaPool := T.pooler.NewPool(replicaCreds)

			for id, r := range replicas {
				replica := recipe.Dialer{
					Network:           r.Network,
					Address:           r.Address,
					Username:          user.Username,
					Credentials:       primaryCreds,
					Database:          database,
					SSLMode:           T.ServerSSLMode,
					SSLConfig:         T.ssl.ClientTLSConfig(),
					StartupParameters: T.serverStartupParameters,
				}
				replicaPool.AddRecipe(id, recipe.NewRecipe(recipe.Config{
					Dialer: replica,
				}))
			}

			T.addPool(replicaUsername, database, replicaPool)
		}
	}
}

func (T *Module) removeReplicas(users []User, databases []string) {
	for _, user := range users {
		username := T.replicaUsername(user.Username)
		for _, database := range databases {
			T.removePool(username, database)
		}
	}
}

func (T *Module) addReplica(users []User, databases []string, id string, endpoint Endpoint) {
	for _, user := range users {
		replicaUsername := T.replicaUsername(user.Username)
		primaryCreds, _ := T.creds(user)
		for _, database := range databases {
			p := T.lookup(replicaUsername, database)
			if p == nil {
				continue
			}

			replica := recipe.Dialer{
				Network:           endpoint.Network,
				Address:           endpoint.Address,
				Username:          user.Username,
				Credentials:       primaryCreds,
				Database:          database,
				SSLMode:           T.ServerSSLMode,
				SSLConfig:         T.ssl.ClientTLSConfig(),
				StartupParameters: T.serverStartupParameters,
			}
			p.AddRecipe(id, recipe.NewRecipe(recipe.Config{
				Dialer: replica,
			}))
		}
	}
}

func (T *Module) removeReplica(users []User, databases []string, id string) {
	for _, user := range users {
		username := T.replicaUsername(user.Username)
		for _, database := range databases {
			p := T.lookup(username, database)
			if p == nil {
				continue
			}
			p.RemoveRecipe(id)
		}
	}
}

func (T *Module) addUser(primaryEndpoint Endpoint, replicas map[string]Endpoint, databases []string, user User) {
	replicaUsername := T.replicaUsername(user.Username)
	primaryCreds, replicaCreds := T.creds(user)
	for _, database := range databases {
		base := recipe.Dialer{
			Username:          user.Username,
			Credentials:       primaryCreds,
			Database:          database,
			SSLMode:           T.ServerSSLMode,
			SSLConfig:         T.ssl.ClientTLSConfig(),
			StartupParameters: T.serverStartupParameters,
		}

		primary := base
		primary.Network = primaryEndpoint.Network
		primary.Address = primaryEndpoint.Address

		primaryPool := T.pooler.NewPool(primaryCreds)
		primaryPool.AddRecipe("primary", recipe.NewRecipe(recipe.Config{
			Dialer: primary,
		}))
		T.addPool(user.Username, database, primaryPool)

		if len(replicas) > 0 {
			replicaPool := T.pooler.NewPool(replicaCreds)

			for id, r := range replicas {
				replica := base
				replica.Network = r.Network
				replica.Address = r.Address
				replicaPool.AddRecipe(id, recipe.NewRecipe(recipe.Config{
					Dialer: replica,
				}))
			}

			T.addPool(replicaUsername, database, replicaPool)
		}
	}
}

func (T *Module) removeUser(replicas map[string]Endpoint, databases []string, username string) {
	for _, database := range databases {
		T.removePool(username, database)
	}
	if len(replicas) > 0 {
		user := T.replicaUsername(username)
		for _, database := range databases {
			T.removePool(user, database)
		}
	}
}

func (T *Module) addDatabase(primaryEndpoint Endpoint, replicas map[string]Endpoint, users []User, database string) {
	for _, user := range users {
		replicaUsername := T.replicaUsername(user.Username)
		primaryCreds, replicaCreds := T.creds(user)

		base := recipe.Dialer{
			Username:          user.Username,
			Credentials:       primaryCreds,
			Database:          database,
			SSLMode:           T.ServerSSLMode,
			SSLConfig:         T.ssl.ClientTLSConfig(),
			StartupParameters: T.serverStartupParameters,
		}

		primary := base
		primary.Network = primaryEndpoint.Network
		primary.Address = primaryEndpoint.Address

		primaryPool := T.pooler.NewPool(primaryCreds)
		primaryPool.AddRecipe("primary", recipe.NewRecipe(recipe.Config{
			Dialer: primary,
		}))
		T.addPool(user.Username, database, primaryPool)

		if len(replicas) > 0 {
			replicaPool := T.pooler.NewPool(replicaCreds)

			for id, r := range replicas {
				replica := base
				replica.Network = r.Network
				replica.Address = r.Address
				replicaPool.AddRecipe(id, recipe.NewRecipe(recipe.Config{
					Dialer: replica,
				}))
			}

			T.addPool(replicaUsername, database, replicaPool)
		}
	}
}

func (T *Module) removeDatabase(replicas map[string]Endpoint, users []User, database string) {
	for _, user := range users {
		T.removePool(user.Username, database)
		if len(replicas) > 0 {
			T.removePool(T.replicaUsername(user.Username), database)
		}
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
		r := time.NewTicker(T.ReconcilePeriod.Duration())
		defer r.Stop()

		reconcile = r.C
	}
	for {
		select {
		case cluster := <-T.discoverer.Added():
			T.added(cluster)
		case id := <-T.discoverer.Removed():
			T.removed(id)
		case next := <-T.discoverer.Updated():
			T.updated(T.clusters[next.ID], next)
		case <-reconcile:
			err := T.reconcile()
			if err != nil {
				T.log.Warn("failed to reconcile", zap.Error(err))
			}
		}
	}
}

func (T *Module) addPool(user, database string, p *pool.Pool) {
	T.mu.Lock()
	defer T.mu.Unlock()
	T.log.Info("added pool", zap.String("user", user), zap.String("database", database))
	if old, ok := T.pools.Load(user, database); ok {
		// shouldn't normally get here
		old.Close()
	}
	T.pools.Store(user, database, p)
}

func (T *Module) removePool(user, database string) {
	T.mu.Lock()
	defer T.mu.Unlock()
	p, ok := T.pools.Load(user, database)
	if !ok {
		return
	}
	p.Close()
	T.log.Info("removed pool", zap.String("user", user), zap.String("database", database))
	T.pools.Delete(user, database)
}

func (T *Module) ReadMetrics(metrics *metrics.Pools) {
	T.mu.RLock()
	defer T.mu.RUnlock()
	T.pools.Range(func(_ string, _ string, p *pool.Pool) bool {
		p.ReadMetrics(&metrics.Pool)
		return true
	})
}

func (T *Module) lookup(user, database string) *gat.Pool {
	T.mu.RLock()
	defer T.mu.RUnlock()
	p, _ := T.pools.Load(user, database)
	return p
}

func (T *Module) Lookup(conn fed.Conn) *gat.Pool {
	return T.lookup(conn.User(), conn.Database())
}

var _ gat.Provider = (*Module)(nil)
var _ caddy.Module = (*Module)(nil)
var _ caddy.Provisioner = (*Module)(nil)
var _ caddy.CleanerUpper = (*Module)(nil)
