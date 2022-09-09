package conn_pool

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"reflect"
	"runtime"
	"sync"

	"gfx.cafe/gfx/pggat/lib/config"
	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/gat/gatling/conn_pool/server"
	"gfx.cafe/gfx/pggat/lib/gat/protocol"
)

type connections struct {
	primary *server.Server
	replica *server.Server

	mu sync.Mutex
}

func (s *connections) choose(role config.ServerRole) *server.Server {
	switch role {
	case config.SERVERROLE_PRIMARY:
		return s.primary
	case config.SERVERROLE_REPLICA:
		if s.replica == nil {
			// fallback to primary
			return s.primary
		}
		return s.replica
	default:
		return nil
	}
}

type shard struct {
	conf  *config.Shard
	conns []*connections

	mu sync.Mutex
}

type ConnectionPool struct {
	// the pool connection
	c      *config.Pool
	user   *config.User
	pool   gat.Pool
	shards []shard

	workerPool chan *worker
	// the lock for config related things
	mu sync.RWMutex
}

func NewConnectionPool(pool gat.Pool, conf *config.Pool, user *config.User) *ConnectionPool {
	p := &ConnectionPool{
		user:       user,
		pool:       pool,
		workerPool: make(chan *worker, 1+runtime.NumCPU()*4),
	}
	p.EnsureConfig(conf)
	for i := 0; i < user.PoolSize; i++ {
		p.add_pool()
	}
	return p
}

func (c *ConnectionPool) add_pool() {
	select {
	case c.workerPool <- &worker{
		w: c,
	}:
	default:
	}
}

func (c *ConnectionPool) EnsureConfig(conf *config.Pool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.c = conf
	for i, s := range conf.Shards {
		for i >= len(c.shards) {
			c.shards = append(c.shards, shard{})
		}
		sc := s
		if !reflect.DeepEqual(c.shards[i].conf, &sc) {
			// disconnect all conns, switch to new conf
			c.shards[i].conns = nil
			c.shards[i].conf = sc
		}
	}
}

func (c *ConnectionPool) chooseShard() *shard {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.shards) == 0 {
		return nil
	}

	// TODO better choose func for sharding, this is not deterministic
	return &c.shards[rand.Intn(len(c.shards))]
}

// chooseConnections locks and returns connections for you to use
func (c *ConnectionPool) chooseConnections() *connections {
	s := c.chooseShard()
	if s == nil {
		log.Println("no available shard for query :(")
		return nil
	}
	// lock the shard
	s.mu.Lock()
	defer s.mu.Unlock()
	// TODO ideally this would choose the server based on load, capabilities, etc. for now we just trylock
	for _, srv := range s.conns {
		if srv.mu.TryLock() {
			return srv
		}
	}
	// there are no conns available in the shard, let's make a new connection
	// connect to servers in shard config
	srvs := &connections{}
	for _, srvConf := range s.conf.Servers {
		srv, err := server.Dial(
			context.Background(),
			fmt.Sprintf("%s:%d", srvConf.Host, srvConf.Port),
			c.user, s.conf.Database,
			srvConf.Username, srvConf.Password,
			nil)
		if err != nil {
			log.Println("failed to connect to server", err)
			continue
		}
		switch srvConf.Role {
		case config.SERVERROLE_PRIMARY:
			srvs.primary = srv
		case config.SERVERROLE_REPLICA:
			srvs.replica = srv
		}
	}
	if srvs.primary == nil {
		return nil
	}
	srvs.mu.Lock()
	s.conns = append(s.conns, srvs)
	return srvs
}

func (c *ConnectionPool) GetUser() *config.User {
	return c.user
}

func (c *ConnectionPool) GetServerInfo() []*protocol.ParameterStatus {
	srv := c.chooseConnections()
	if srv == nil {
		return nil
	}
	defer srv.mu.Unlock()
	return srv.primary.GetServerInfo()
}

func (c *ConnectionPool) Describe(ctx context.Context, client gat.Client, d *protocol.Describe) error {
	return (<-c.workerPool).HandleDescribe(ctx, client, d)
}

func (c *ConnectionPool) Execute(ctx context.Context, client gat.Client, e *protocol.Execute) error {
	return (<-c.workerPool).HandleExecute(ctx, client, e)
}

func (c *ConnectionPool) SimpleQuery(ctx context.Context, client gat.Client, q string) error {
	return (<-c.workerPool).HandleSimpleQuery(ctx, client, q)
}

func (c *ConnectionPool) Transaction(ctx context.Context, client gat.Client, q string) error {
	return (<-c.workerPool).HandleTransaction(ctx, client, q)
}

func (c *ConnectionPool) CallFunction(ctx context.Context, client gat.Client, f *protocol.FunctionCall) error {
	return (<-c.workerPool).HandleFunction(ctx, client, f)
}

var _ gat.ConnectionPool = (*ConnectionPool)(nil)
