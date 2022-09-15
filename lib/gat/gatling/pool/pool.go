package pool

import (
	"fmt"
	"gfx.cafe/gfx/pggat/lib/gat/gatling/pool/conn_pool"
	"gfx.cafe/gfx/pggat/lib/gat/gatling/pool/query_router"
	"sync"

	"gfx.cafe/gfx/pggat/lib/config"
	"gfx.cafe/gfx/pggat/lib/gat"
)

type Pool struct {
	c         *config.Pool
	users     map[string]config.User
	connPools map[string]gat.ConnectionPool

	stats *gat.PoolStats

	router query_router.QueryRouter

	mu sync.RWMutex
}

func NewPool(conf *config.Pool) *Pool {
	pool := &Pool{
		connPools: make(map[string]gat.ConnectionPool),
		stats:     gat.NewPoolStats(),
	}
	pool.EnsureConfig(conf)
	return pool
}

func (p *Pool) EnsureConfig(conf *config.Pool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.c = conf
	p.users = make(map[string]config.User)
	for _, user := range conf.Users {
		p.users[user.Name] = *user
	}
	// ensure conn pools
	for name, user := range p.users {
		if existing, ok := p.connPools[name]; ok {
			existing.EnsureConfig(conf)
		} else {
			u := user
			p.connPools[name] = conn_pool.NewConnectionPool(p, conf, &u)
		}
	}
}

func (p *Pool) GetUser(name string) (*config.User, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	user, ok := p.users[name]
	if !ok {
		return nil, fmt.Errorf("%w: %s", gat.UserNotFound, name)
	}
	return &user, nil
}

func (p *Pool) GetRouter() gat.QueryRouter {
	return &p.router
}

func (p *Pool) WithUser(name string) (gat.ConnectionPool, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	pool, ok := p.connPools[name]
	if !ok {
		return nil, fmt.Errorf("no pool for '%s'", name)
	}
	return pool, nil
}

func (p *Pool) ConnectionPools() []gat.ConnectionPool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]gat.ConnectionPool, len(p.connPools))
	idx := 0
	for _, v := range p.connPools {
		out[idx] = v
		idx += 1
	}
	return out
}

func (p *Pool) GetStats() *gat.PoolStats {
	return p.stats
}

var _ gat.Pool = (*Pool)(nil)
