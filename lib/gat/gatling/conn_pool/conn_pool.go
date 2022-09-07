package conn_pool

import (
	"context"
	"fmt"
	"gfx.cafe/gfx/pggat/lib/config"
	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/gat/gatling/server"
	"gfx.cafe/gfx/pggat/lib/gat/protocol"
	"log"
)

type ConnectionPool struct {
	c       *config.Pool
	user    *config.User
	pool    gat.Pool
	servers []*server.Server
}

func NewConnectionPool(pool gat.Pool, conf *config.Pool, user *config.User) *ConnectionPool {
	p := &ConnectionPool{
		user: user,
		pool: pool,
	}
	p.EnsureConfig(conf)
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

func (c *ConnectionPool) GetUser() *config.User {
	return c.user
}

func (c *ConnectionPool) GetServerInfo() []*protocol.ParameterStatus {
	if len(c.servers) > 0 {
		return c.servers[0].GetServerInfo()
	}
	return nil
}

func (c *ConnectionPool) Query(ctx context.Context, query string) (<-chan protocol.Packet, error) {
	rep := make(chan protocol.Packet)

	// TODO ideally, this would look at loads, capabilities, etc and choose the server accordingly
	go func() {
		err := c.servers[0].Query(query, rep)
		if err != nil {
			log.Println(err)
		}
		close(rep)
	}()

	return rep, nil
}

var _ gat.ConnectionPool = (*ConnectionPool)(nil)
