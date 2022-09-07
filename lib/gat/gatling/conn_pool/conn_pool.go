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
	"time"
)

type request[T any] struct {
	client  gat.Client
	payload T
	ctx     context.Context
	done    context.CancelFunc
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
	c             *config.Pool
	user          *config.User
	pool          gat.Pool
	shards        []shard
	queries       chan request[string]
	functionCalls chan request[*protocol.FunctionCall]

	mu sync.RWMutex
}

func NewConnectionPool(pool gat.Pool, conf *config.Pool, user *config.User) *ConnectionPool {
	p := &ConnectionPool{
		user:          user,
		pool:          pool,
		queries:       make(chan request[string]),
		functionCalls: make(chan request[*protocol.FunctionCall]),
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

	// there are no servers available in the pool, let's make a new connection

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
		func() {
			select {
			case q := <-c.queries:
				defer q.done()
				srv := c.chooseServer(q.payload)
				if srv == nil {
					log.Printf("call to query '%s' failed", q.payload)
					return
				}

				defer srv.mu.Unlock()

				// run the query
				err := srv.primary.Query(q.client, q.ctx, q.payload)
				if err != nil {
					log.Println("error executing query:", err)
				}
			case f := <-c.functionCalls:
				defer f.done()
				srv := c.chooseServer("")
				if srv == nil {
					log.Printf("function call '%+v' failed", f.payload)
					return
				}

				defer srv.mu.Unlock()

				// call the function
				err := srv.primary.CallFunction(f.client, f.payload)
				if err != nil {
					log.Println("error calling function:", err)
				}
			}
		}()
	}
}

func (c *ConnectionPool) GetUser() *config.User {
	return c.user
}

func (c *ConnectionPool) GetServerInfo() []*protocol.ParameterStatus {
	srv := c.chooseServer("")
	if srv == nil {
		return nil
	}
	defer srv.mu.Unlock()
	return srv.primary.GetServerInfo()
}

func (c *ConnectionPool) Query(client gat.Client, ctx context.Context, q string) (context.Context, error) {
	cmdCtx, done := context.WithDeadline(ctx, time.Now().Add(1*time.Second))

	c.queries <- request[string]{
		client:  client,
		payload: q,
		ctx:     cmdCtx,
		done:    done,
	}

	return cmdCtx, nil
}

func (c *ConnectionPool) CallFunction(client gat.Client, ctx context.Context, f *protocol.FunctionCall) (context.Context, error) {
	cmdCtx, done := context.WithDeadline(ctx, time.Now().Add(1*time.Second))

	c.functionCalls <- request[*protocol.FunctionCall]{
		client:  client,
		payload: f,
		ctx:     cmdCtx,
		done:    done,
	}

	return cmdCtx, nil
}

var _ gat.ConnectionPool = (*ConnectionPool)(nil)
