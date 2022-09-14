package conn_pool

import (
	"context"
	"reflect"
	"runtime"
	"sync"

	"gfx.cafe/gfx/pggat/lib/config"
	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/gat/protocol"
)

type ConnectionPool struct {
	// the pool connection
	c      *config.Pool
	user   *config.User
	pool   gat.Pool
	shards []*config.Shard

	workers []*worker
	// see: https://github.com/golang/go/blob/master/src/runtime/chan.go#L33
	// channels are a thread safe ring buffer implemented via a linked list of goroutines.
	// the idea is that goroutines are cheap, and we can afford to have one per pending request.
	// there is no real reason to implement a complicated worker pool pattern when well, if we're okay with having a 2-4kb overhead per request, then this is fine. trading space for code complexity
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
	return p
}

func (c *ConnectionPool) getWorker() *worker {
	select {
	case w := <-c.workerPool:
		return w
	default:
		c.mu.Lock()
		defer c.mu.Unlock()
		if len(c.workers) < c.user.PoolSize {
			next := &worker{
				w: c,
			}
			c.workers = append(c.workers, next)
			return next
		} else {
			return <-c.workerPool
		}
	}
}

func (c *ConnectionPool) EnsureConfig(conf *config.Pool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.c = conf
	for i, s := range conf.Shards {
		for i >= len(c.shards) {
			c.shards = append(c.shards, s)
		}
		sc := s
		if !reflect.DeepEqual(c.shards[i], &sc) {
			c.shards[i] = sc
			// disconnect all conns using shard i, switch to new conf
			for _, w := range c.workers {
				w.invalidateShard(i)
			}
		}
	}
}

func (c *ConnectionPool) GetUser() *config.User {
	return c.user
}

func (c *ConnectionPool) GetServerInfo() []*protocol.ParameterStatus {
	return c.getWorker().GetServerInfo()
}

func (c *ConnectionPool) Shards() []gat.Shard {
	var shards []gat.Shard
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, w := range c.workers {
		shards = append(shards, w.shards...)
	}
	return shards
}

func (c *ConnectionPool) Describe(ctx context.Context, client gat.Client, d *protocol.Describe) error {
	return c.getWorker().HandleDescribe(ctx, client, d)
}

func (c *ConnectionPool) Execute(ctx context.Context, client gat.Client, e *protocol.Execute) error {
	return c.getWorker().HandleExecute(ctx, client, e)
}

func (c *ConnectionPool) SimpleQuery(ctx context.Context, client gat.Client, q string) error {
	return c.getWorker().HandleSimpleQuery(ctx, client, q)
}

func (c *ConnectionPool) Transaction(ctx context.Context, client gat.Client, q string) error {
	return c.getWorker().HandleTransaction(ctx, client, q)
}

func (c *ConnectionPool) CallFunction(ctx context.Context, client gat.Client, f *protocol.FunctionCall) error {
	return c.getWorker().HandleFunction(ctx, client, f)
}

var _ gat.ConnectionPool = (*ConnectionPool)(nil)
