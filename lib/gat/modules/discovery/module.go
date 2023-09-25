package discovery

import (
	"sync"
	"time"

	"tuxpa.in/a/zlog/log"

	"pggat/lib/auth"
	"pggat/lib/auth/credentials"
	"pggat/lib/gat"
	"pggat/lib/gat/metrics"
	"pggat/lib/gat/pool"
	"pggat/lib/gat/pool/pools/session"
	"pggat/lib/gat/pool/pools/transaction"
	"pggat/lib/gat/pool/recipe"
	"pggat/lib/util/maps"
	"pggat/lib/util/slices"
)

type Module struct {
	config Config

	// this is fine to have no locking because it is only accessed by discoverLoop
	clusters map[string]Cluster

	pools maps.TwoKey[string, string, *pool.Pool]
	mu    sync.RWMutex
}

func NewModule(config Config) (*Module, error) {
	m := &Module{
		config: config,
	}
	if err := m.reconcile(); err != nil {
		return nil, err
	}
	go m.discoverLoop()
	return m, nil
}

func (T *Module) replicaUsername(username string) string {
	return username + "_ro"
}

func (T *Module) creds(user User) (primary, replica auth.Credentials) {
	primary = credentials.FromString(user.Username, user.Password)
	replica = credentials.FromString(T.replicaUsername(user.Username), user.Password)
	return
}

func (T *Module) backendAcceptOptions(username string, creds auth.Credentials, database string) recipe.BackendAcceptOptions {
	return recipe.BackendAcceptOptions{
		SSLMode:           T.config.ServerSSLMode,
		SSLConfig:         T.config.ServerSSLConfig,
		Username:          username,
		Credentials:       creds,
		Database:          database,
		StartupParameters: T.config.ServerStartupParameters,
	}
}

