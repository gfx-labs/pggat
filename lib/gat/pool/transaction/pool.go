package transaction

import (
	"context"
	"errors"
	"fmt"
	"gfx.cafe/gfx/pggat/lib/config"
	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/gat/pool/transaction/shard"
	"gfx.cafe/gfx/pggat/lib/gat/protocol"
	"gfx.cafe/gfx/pggat/lib/metrics"
	"math/rand"
	"sync"
	"time"
)

type Pool struct {
	// config
	c    *config.Pool
	user *config.User

	shards []*shard.Shard
	mu     sync.RWMutex

	// these never change
	database gat.Database
	dialer   gat.Dialer
}

func New(database gat.Database, dialer gat.Dialer, conf *config.Pool, user *config.User) *Pool {
	p := &Pool{
		database: database,
		dialer:   dialer,
	}
	p.EnsureConfig(conf, user)
	return p
}

func (c *Pool) GetDatabase() gat.Database {
	return c.database
}

func (c *Pool) fetchShard(client gat.Client, n int) *shard.Shard {
	c.mu.Lock()
	defer c.mu.Unlock()

	n = n % len(c.shards)

	if c.shards[n] != nil {
		return c.shards[n]
	}

	c.shards[n] = shard.FromConfig(c.dialer, client.GetOptions(), c.c, c.user, c.c.Shards[n], c.database)
	return c.shards[n]
}

func (c *Pool) chooseShard(client gat.Client) *shard.Shard {
	preferred := -1
	if client != nil {
		if p, ok := client.GetRequestedShard(); ok {
			preferred = p
		}

		key := client.GetShardingKey()
		if key != "" {
			// TODO do sharding function on key
		}
	}

	c.mu.RLock()
	if preferred == -1 {
		preferred = rand.Intn(len(c.shards))
	} else {
		preferred = preferred % len(c.shards)
	}
	s := c.shards[preferred]
	c.mu.RUnlock()
	if s != nil {
		return s
	}

	return c.fetchShard(client, preferred)
}

func (c *Pool) EnsureConfig(p *config.Pool, u *config.User) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.c = p
	c.user = u

	c.shards = make([]*shard.Shard, len(p.Shards))
}

func (c *Pool) OnDisconnect(_ gat.Client) {}

func (c *Pool) GetUser() *config.User {
	return c.user
}

var errNoServer = errors.New("fail: no server")
var errPermissionDenied = errors.New("permission denied")

func (c *Pool) GetServerInfo(client gat.Client) []*protocol.ParameterStatus {
	s := c.chooseShard(client)
	conn := s.Choose(config.SERVERROLE_PRIMARY)
	if conn == nil {
		return nil
	}
	defer s.Return(conn)
	conn.SetClient(client)
	client.SetCurrentConn(conn)
	defer conn.SetClient(nil)
	defer client.SetCurrentConn(nil)
	return conn.GetServerInfo()
}

func (c *Pool) Describe(ctx context.Context, client gat.Client, d *protocol.Describe) error {
	if c.user.StatementTimeout != 0 {
		var done context.CancelFunc
		ctx, done = context.WithTimeout(ctx, time.Duration(c.user.StatementTimeout)*time.Millisecond)
		defer done()
	}

	start := time.Now()
	defer func() {
		metrics.RecordTransactionTime(c.GetDatabase().GetName(), c.user.Name, time.Since(start))
	}()

	which := client.GetUnderlyingRole(d)
	if !c.user.Role.CanUse(which) {
		return errPermissionDenied
	}

	s := c.chooseShard(client)
	conn := s.Choose(which)
	if conn == nil {
		return errNoServer
	}
	conn.SetClient(client)
	client.SetCurrentConn(conn)
	err := conn.Describe(ctx, client, d)
	conn.SetClient(nil)
	client.SetCurrentConn(nil)
	s.Return(conn)

	select {
	case <-ctx.Done():
		metrics.RecordTransactionError(c.GetDatabase().GetName(), c.user.Name, ctx.Err())
		return ctx.Err()
	default:
		metrics.RecordTransactionError(c.GetDatabase().GetName(), c.user.Name, err)
		return err
	}
}

