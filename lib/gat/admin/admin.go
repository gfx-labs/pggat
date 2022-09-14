package admin

import (
	"errors"
	"fmt"
	"gfx.cafe/gfx/pggat/lib/config"
	"gfx.cafe/gfx/pggat/lib/gat"
)

// The admin database, implemented through the gat.Pool interface, allowing it to be added to any existing Gat

import (
	"context"
	"gfx.cafe/gfx/pggat/lib/gat/protocol"
)

func getServerInfo(g gat.Gat) []*protocol.ParameterStatus {
	return []*protocol.ParameterStatus{
		{
			Fields: protocol.FieldsParameterStatus{
				Parameter: "application_name",
				Value:     "",
			},
		},
		{
			Fields: protocol.FieldsParameterStatus{
				Parameter: "client_encoding",
				Value:     "UTF8",
			},
		},
		{
			Fields: protocol.FieldsParameterStatus{
				Parameter: "server_encoding",
				Value:     "UTF8",
			},
		},
		{
			Fields: protocol.FieldsParameterStatus{
				Parameter: "server_encoding",
				Value:     "UTF8",
			},
		},
		{
			Fields: protocol.FieldsParameterStatus{
				Parameter: "server_version",
				Value:     g.Version(),
			},
		},
		{
			Fields: protocol.FieldsParameterStatus{
				Parameter: "DataStyle",
				Value:     "ISO, MDY",
			},
		},
	}
}

func getAdminUser(g gat.Gat) *config.User {
	conf := g.Config()
	return &config.User{
		Name:     conf.General.AdminUsername,
		Password: conf.General.AdminPassword,

		Role:             config.USERROLE_ADMIN,
		PoolSize:         1,
		StatementTimeout: 0,
	}
}

type Pool struct {
	gat      gat.Gat
	connPool *ConnectionPool
}

func NewPool(g gat.Gat) *Pool {
	out := &Pool{
		gat: g,
	}
	out.connPool = &ConnectionPool{
		pool: out,
	}
	return out
}

func (p *Pool) GetUser(name string) (*config.User, error) {
	u := getAdminUser(p.gat)
	if name != u.Name {
		return nil, fmt.Errorf("%w: %s", gat.UserNotFound, name)
	}
	return u, nil
}

func (p *Pool) GetRouter() gat.QueryRouter {
	return nil
}

func (p *Pool) WithUser(name string) (gat.ConnectionPool, error) {
	conf := p.gat.Config()
	if name != conf.General.AdminUsername {
		return nil, fmt.Errorf("%w: %s", gat.UserNotFound, name)
	}
	return p.connPool, nil
}

func (p *Pool) ConnectionPools() []gat.ConnectionPool {
	return []gat.ConnectionPool{
		p.connPool,
	}
}

func (p *Pool) Stats() gat.PoolStats {
	return nil // TODO
}

func (p *Pool) EnsureConfig(c *config.Pool) {
	// TODO
}

var _ gat.Pool = (*Pool)(nil)

type ConnectionPool struct {
	pool *Pool
}

func (c *ConnectionPool) GetUser() *config.User {
	return getAdminUser(c.pool.gat)
}

func (c *ConnectionPool) GetServerInfo() []*protocol.ParameterStatus {
	return getServerInfo(c.pool.gat)
}

func (c *ConnectionPool) Shards() []gat.Shard {
	// this db is within gat, there are no shards
	return nil
}

func (c *ConnectionPool) EnsureConfig(conf *config.Pool) {
	// TODO
}

func (c *ConnectionPool) Describe(ctx context.Context, client gat.Client, describe *protocol.Describe) error {
	return errors.New("not implemented")
}

func (c *ConnectionPool) Execute(ctx context.Context, client gat.Client, execute *protocol.Execute) error {
	return errors.New("not implemented")
}

func (c *ConnectionPool) SimpleQuery(ctx context.Context, client gat.Client, query string) error {
	return errors.New("not implemented")
}

func (c *ConnectionPool) Transaction(ctx context.Context, client gat.Client, query string) error {
	return errors.New("not implemented")
}

func (c *ConnectionPool) CallFunction(ctx context.Context, client gat.Client, payload *protocol.FunctionCall) error {
	return errors.New("not implemented")
}

var _ gat.ConnectionPool = (*ConnectionPool)(nil)
