package conn_pool

import (
	"context"
	"fmt"
	"gfx.cafe/gfx/pggat/lib/config"
	"log"

	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/gat/protocol"
)

type _wp ConnectionPool

// a single use worker with an embedded connection pool.
// it wraps a pointer to the connection pool.
type worker struct {
	// the parent connectino pool
	w *ConnectionPool
}

// ret urn worker to pool
func (w *worker) ret() {
	w.w.workerPool <- w
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
	log.Println("worker selected for fn")
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

func (w *worker) z_actually_do_describe(ctx context.Context, client gat.Client, payload *protocol.Describe) error {
	c := w.w
	srv := c.chooseConnections()
	if srv == nil {
		return fmt.Errorf("describe('%+v') fail: no server", payload)
	}
	defer srv.mu.Unlock()
	// describe the portal
	// we can use a replica because we are just describing what this query will return, query content doesn't matter
	// because nothing is actually executed yet
	target := srv.choose(config.SERVERROLE_REPLICA)
	if target == nil {
		return fmt.Errorf("describe('%+v') fail: no server", payload)
	}
	return target.Describe(client, payload)
}
func (w *worker) z_actually_do_execute(ctx context.Context, client gat.Client, payload *protocol.Execute) error {
	c := w.w
	srv := c.chooseConnections()
	if srv == nil {
		return fmt.Errorf("describe('%+v') fail: no server", payload)
	}
	defer srv.mu.Unlock()
	// execute the query
	// for now, use primary
	// TODO read the query of the underlying prepared statement and choose server accordingly
	target := srv.primary
	if target == nil {
		return fmt.Errorf("describe('%+v') fail: no server", payload)
	}
	return target.Execute(client, payload)
}
func (w *worker) z_actually_do_fn(ctx context.Context, client gat.Client, payload *protocol.FunctionCall) error {
	c := w.w
	srv := c.chooseConnections()
	if srv == nil {
		return fmt.Errorf("fn('%+v') fail: no server", payload)
	}
	defer srv.mu.Unlock()
	// call the function
	target := srv.primary
	if target == nil {
		return fmt.Errorf("fn('%+v') fail: no target ", payload)
	}
	err := srv.primary.CallFunction(client, payload)
	if err != nil {
		return fmt.Errorf("fn('%+v') fail: %w ", payload, err)
	}
	return nil
}
func (w *worker) z_actually_do_simple_query(ctx context.Context, client gat.Client, payload string) error {
	c := w.w
	// chose a server
	srv := c.chooseConnections()
	if srv == nil {
		return fmt.Errorf("call to query '%s' failed", payload)
	}
	// note that the server comes locked. you MUST unlock it
	defer srv.mu.Unlock()
	// run the query on the server
	which, err := c.pool.GetRouter().InferRole(payload)
	if err != nil {
		return fmt.Errorf("error parsing '%s': %w", payload, err)
	}
	// configures the server to run with a specific role
	target := srv.choose(which)
	if target == nil {
		return fmt.Errorf("call to query '%s' failed", payload)
	}
	// actually do the query
	err = target.SimpleQuery(ctx, client, payload)
	if err != nil {
		return fmt.Errorf("error executing query: %w", err)
	}
	return nil
}
func (w *worker) z_actually_do_transaction(ctx context.Context, client gat.Client, payload string) error {
	c := w.w
	// chose a server
	srv := c.chooseConnections()
	if srv == nil {
		return fmt.Errorf("call to transaction '%s' failed", payload)
	}
	// note that the server comes locked. you MUST unlock it
	defer srv.mu.Unlock()
	// run the query on the server
	which, err := c.pool.GetRouter().InferRole(payload)
	if err != nil {
		return fmt.Errorf("error parsing '%s': %w", payload, err)
	}
	// configures the server to run with a specific role
	target := srv.choose(which)
	if target == nil {
		return fmt.Errorf("call to transaction '%s' failed", payload)
	}
	// actually do the query
	err = target.Transaction(ctx, client, payload)
	if err != nil {
		return fmt.Errorf("error executing query: %w", err)
	}
	return nil
}
