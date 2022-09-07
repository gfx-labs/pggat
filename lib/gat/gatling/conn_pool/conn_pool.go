package conn_pool

import (
	"context"
	"fmt"
	"gfx.cafe/gfx/pggat/lib/config"
	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/gat/gatling/server"
	"gfx.cafe/gfx/pggat/lib/gat/protocol"
	"log"
	"sync"
)

type query struct {
	query string
	rep   chan<- protocol.Packet
}

type ConnectionPool struct {
	c       *config.Pool
	user    *config.User
	pool    gat.Pool
	servers []*server.Server
	queries chan query

	sync.Mutex
}

const workerCount = 2

func NewConnectionPool(pool gat.Pool, conf *config.Pool, user *config.User) *ConnectionPool {
	p := &ConnectionPool{
		user:    user,
		pool:    pool,
		queries: make(chan query),
	}
	p.EnsureConfig(conf)
	for i := 0; i < workerCount; i++ {
		go p.worker()
	}
	return p
}

func (c *ConnectionPool) EnsureConfig(conf *config.Pool) {
	c.c = conf
	if len(c.servers) == 0 {
		// connect to a server
		shard := c.c.Shards["0"]
		srv := shard.Servers[0] // TODO choose a better way
		s, err := server.Dial(context.Background(), fmt.Sprintf("%s:%d", srv.Host(), srv.Port()), c.user, shard.Database, nil)
		if err != nil {
			log.Println("error connecting to server", err)
		}
		c.servers = append(c.servers, s)
	}
}

func (c *ConnectionPool) worker() {
	for {
		q := <-c.queries
		// TODO ideally this would choose the server based on load, capabilities, etc
		err := c.servers[0].Query(q.query, q.rep)
		if err != nil {
			log.Println(err)
		}
		close(q.rep)
	}
}

func (c *ConnectionPool) GetUser() *config.User {
	return c.user
}

func (c *ConnectionPool) GetServerInfo() []*protocol.ParameterStatus {
	if len(c.servers) > 0 {
		return c.servers[0].GetServerInfo()
	}
	return nil
}

func (c *ConnectionPool) Query(ctx context.Context, q string) (<-chan protocol.Packet, error) {
	rep := make(chan protocol.Packet)

	c.queries <- query{
		query: q,
		rep:   rep,
	}

	return rep, nil
}

var _ gat.ConnectionPool = (*ConnectionPool)(nil)
