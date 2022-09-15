package conn_pool

import (
	"context"
	"fmt"
	"gfx.cafe/gfx/pggat/lib/config"
	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/gat/gatling/pool/conn_pool/shard"
	"gfx.cafe/gfx/pggat/lib/gat/protocol"
	"gfx.cafe/gfx/pggat/lib/gat/protocol/pg_error"
	"sync"
	"time"
)

// a single use worker with an embedded connection pool.
// it wraps a pointer to the connection pool.
type worker struct {
	// the parent connectino pool
	w *ConnectionPool

	shards []gat.Shard

	mu sync.Mutex
}

// ret urn worker to pool
func (w *worker) ret() {
	w.w.workerPool <- w
}

// attempt to connect to a new shard with this worker
func (w *worker) fetchShard(n int) bool {
	if n < 0 || n >= len(w.w.shards) {
		return false
	}

	for len(w.shards) <= n {
		w.shards = append(w.shards, nil)
	}

	w.shards[n] = shard.FromConfig(w.w.user, w.w.shards[n])
	return true
}

func (w *worker) invalidateShard(n int) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.shards[n] = nil
}

func (w *worker) anyShard() gat.Shard {
	w.mu.Lock()
	defer w.mu.Unlock()

	for _, s := range w.shards {
		if s != nil {
			return s
		}
	}

	// we need to fetch a shard
	if w.fetchShard(0) {
		return w.shards[0]
	}

	return nil
}

func (w *worker) chooseShardDescribe(client gat.Client, payload *protocol.Describe) gat.Shard {
	return w.anyShard() // TODO
}

func (w *worker) chooseShardExecute(client gat.Client, payload *protocol.Execute) gat.Shard {
	return w.anyShard() // TODO
}

func (w *worker) chooseShardFn(client gat.Client, fn *protocol.FunctionCall) gat.Shard {
	return w.anyShard() // TODO
}

func (w *worker) chooseShardSimpleQuery(client gat.Client, payload string) gat.Shard {
	return w.anyShard() // TODO
}

func (w *worker) chooseShardTransaction(client gat.Client, payload string) gat.Shard {
	return w.anyShard() // TODO
}

func (w *worker) GetServerInfo() []*protocol.ParameterStatus {
	defer w.ret()

	shard := w.anyShard()
	if shard == nil {
		return nil
	}

	primary := shard.Primary()
	if primary == nil {
		return nil
	}

	return primary.GetServerInfo()
}

