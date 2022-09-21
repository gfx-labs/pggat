package session

import (
	"context"
	"gfx.cafe/gfx/pggat/lib/config"
	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/gat/protocol"
	"gfx.cafe/util/go/generic"
	"runtime"
	"sync/atomic"
)

type Pool struct {
	c        atomic.Pointer[config.Pool]
	user     *config.User
	database gat.Database

	dialer gat.Dialer

	assigned generic.Map[gat.ClientID, gat.Connection]

	servers chan gat.Connection
}

func New(database gat.Database, dialer gat.Dialer, conf *config.Pool, user *config.User) *Pool {
	p := &Pool{
		user:     user,
		database: database,

		dialer: dialer,

		servers: make(chan gat.Connection, 1+runtime.NumCPU()*4),
	}
	p.EnsureConfig(conf)
	return p
}

func (p *Pool) getConnection() (gat.Connection, error) {
	select {
	case c := <-p.servers:
		return c, nil
	default:
		shard := p.c.Load().Shards[0]
		return p.dialer(context.TODO(), p.user, shard, shard.Servers[0])
	}
}

func (p *Pool) returnConnection(c gat.Connection) {
	p.servers <- c
}

func (p *Pool) getOrAssign(client gat.Client) (gat.Connection, error) {
	cid := client.GetId()
	c, ok := p.assigned.Load(cid)
	if !ok {
		get, err := p.getConnection()
		if err != nil {
			return nil, err
		}
		p.assigned.Store(cid, get)
		return get, err
	}
	return c, nil
}

func (p *Pool) GetDatabase() gat.Database {
	return p.database
}

func (p *Pool) EnsureConfig(c *config.Pool) {
	p.c.Store(c)
}

func (p *Pool) OnDisconnect(client gat.Client) {
	cid := client.GetId()
	c, ok := p.assigned.LoadAndDelete(cid)
	if !ok {
		return
	}
	p.servers <- c
}

func (p *Pool) GetUser() *config.User {
	return p.user
}

func (p *Pool) GetServerInfo() []*protocol.ParameterStatus {
	c, err := p.getConnection()
	if err != nil {
		return nil
	}
	defer p.returnConnection(c)
	return c.GetServerInfo()
}

func (p *Pool) Describe(ctx context.Context, client gat.Client, describe *protocol.Describe) error {
	c, err := p.getOrAssign(client)
	if err != nil {
		return err
	}
	return c.Describe(client, describe)
}

func (p *Pool) Execute(ctx context.Context, client gat.Client, execute *protocol.Execute) error {
	c, err := p.getOrAssign(client)
	if err != nil {
		return err
	}
	return c.Execute(client, execute)
}

func (p *Pool) SimpleQuery(ctx context.Context, client gat.Client, query string) error {
	c, err := p.getOrAssign(client)
	if err != nil {
		return err
	}
	return c.SimpleQuery(ctx, client, query)
}

func (p *Pool) Transaction(ctx context.Context, client gat.Client, query string) error {
	c, err := p.getOrAssign(client)
	if err != nil {
		return err
	}
	return c.Transaction(ctx, client, query)
}

func (p *Pool) CallFunction(ctx context.Context, client gat.Client, payload *protocol.FunctionCall) error {
	c, err := p.getOrAssign(client)
	if err != nil {
		return err
	}
	return c.CallFunction(client, payload)
}

var _ gat.Pool = (*Pool)(nil)
