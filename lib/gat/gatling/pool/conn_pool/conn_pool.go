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
	for i := 0; i < user.PoolSize; i++ {
		p.addWorker()
	}
	return p
}

func (c *ConnectionPool) addWorker() {
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
			c.shards = append(c.shards, s)
		}
		sc := s
		if !reflect.DeepEqual(c.shards[i], &sc) {
			// disconnect all conns, switch to new conf
			// TODO notify workers that they need to update that shard
			c.shards[i] = sc
		}
	}
}

func (c *ConnectionPool) GetUser() *config.User {
	return c.user
}

func (c *ConnectionPool) GetServerInfo() []*protocol.ParameterStatus {
	return (<-c.workerPool).GetServerInfo()
}

func (c *ConnectionPool) Shards() []gat.Shard {
	// TODO go through each worker
	return nil
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
