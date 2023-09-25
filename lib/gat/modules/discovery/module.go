package discovery

import (
	"sync"
	"time"

	"tuxpa.in/a/zlog/log"

	"pggat/lib/auth/credentials"
	"pggat/lib/gat"
	"pggat/lib/gat/metrics"
	"pggat/lib/gat/pool"
	"pggat/lib/gat/pool/pools/session"
	"pggat/lib/gat/pool/pools/transaction"
	"pggat/lib/gat/pool/recipe"
	"pggat/lib/util/maps"
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

func (T *Module) added(cluster Cluster) {
	if T.clusters == nil {
		T.clusters = make(map[string]Cluster)
	}
	T.clusters[cluster.ID] = cluster

	for _, user := range cluster.Users {
		primaryCreds := credentials.FromString(user.Username, user.Password)
		replicaUsername := user.Username + "_ro"
		replicaCreds := credentials.FromString(replicaUsername, user.Password)
		for _, database := range cluster.Databases {
			acceptOptions := recipe.BackendAcceptOptions{
				SSLMode:           T.config.ServerSSLMode,
				SSLConfig:         T.config.ServerSSLConfig,
				Username:          user.Username,
				Credentials:       primaryCreds,
				Database:          database,
				StartupParameters: T.config.ServerStartupParameters,
			}

			primary := recipe.Dialer{
				Network:       cluster.Primary.Network,
				Address:       cluster.Primary.Address,
				AcceptOptions: acceptOptions,
			}

			primaryPoolOptions := pool.Options{
				Credentials:                primaryCreds,
				ServerReconnectInitialTime: T.config.ServerReconnectInitialTime,
				ServerReconnectMaxTime:     T.config.ServerReconnectMaxTime,
				ServerIdleTimeout:          T.config.ServerIdleTimeout,
				TrackedParameters:          T.config.TrackedParameters,
				ServerResetQuery:           T.config.ServerResetQuery,
			}

			switch T.config.PoolMode {
			case "session":
				primaryPoolOptions = session.Apply(primaryPoolOptions)
			case "transaction":
				primaryPoolOptions = transaction.Apply(primaryPoolOptions)
			default:
				log.Printf("unknown pool mode: %s", T.config.PoolMode)
				return
			}

			primaryPool := pool.NewPool(primaryPoolOptions)
			primaryPool.AddRecipe("primary", recipe.NewRecipe(recipe.Options{
				Dialer: primary,
			}))
			T.addPool(user.Username, database, primaryPool)

			if len(cluster.Replicas) > 0 {
				replicaPoolOptions := primaryPoolOptions
				replicaPoolOptions.Credentials = replicaCreds

				replicaPool := pool.NewPool(replicaPoolOptions)

				for id, r := range cluster.Replicas {
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
}

func (T *Module) updated(prev, next Cluster) {
	T.removed(prev.ID)
	T.added(next) // TODO(garet) actually do something useful
}

func (T *Module) removed(id string) {
	cluster, ok := T.clusters[id]
	if !ok {
		return
	}
	delete(T.clusters, id)

	for _, user := range cluster.Users {
		for _, database := range cluster.Databases {
			T.removePool(user.Username, database)
			if len(cluster.Replicas) > 0 {
				T.removePool(user.Username+"_ro", database)
			}
		}
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
			_ = T.reconcile() // TODO(garet) do something with this error
		}
	}
}

func (T *Module) addPool(user, database string, p *pool.Pool) {
	T.mu.Lock()
	defer T.mu.Unlock()
	log.Printf("added pool user=%s database=%s", user, database)
	T.pools.Store(user, database, p)
}

func (T *Module) removePool(user, database string) {
	T.mu.Lock()
	defer T.mu.Unlock()
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
