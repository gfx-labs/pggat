package conn_pool

import (
	"context"
	"fmt"
	"gfx.cafe/gfx/pggat/lib/config"
	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/gat/gatling/server"
	"gfx.cafe/gfx/pggat/lib/gat/protocol"
	"log"
	"math/rand"
	"reflect"
	"strconv"
	"sync"
)

type query struct {
	query string
	rep   chan<- protocol.Packet
}

type servers struct {
	primary  *server.Server
	replicas []*server.Server

	mu sync.Mutex
}

type shard struct {
	conf    *config.Shard
	servers []*servers

	mu sync.Mutex
}

type ConnectionPool struct {
	c       *config.Pool
	user    *config.User
	pool    gat.Pool
	shards  []shard
	queries chan query

	mu sync.RWMutex
}

func NewConnectionPool(pool gat.Pool, conf *config.Pool, user *config.User) *ConnectionPool {
	p := &ConnectionPool{
		user:    user,
		pool:    pool,
		queries: make(chan query),
	}
	p.EnsureConfig(conf)
	for i := 0; i < user.PoolSize; i++ {
		go p.worker()
	}
	return p
}

func (c *ConnectionPool) EnsureConfig(conf *config.Pool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.c = conf
	for idx, s := range conf.Shards {
		i, err := strconv.Atoi(idx)
		if err != nil {
			log.Printf("expected shard name to be a number, found '%s'", idx)
			continue
		}
		for i >= len(c.shards) {
			c.shards = append(c.shards, shard{})
		}
		sc := s
		if !reflect.DeepEqual(c.shards[i].conf, &sc) {
			// disconnect all connections, switch to new conf
			c.shards[i].servers = nil
			c.shards[i].conf = &sc
		}
	}
}

func (c *ConnectionPool) chooseShard(query string) *shard {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.shards) == 0 {
		return nil
	}

	// TODO better choose func for sharding, this is not deterministic
	return &c.shards[rand.Intn(len(c.shards))]
}

// chooseServer locks and returns a server for you to use
func (c *ConnectionPool) chooseServer(query string) *servers {
	s := c.chooseShard(query)
	if s == nil {
		log.Println("no available shard for query!")
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// TODO ideally this would choose the server based on load, capabilities, etc
	// TODO protect this server from being used by other workers while we use it
	// TODO use c.pool.query_router to route queries
	for _, srv := range s.servers {
		if srv.mu.TryLock() {
			return srv
		}
	}

	// connect to primary server
	// TODO primary server might not be 0, could have no primary server so should fall back to server with role None
	primary, err := server.Dial(context.Background(), fmt.Sprintf("%s:%d", s.conf.Servers[0].Host(), s.conf.Servers[0].Port()), c.user, s.conf.Database, nil)
	if err != nil {
		log.Println("failed to connect to server", err)
		return nil
	}

	srv := &servers{
		primary: primary,
	}
	srv.mu.Lock()

	s.servers = append(s.servers, srv)

	return srv
}

func (c *ConnectionPool) worker() {
	for {
		q := <-c.queries

		srv := c.chooseServer(q.query)
		if srv == nil {
			log.Printf("call to query '%s' failed", q.query)
			continue
		}

		// run the query
		err := srv.primary.Query(q.query, q.rep)
		srv.mu.Unlock()

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
	srv := c.chooseServer("")
	defer srv.mu.Unlock()
	if srv == nil {
		return nil
	}
	return srv.primary.GetServerInfo()
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
