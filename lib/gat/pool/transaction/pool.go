package transaction

import (
	"context"
	"sync/atomic"
	"time"

	"gfx.cafe/gfx/pggat/lib/config"
	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/gat/protocol"
)

type Pool struct {
	// the database connection
	c           atomic.Pointer[config.Pool]
	user        *config.User
	database    gat.Database
	workerCount atomic.Int64

	dialer gat.Dialer

	workerPool WorkerPool[*Worker]
}

func New(database gat.Database, dialer gat.Dialer, conf *config.Pool, user *config.User) *Pool {
	p := &Pool{
		user:       user,
		database:   database,
		dialer:     dialer,
		workerPool: NewChannelPool[*Worker](user.PoolSize),
	}
	p.EnsureConfig(conf)
	return p
}

func (c *Pool) WithWorkerPool(w WorkerPool[*Worker]) {
	c.workerPool = w
}

func (c *Pool) GetDatabase() gat.Database {
	return c.database
}

func (c *Pool) getWorker() *Worker {
	start := time.Now()
	defer func() {
		c.database.GetStats().AddWaitTime(time.Now().Sub(start).Microseconds())
	}()
	w, ok := c.workerPool.TryGet()
	if ok {
		return w
	} else {
		if c.workerCount.Add(1)-1 < int64(c.user.PoolSize) {
			next := &Worker{
				w: c,
			}
			return next
		} else {
			w := c.workerPool.Get()
			return w
		}
	}
}

func (c *Pool) returnWorker(w *Worker) {
	c.workerPool.Put(w)
}

func (c *Pool) EnsureConfig(conf *config.Pool) {
	c.c.Store(conf)
}

func (c *Pool) OnDisconnect(_ gat.Client) {}

func (c *Pool) GetUser() *config.User {
	return c.user
}

func (c *Pool) GetServerInfo(client gat.Client) []*protocol.ParameterStatus {
	return c.getWorker().GetServerInfo(client)
}

func (c *Pool) Describe(ctx context.Context, client gat.Client, d *protocol.Describe) error {
	return c.getWorker().HandleDescribe(ctx, client, d)
}

func (c *Pool) Execute(ctx context.Context, client gat.Client, e *protocol.Execute) error {
	return c.getWorker().HandleExecute(ctx, client, e)
}

func (c *Pool) SimpleQuery(ctx context.Context, client gat.Client, q string) error {
	// see if the database router can handle it
	handled, err := c.database.GetRouter().TryHandle(client, q)
	if err != nil {
		return err
	}
	if handled {
		return nil
	}
	return c.getWorker().HandleSimpleQuery(ctx, client, q)
}

func (c *Pool) Transaction(ctx context.Context, client gat.Client, q string) error {
	return c.getWorker().HandleTransaction(ctx, client, q)
}

func (c *Pool) CallFunction(ctx context.Context, client gat.Client, f *protocol.FunctionCall) error {
	return c.getWorker().HandleFunction(ctx, client, f)
}

var _ gat.Pool = (*Pool)(nil)
