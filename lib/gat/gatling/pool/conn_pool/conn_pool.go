package conn_pool

import (
	"context"
	"gfx.cafe/gfx/pggat/lib/config"
	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/gat/protocol"
	"runtime"
	"sync/atomic"
	"time"
)

type ConnectionPool struct {
	// the pool connection
	c           atomic.Pointer[config.Pool]
	user        *config.User
	pool        gat.Pool
	workerCount atomic.Int64

	// see: https://github.com/golang/go/blob/master/src/runtime/chan.go#L33
	// channels are a thread safe ring buffer implemented via a linked list of goroutines.
	// the idea is that goroutines are cheap, and we can afford to have one per pending request.
	// there is no real reason to implement a complicated worker pool pattern when well, if we're okay with having a 2-4kb overhead per request, then this is fine. trading space for code complexity
	workerPool chan *worker
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

func (c *ConnectionPool) GetPool() gat.Pool {
	return c.pool
}

func (c *ConnectionPool) getWorker() *worker {
	start := time.Now()
	defer func() {
		c.pool.GetStats().AddWaitTime(int(time.Now().Sub(start).Microseconds()))
	}()
	select {
	case w := <-c.workerPool:
		return w
	default:
		if c.workerCount.Add(1)-1 < int64(c.user.PoolSize) {
			next := &worker{
				w: c,
			}
			return next
		} else {
			w := <-c.workerPool
			return w
		}
	}
}

func (c *ConnectionPool) EnsureConfig(conf *config.Pool) {
	c.c.Store(conf)
}

func (c *ConnectionPool) GetUser() *config.User {
	return c.user
}

func (c *ConnectionPool) GetServerInfo() []*protocol.ParameterStatus {
	return c.getWorker().GetServerInfo()
}

func (c *ConnectionPool) Describe(ctx context.Context, client gat.Client, d *protocol.Describe) error {
	return c.getWorker().HandleDescribe(ctx, client, d)
}

func (c *ConnectionPool) Execute(ctx context.Context, client gat.Client, e *protocol.Execute) error {
	return c.getWorker().HandleExecute(ctx, client, e)
}

func (c *ConnectionPool) SimpleQuery(ctx context.Context, client gat.Client, q string) error {
	// see if the pool router can handle it
	handled, err := c.pool.GetRouter().TryHandle(client, q)
	if err != nil {
		return err
	}
	if handled {
		return nil
	}
	return c.getWorker().HandleSimpleQuery(ctx, client, q)
}

func (c *ConnectionPool) Transaction(ctx context.Context, client gat.Client, q string) error {
	return c.getWorker().HandleTransaction(ctx, client, q)
}

func (c *ConnectionPool) CallFunction(ctx context.Context, client gat.Client, f *protocol.FunctionCall) error {
	return c.getWorker().HandleFunction(ctx, client, f)
}

var _ gat.ConnectionPool = (*ConnectionPool)(nil)
