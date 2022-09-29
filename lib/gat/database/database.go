package database

import (
	"gfx.cafe/gfx/pggat/lib/gat/database/query_router"
	"gfx.cafe/gfx/pggat/lib/gat/pool/session"
	"gfx.cafe/gfx/pggat/lib/gat/pool/transaction"
	"sync"

	"gfx.cafe/gfx/pggat/lib/config"
	"gfx.cafe/gfx/pggat/lib/gat"
)

type Database struct {
	c         *config.Pool
	users     map[string]config.User
	connPools map[string]gat.Pool

	stats *gat.PoolStats

	router *query_router.QueryRouter

	dialer gat.Dialer

	mu sync.RWMutex
}

func New(dialer gat.Dialer, conf *config.Pool) *Database {
	pool := &Database{
		connPools: make(map[string]gat.Pool),
		stats:     gat.NewPoolStats(),
		router:    query_router.DefaultRouter(conf),

		dialer: dialer,
	}
	pool.EnsureConfig(conf)
	return pool
}

func (p *Database) EnsureConfig(conf *config.Pool) {
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
			switch p.c.PoolMode {
			case config.POOLMODE_SESSION:
				p.connPools[name] = session.New(p, p.dialer, conf, &u)
			case config.POOLMODE_TXN:
				p.connPools[name] = transaction.New(p, p.dialer, conf, &u)
			}
		}
	}
}

func (p *Database) GetUser(name string) *config.User {
	p.mu.RLock()
	defer p.mu.RUnlock()
	user, ok := p.users[name]
	if !ok {
		return nil
	}
	return &user
}

func (p *Database) GetRouter() gat.QueryRouter {
	return p.router
}

func (p *Database) WithUser(name string) gat.Pool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	pool, ok := p.connPools[name]
	if !ok {
		return nil
	}
	return pool
}

func (p *Database) GetPools() []gat.Pool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]gat.Pool, len(p.connPools))
	idx := 0
	for _, v := range p.connPools {
		out[idx] = v
		idx += 1
	}
	return out
}

func (p *Database) GetStats() *gat.PoolStats {
	return p.stats
}

var _ gat.Database = (*Database)(nil)