func (T *Module) poolOptions(creds auth.Credentials) pool.Options {
	options := pool.Options{
		Credentials:                creds,
		ServerReconnectInitialTime: T.config.ServerReconnectInitialTime,
		ServerReconnectMaxTime:     T.config.ServerReconnectMaxTime,
		ServerIdleTimeout:          T.config.ServerIdleTimeout,
		TrackedParameters:          T.config.TrackedParameters,
		ServerResetQuery:           T.config.ServerResetQuery,
	}

	switch T.config.PoolMode {
	case "session":
		options = session.Apply(options)
	case "transaction":
		options = transaction.Apply(options)
	default:
		log.Printf("unknown pool mode: %s", T.config.PoolMode)
	}

	return options
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
			acceptOptions := T.backendAcceptOptions(user.Username, primaryCreds, database)

			primary := recipe.Dialer{
				Network:       endpoint.Network,
				Address:       endpoint.Address,
				AcceptOptions: acceptOptions,
			}

			p := T.Lookup(user.Username, database)
			if p == nil {
				continue
			}

			p.RemoveRecipe("primary")
			p.AddRecipe("primary", recipe.NewRecipe(recipe.Options{
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
			acceptOptions := T.backendAcceptOptions(user.Username, primaryCreds, database)

			replicaPoolOptions := T.poolOptions(replicaCreds)
			replicaPool := pool.NewPool(replicaPoolOptions)

			for id, r := range replicas {
				replica := recipe.Dialer{
					Network:       r.Network,
					Address:       r.Address,
					AcceptOptions: acceptOptions,
				}
				replicaPool.AddRecipe(id, recipe.NewRecipe(recipe.Options{
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
			acceptOptions := T.backendAcceptOptions(user.Username, primaryCreds, database)

			p := T.Lookup(replicaUsername, database)
			if p == nil {
				continue
			}

			replica := recipe.Dialer{
				Network:       endpoint.Network,
				Address:       endpoint.Address,
				AcceptOptions: acceptOptions,
			}
			p.AddRecipe(id, recipe.NewRecipe(recipe.Options{
				Dialer: replica,
			}))
		}
	}
}

func (T *Module) removeReplica(users []User, databases []string, id string) {
	for _, user := range users {
		username := T.replicaUsername(user.Username)
		for _, database := range databases {
			p := T.Lookup(username, database)
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
		acceptOptions := T.backendAcceptOptions(user.Username, primaryCreds, database)

		primary := recipe.Dialer{
			Network:       primaryEndpoint.Network,
			Address:       primaryEndpoint.Address,
			AcceptOptions: acceptOptions,
		}

		primaryPoolOptions := T.poolOptions(primaryCreds)

		primaryPool := pool.NewPool(primaryPoolOptions)
		primaryPool.AddRecipe("primary", recipe.NewRecipe(recipe.Options{
			Dialer: primary,
		}))
		T.addPool(user.Username, database, primaryPool)

		if len(replicas) > 0 {
			replicaPoolOptions := T.poolOptions(replicaCreds)

			replicaPool := pool.NewPool(replicaPoolOptions)

			for id, r := range replicas {
				replica := recipe.Dialer{
					Network:       r.Network,
					Address:       r.Address,
					AcceptOptions: acceptOptions,
				}
				replicaPool.AddRecipe(id, recipe.NewRecipe(recipe.Options{
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

		acceptOptions := T.backendAcceptOptions(user.Username, primaryCreds, database)

		primary := recipe.Dialer{
			Network:       primaryEndpoint.Network,
			Address:       primaryEndpoint.Address,
			AcceptOptions: acceptOptions,
		}

		primaryPoolOptions := T.poolOptions(primaryCreds)

		primaryPool := pool.NewPool(primaryPoolOptions)
		primaryPool.AddRecipe("primary", recipe.NewRecipe(recipe.Options{
			Dialer: primary,
		}))
		T.addPool(user.Username, database, primaryPool)

		if len(replicas) > 0 {
			replicaPoolOptions := T.poolOptions(replicaCreds)

			replicaPool := pool.NewPool(replicaPoolOptions)

			for id, r := range replicas {
				replica := recipe.Dialer{
					Network:       r.Network,
					Address:       r.Address,
					AcceptOptions: acceptOptions,
				}
				replicaPool.AddRecipe(id, recipe.NewRecipe(recipe.Options{
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
	clusters, err := T.config.Discoverer.Clusters()
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
	if T.config.ReconcilePeriod != 0 {
		r := time.NewTicker(T.config.ReconcilePeriod)
		defer r.Stop()

		reconcile = r.C
	}
	for {
		select {
		case cluster := <-T.config.Discoverer.Added():
			T.added(cluster)
		case id := <-T.config.Discoverer.Removed():
			T.removed(id)
		case next := <-T.config.Discoverer.Updated():
			T.updated(T.clusters[next.ID], next)
		case <-reconcile:
			err := T.reconcile()
			if err != nil {
				log.Printf("failed to reconcile: %v", err)
			}
		}
	}
}

func (T *Module) addPool(user, database string, p *pool.Pool) {
	T.mu.Lock()
	defer T.mu.Unlock()
	log.Printf("added pool user=%s database=%s", user, database)
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
	log.Printf("removed pool user=%s database=%s", user, database)
	T.pools.Delete(user, database)
}

func (T *Module) GatModule() {}

func (T *Module) ReadMetrics(metrics *metrics.Pools) {
	T.mu.RLock()
	defer T.mu.RUnlock()
	T.pools.Range(func(_ string, _ string, p *pool.Pool) bool {
		p.ReadMetrics(&metrics.Pool)
		return true
	})
}

func (T *Module) Lookup(user, database string) *gat.Pool {
	T.mu.RLock()
	defer T.mu.RUnlock()
	p, _ := T.pools.Load(user, database)
	return p
}

var _ gat.Module = (*Module)(nil)
var _ gat.Provider = (*Module)(nil)