func (c *Pool) Execute(ctx context.Context, client gat.Client, e *protocol.Execute) error {
	if c.user.StatementTimeout != 0 {
		var done context.CancelFunc
		ctx, done = context.WithTimeout(ctx, time.Duration(c.user.StatementTimeout)*time.Millisecond)
		defer done()
	}

	start := time.Now()
	defer func() {
		metrics.RecordTransactionTime(c.GetDatabase().GetName(), c.user.Name, time.Since(start))
	}()

	which := client.GetUnderlyingPortalRole(e.Fields.Name)
	if !c.user.Role.CanUse(which) {
		return errPermissionDenied
	}

	s := c.chooseShard(client)
	conn := s.Choose(which)
	if conn == nil {
		return errNoServer
	}
	conn.SetClient(client)
	client.SetCurrentConn(conn)
	err := conn.Execute(ctx, client, e)
	conn.SetClient(nil)
	client.SetCurrentConn(nil)
	s.Return(conn)

	select {
	case <-ctx.Done():
		metrics.RecordTransactionError(c.GetDatabase().GetName(), c.user.Name, ctx.Err())
		return ctx.Err()
	default:
		metrics.RecordTransactionError(c.GetDatabase().GetName(), c.user.Name, err)
		return err
	}
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

	if c.user.StatementTimeout != 0 {
		var done context.CancelFunc
		ctx, done = context.WithTimeout(ctx, time.Duration(c.user.StatementTimeout)*time.Millisecond)
		defer done()
	}

	start := time.Now()
	defer func() {
		metrics.RecordQueryTime(c.GetDatabase().GetName(), c.user.Name, time.Since(start))
	}()

	which, err := c.database.GetRouter().InferRole(q)
	if err != nil {
		return fmt.Errorf("error parsing '%s': %w", q, err)
	}
	if !c.user.Role.CanUse(which) {
		return errPermissionDenied
	}

	s := c.chooseShard(client)
	conn := s.Choose(which)
	if conn == nil {
		return errNoServer
	}
	conn.SetClient(client)
	client.SetCurrentConn(conn)
	err = conn.SimpleQuery(ctx, client, q)
	conn.SetClient(nil)
	client.SetCurrentConn(nil)
	s.Return(conn)

	select {
	case <-ctx.Done():
		metrics.RecordQueryError(c.GetDatabase().GetName(), c.user.Name, ctx.Err())
		return ctx.Err()
	default:
		metrics.RecordQueryError(c.GetDatabase().GetName(), c.user.Name, err)
		return err
	}
}

func (c *Pool) Transaction(ctx context.Context, client gat.Client, q string) error {
	if c.user.StatementTimeout != 0 {
		var done context.CancelFunc
		ctx, done = context.WithTimeout(ctx, time.Duration(c.user.StatementTimeout)*time.Millisecond)
		defer done()
	}

	start := time.Now()
	defer func() {
		metrics.RecordTransactionTime(c.GetDatabase().GetName(), c.user.Name, time.Since(start))
	}()

	which, err := c.database.GetRouter().InferRole(q)
	if err != nil {
		return fmt.Errorf("error parsing '%s': %w", q, err)
	}
	if !c.user.Role.CanUse(which) {
		return errPermissionDenied
	}

	s := c.chooseShard(client)
	conn := s.Choose(which)
	if conn == nil {
		return errNoServer
	}
	conn.SetClient(client)
	client.SetCurrentConn(conn)
	err = conn.Transaction(ctx, client, q)
	conn.SetClient(nil)
	client.SetCurrentConn(nil)
	s.Return(conn)

	select {
	case <-ctx.Done():
		metrics.RecordTransactionError(c.GetDatabase().GetName(), c.user.Name, ctx.Err())
		return ctx.Err()
	default:
		metrics.RecordTransactionError(c.GetDatabase().GetName(), c.user.Name, err)
		return err
	}
}

func (c *Pool) CallFunction(ctx context.Context, client gat.Client, f *protocol.FunctionCall) error {
	if c.user.StatementTimeout != 0 {
		var done context.CancelFunc
		ctx, done = context.WithTimeout(ctx, time.Duration(c.user.StatementTimeout)*time.Millisecond)
		defer done()
	}

	start := time.Now()
	defer func() {
		metrics.RecordQueryTime(c.GetDatabase().GetName(), c.user.Name, time.Since(start))
	}()

	if !c.user.Role.CanUse(config.SERVERROLE_PRIMARY) {
		return errPermissionDenied
	}

	s := c.chooseShard(client)
	conn := s.Choose(config.SERVERROLE_PRIMARY)
	if conn == nil {
		return errNoServer
	}
	conn.SetClient(client)
	client.SetCurrentConn(conn)
	err := conn.CallFunction(ctx, client, f)
	conn.SetClient(nil)
	client.SetCurrentConn(nil)
	s.Return(conn)

	select {
	case <-ctx.Done():
		metrics.RecordTransactionError(c.GetDatabase().GetName(), c.user.Name, ctx.Err())
		return ctx.Err()
	default:
		metrics.RecordTransactionError(c.GetDatabase().GetName(), c.user.Name, err)
		return err
	}
}

var _ gat.Pool = (*Pool)(nil)
