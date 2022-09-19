package session

import (
	"context"
	"gfx.cafe/gfx/pggat/lib/config"
	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/gat/protocol"
	"runtime"
	"sync/atomic"
)

type Pool struct {
	c        atomic.Pointer[config.Pool]
	user     *config.User
	database gat.Database

	dialer gat.Dialer

	assigned map[gat.ClientID]gat.Connection

	servers chan gat.Connection
}

func New(database gat.Database, dialer gat.Dialer, conf *config.Pool, user *config.User) *Pool {
	p := &Pool{
		user:     user,
		database: database,

		dialer: dialer,

		assigned: make(map[gat.ClientID]gat.Connection),

		servers: make(chan gat.Connection, 1+runtime.NumCPU()*4),
	}
	p.EnsureConfig(conf)
	return p
}

func (p *Pool) getConnection() gat.Connection {
	select {
	case c := <-p.servers:
		return c
	default:
		shard := p.c.Load().Shards[0]
		s, _ := p.dialer(context.TODO(), p.user, shard, shard.Servers[0])
		return s
	}
}

func (p *Pool) returnConnection(c gat.Connection) {
	p.servers <- c
}

func (p *Pool) getOrAssign(client gat.Client) gat.Connection {
	cid := client.GetId()
	c, ok := p.assigned[cid]
	if !ok {
		get := p.getConnection()
		p.assigned[cid] = get
		return get
	}
	return c
}

func (p *Pool) GetDatabase() gat.Database {
	return p.database
}

func (p *Pool) EnsureConfig(c *config.Pool) {
	p.c.Store(c)
}

func (p *Pool) OnDisconnect(client gat.Client) {
	cid := client.GetId()
	c, ok := p.assigned[cid]
	if !ok {
		return
	}
	delete(p.assigned, cid)
	p.servers <- c
}

func (p *Pool) GetUser() *config.User {
	return p.user
}

func (p *Pool) GetServerInfo() []*protocol.ParameterStatus {
	c := p.getConnection()
	defer p.returnConnection(c)
	return c.GetServerInfo()
}

func (p *Pool) Describe(ctx context.Context, client gat.Client, describe *protocol.Describe) error {
	return p.getOrAssign(client).Describe(client, describe)
}

func (p *Pool) Execute(ctx context.Context, client gat.Client, execute *protocol.Execute) error {
	return p.getOrAssign(client).Execute(client, execute)
}

func (p *Pool) SimpleQuery(ctx context.Context, client gat.Client, query string) error {
	return p.getOrAssign(client).SimpleQuery(ctx, client, query)
}

func (p *Pool) Transaction(ctx context.Context, client gat.Client, query string) error {
	return p.getOrAssign(client).Transaction(ctx, client, query)
}

func (p *Pool) CallFunction(ctx context.Context, client gat.Client, payload *protocol.FunctionCall) error {
	return p.getOrAssign(client).CallFunction(client, payload)
}

var _ gat.Pool = (*Pool)(nil)
