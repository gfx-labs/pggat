package transaction

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"gfx.cafe/gfx/pggat/lib/config"
	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/gat/pool/transaction/shard"
	"gfx.cafe/gfx/pggat/lib/gat/protocol"
	"gfx.cafe/gfx/pggat/lib/gat/protocol/pg_error"
	"gfx.cafe/gfx/pggat/lib/metrics"
)

// a single use worker with an embedded connection database.
// it wraps a pointer to the connection database.
type worker struct {
	// the parent connectino database
	w   *Pool
	rev int

	shards []*shard.Shard

	mu sync.Mutex
}

// ret urn worker to database
func (w *worker) ret() {
	w.w.workerPool <- w
}

// attempt to connect to a new shard with this worker
func (w *worker) fetchShard(client gat.Client, n int) bool {
	conf := w.w.c.Load()
	if n < 0 || n >= len(conf.Shards) {
		return false
	}

	for len(w.shards) <= n {
		w.shards = append(w.shards, nil)
	}

	w.shards[n] = shard.FromConfig(w.w.dialer, client.GetOptions(), w.w.c.Load(), w.w.user, conf.Shards[n])
	return true
}

func (w *worker) invalidateShard(n int) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.shards[n] = nil
}

func (w *worker) chooseShard(client gat.Client) *shard.Shard {
	w.mu.Lock()
	defer w.mu.Unlock()

	conf := w.w.c.Load()

	preferred := rand.Intn(len(conf.Shards))
	if client != nil {
		if p, ok := client.GetRequestedShard(); ok {
			preferred = p % len(conf.Shards)
		}

		key := client.GetShardingKey()
		if key != "" {
			// do sharding function on key TODO
		}
	}

	if preferred < len(w.shards) && w.shards[preferred] != nil {
		w.shards[preferred].EnsureConfig(conf.Shards[preferred])
		return w.shards[preferred]
	}

	// we need to fetch a shard
	if w.fetchShard(client, preferred) {
		return w.shards[preferred]
	}

	return nil
}

func (w *worker) GetServerInfo(client gat.Client) []*protocol.ParameterStatus {
	defer w.ret()

	s := w.chooseShard(client)
	if s == nil {
		return nil
	}

	primary := s.GetPrimary()
	if primary == nil {
		return nil
	}

	return primary.GetServerInfo()
}

func (w *worker) HandleDescribe(ctx context.Context, c gat.Client, d *protocol.Describe) error {
	defer w.ret()

	if w.w.user.StatementTimeout != 0 {
		var done context.CancelFunc
		ctx, done = context.WithTimeout(ctx, time.Duration(w.w.user.StatementTimeout)*time.Millisecond)
		defer done()
	}

	start := time.Now()
	defer func() {
		metrics.RecordTransactionTime(w.w.GetDatabase().GetName(), w.w.user.Name, time.Since(start))
	}()

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

	if w.w.user.StatementTimeout != 0 {
		var done context.CancelFunc
		ctx, done = context.WithTimeout(ctx, time.Duration(w.w.user.StatementTimeout)*time.Millisecond)
		defer done()
	}

	start := time.Now()
	defer func() {
		metrics.RecordTransactionTime(w.w.GetDatabase().GetName(), w.w.user.Name, time.Since(start))
	}()

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

	if w.w.user.StatementTimeout != 0 {
		var done context.CancelFunc
		ctx, done = context.WithTimeout(ctx, time.Duration(w.w.user.StatementTimeout)*time.Millisecond)
		defer done()
	}

	start := time.Now()
	defer func() {
		metrics.RecordQueryTime(w.w.GetDatabase().GetName(), w.w.user.Name, time.Since(start))
	}()

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

	if w.w.user.StatementTimeout != 0 {
		var done context.CancelFunc
		ctx, done = context.WithTimeout(ctx, time.Duration(w.w.user.StatementTimeout)*time.Millisecond)
		defer done()
	}

	start := time.Now()
	defer func() {
		metrics.RecordQueryTime(w.w.GetDatabase().GetName(), w.w.user.Name, time.Since(start))
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

	if w.w.user.StatementTimeout != 0 {
		var done context.CancelFunc
		ctx, done = context.WithTimeout(ctx, time.Duration(w.w.user.StatementTimeout)*time.Millisecond)
		defer done()
	}

	start := time.Now()
	defer func() {
		metrics.RecordTransactionTime(w.w.GetDatabase().GetName(), w.w.user.Name, time.Since(start))
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
	srv := w.chooseShard(client)
	if srv == nil {
		return fmt.Errorf("describe('%+v') fail: no server", payload)
	}
	// describe the portal
	// we can use a replica because we are just describing what this query will return, query content doesn't matter
	// because nothing is actually executed yet
	if !w.w.user.Role.CanUse(config.SERVERROLE_REPLICA) {
		return errors.New("permission denied")
	}
	target := srv.Choose(config.SERVERROLE_REPLICA)
	if target == nil {
		return fmt.Errorf("describe('%+v') fail: no server", payload)
	}
	w.setCurrentBinding(client, target)
	defer w.unsetCurrentBinding(client, target)
	return target.Describe(ctx, client, payload)
}
func (w *worker) z_actually_do_execute(ctx context.Context, client gat.Client, payload *protocol.Execute) error {
	srv := w.chooseShard(client)
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

	which, err := w.w.database.GetRouter().InferRole(ps.Fields.Query)
	if err != nil {
		return err
	}
	if !w.w.user.Role.CanUse(which) {
		return errors.New("permission denied")
	}
	target := srv.Choose(which)
	w.setCurrentBinding(client, target)
	defer w.unsetCurrentBinding(client, target)
	if target == nil {
		return fmt.Errorf("describe('%+v') fail: no server", payload)
	}
	return target.Execute(ctx, client, payload)
}
func (w *worker) z_actually_do_fn(ctx context.Context, client gat.Client, payload *protocol.FunctionCall) error {
	srv := w.chooseShard(client)
	if srv == nil {
		return fmt.Errorf("fn('%+v') fail: no server", payload)
	}
	// call the function
	if !w.w.user.Role.CanUse(config.SERVERROLE_PRIMARY) {
		return errors.New("permission denied")
	}
	target := srv.GetPrimary()
	if target == nil {
		return fmt.Errorf("fn('%+v') fail: no target ", payload)
	}
	w.setCurrentBinding(client, target)
	defer w.unsetCurrentBinding(client, target)
	err := target.CallFunction(ctx, client, payload)
	if err != nil {
		return fmt.Errorf("fn('%+v') fail: %w ", payload, err)
	}
	return nil
}
func (w *worker) z_actually_do_simple_query(ctx context.Context, client gat.Client, payload string) error {
	// chose a server
	srv := w.chooseShard(client)
	if srv == nil {
		return fmt.Errorf("call to query '%s' failed", payload)
	}
	// run the query on the server
	which, err := w.w.database.GetRouter().InferRole(payload)
	if err != nil {
		return fmt.Errorf("error parsing '%s': %w", payload, err)
	}
	if !w.w.user.Role.CanUse(which) {
		return errors.New("permission denied")
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
	srv := w.chooseShard(client)
	if srv == nil {
		return fmt.Errorf("call to transaction '%s' failed", payload)
	}
	// run the query on the server
	which, err := w.w.database.GetRouter().InferRole(payload)
	if err != nil {
		return fmt.Errorf("error parsing '%s': %w", payload, err)
	}
	if !w.w.user.Role.CanUse(which) {
		return errors.New("permission denied")
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