func (w *worker) HandleDescribe(ctx context.Context, c gat.Client, d *protocol.Describe) error {
	defer w.ret()

	errch := make(chan error)
	go func() {
		defer close(errch)
		select {
		case errch <- w.z_actually_do_describe(ctx, c, d):
		case <-ctx.Done():
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errch:
		return err
	}
}

func (w *worker) HandleExecute(ctx context.Context, c gat.Client, e *protocol.Execute) error {
	defer w.ret()

	errch := make(chan error)
	go func() {
		defer close(errch)
		select {
		case errch <- w.z_actually_do_execute(ctx, c, e):
		case <-ctx.Done():
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errch:
		return err
	}
}

func (w *worker) HandleFunction(ctx context.Context, c gat.Client, fn *protocol.FunctionCall) error {
	defer w.ret()

	errch := make(chan error)
	go func() {
		defer close(errch)
		select {
		case errch <- w.z_actually_do_fn(ctx, c, fn):
		case <-ctx.Done():
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errch:
		return err
	}
}

func (w *worker) HandleSimpleQuery(ctx context.Context, c gat.Client, query string) error {
	defer w.ret()

	start := time.Now()
	defer func() {
		w.w.pool.GetStats().AddQueryTime(int(time.Now().Sub(start).Microseconds()))
	}()

	errch := make(chan error)
	go func() {
		defer close(errch)
		select {
		case errch <- w.z_actually_do_simple_query(ctx, c, query):
		case <-ctx.Done():
		}
	}()

	// wait until query or close
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errch:
		return err
	}
}

func (w *worker) HandleTransaction(ctx context.Context, c gat.Client, query string) error {
	defer w.ret()

	start := time.Now()
	defer func() {
		w.w.pool.GetStats().AddXactTime(int(time.Now().Sub(start).Microseconds()))
	}()

	errch := make(chan error)
	go func() {
		defer close(errch)
		select {
		case errch <- w.z_actually_do_transaction(ctx, c, query):
		case <-ctx.Done():
		}
	}()

	// wait until query or close
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errch:
		return err
	}
}

func (w *worker) setCurrentBinding(client gat.Client, server gat.Connection) {
	client.SetCurrentConn(server)
	server.SetClient(client)
}

func (w *worker) unsetCurrentBinding(client gat.Client, server gat.Connection) {
	client.SetCurrentConn(nil)
	server.SetClient(nil)
}

func (w *worker) z_actually_do_describe(ctx context.Context, client gat.Client, payload *protocol.Describe) error {
	srv := w.chooseShardDescribe(client, payload)
	if srv == nil {
		return fmt.Errorf("describe('%+v') fail: no server", payload)
	}
	// describe the portal
	// we can use a replica because we are just describing what this query will return, query content doesn't matter
	// because nothing is actually executed yet
	target := srv.Choose(config.SERVERROLE_REPLICA)
	if target == nil {
		return fmt.Errorf("describe('%+v') fail: no server", payload)
	}
	w.setCurrentBinding(client, target)
	defer w.unsetCurrentBinding(client, target)
	return target.Describe(client, payload)
}
func (w *worker) z_actually_do_execute(ctx context.Context, client gat.Client, payload *protocol.Execute) error {
	srv := w.chooseShardExecute(client, payload)
	if srv == nil {
		return fmt.Errorf("describe('%+v') fail: no server", payload)
	}

	// get the query text
	portal := client.GetPortal(payload.Fields.Name)
	if portal == nil {
		return &pg_error.Error{
			Severity: pg_error.Err,
			Code:     pg_error.ProtocolViolation,
			Message:  fmt.Sprintf("portal '%s' not found", payload.Fields.Name),
		}
	}

	ps := client.GetPreparedStatement(portal.Fields.PreparedStatement)
	if ps == nil {
		return &pg_error.Error{
			Severity: pg_error.Err,
			Code:     pg_error.ProtocolViolation,
			Message:  fmt.Sprintf("prepared statement '%s' not found", ps.Fields.PreparedStatement),
		}
	}

	which, err := w.w.pool.GetRouter().InferRole(ps.Fields.Query)
	if err != nil {
		return err
	}
	target := srv.Choose(which)
	w.setCurrentBinding(client, target)
	defer w.unsetCurrentBinding(client, target)
	if target == nil {
		return fmt.Errorf("describe('%+v') fail: no server", payload)
	}
	return target.Execute(client, payload)
}
func (w *worker) z_actually_do_fn(ctx context.Context, client gat.Client, payload *protocol.FunctionCall) error {
	srv := w.chooseShardFn(client, payload)
	if srv == nil {
		return fmt.Errorf("fn('%+v') fail: no server", payload)
	}
	// call the function
	target := srv.Primary()
	if target == nil {
		return fmt.Errorf("fn('%+v') fail: no target ", payload)
	}
	w.setCurrentBinding(client, target)
	defer w.unsetCurrentBinding(client, target)
	err := target.CallFunction(client, payload)
	if err != nil {
		return fmt.Errorf("fn('%+v') fail: %w ", payload, err)
	}
	return nil
}
func (w *worker) z_actually_do_simple_query(ctx context.Context, client gat.Client, payload string) error {
	// chose a server
	srv := w.chooseShardSimpleQuery(client, payload)
	if srv == nil {
		return fmt.Errorf("call to query '%s' failed", payload)
	}
	// run the query on the server
	which, err := w.w.pool.GetRouter().InferRole(payload)
	if err != nil {
		return fmt.Errorf("error parsing '%s': %w", payload, err)
	}
	// configures the server to run with a specific role
	target := srv.Choose(which)
	if target == nil {
		return fmt.Errorf("call to query '%s' failed", payload)
	}
	w.setCurrentBinding(client, target)
	defer w.unsetCurrentBinding(client, target)
	// actually do the query
	err = target.SimpleQuery(ctx, client, payload)
	if err != nil {
		return fmt.Errorf("error executing query: %w", err)
	}
	return nil
}
func (w *worker) z_actually_do_transaction(ctx context.Context, client gat.Client, payload string) error {
	// chose a server
	srv := w.chooseShardTransaction(client, payload)
	if srv == nil {
		return fmt.Errorf("call to transaction '%s' failed", payload)
	}
	// run the query on the server
	which, err := w.w.pool.GetRouter().InferRole(payload)
	if err != nil {
		return fmt.Errorf("error parsing '%s': %w", payload, err)
	}
	// configures the server to run with a specific role
	target := srv.Choose(which)
	if target == nil {
		return fmt.Errorf("call to transaction '%s' failed", payload)
	}
	w.setCurrentBinding(client, target)
	defer w.unsetCurrentBinding(client, target)
	// actually do the query
	err = target.Transaction(ctx, client, payload)
	if err != nil {
		return fmt.Errorf("error executing query: %w", err)
	}
	return nil
}
