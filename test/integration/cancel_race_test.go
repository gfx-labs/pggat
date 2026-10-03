//go:build integration

package integration

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const cancelRequestCode = 80877102

// cancelProxy forwards PostgreSQL connections to a backend address. It holds each
// CancelRequest until the test delivers it, so a cancel can be kept in flight.
type cancelProxy struct {
	listener net.Listener
	upstream string
	held     chan *heldCancel

	mu     sync.Mutex
	conns  map[net.Conn]struct{}
	closed bool
	wg     sync.WaitGroup
}

type heldCancel struct {
	proxy   *cancelProxy
	conn    *net.TCPConn
	request []byte
}

func startCancelProxy(upstream string) (*cancelProxy, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	p := &cancelProxy{
		listener: l,
		upstream: upstream,
		held:     make(chan *heldCancel, 16),
		conns:    make(map[net.Conn]struct{}),
	}
	p.wg.Add(1)
	go p.accept()
	return p, nil
}

func (p *cancelProxy) Addr() string {
	return p.listener.Addr().String()
}

func (p *cancelProxy) Close() {
	_ = p.listener.Close()
	p.mu.Lock()
	p.closed = true
	for c := range p.conns {
		_ = c.Close()
	}
	p.mu.Unlock()
	p.wg.Wait()
}

// track registers c so Close interrupts it. It reports false after Close.
func (p *cancelProxy) track(c net.Conn) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		_ = c.Close()
		return false
	}
	p.conns[c] = struct{}{}
	return true
}

func (p *cancelProxy) untrack(c net.Conn) {
	p.mu.Lock()
	delete(p.conns, c)
	p.mu.Unlock()
	_ = c.Close()
}

func (p *cancelProxy) accept() {
	defer p.wg.Done()
	for {
		c, err := p.listener.Accept()
		if err != nil {
			return
		}
		if !p.track(c) {
			return
		}
		p.wg.Add(1)
		go p.serve(c.(*net.TCPConn))
	}
}

func (p *cancelProxy) serve(c *net.TCPConn) {
	defer p.wg.Done()
	var head [8]byte
	if _, err := io.ReadFull(c, head[:]); err != nil {
		p.untrack(c)
		return
	}
	if binary.BigEndian.Uint32(head[4:]) == cancelRequestCode {
		n := binary.BigEndian.Uint32(head[:4])
		if n < 8 || n > 1024 {
			p.untrack(c)
			return
		}
		request := make([]byte, n)
		copy(request, head[:])
		if _, err := io.ReadFull(c, request[8:]); err != nil {
			p.untrack(c)
			return
		}
		select {
		case p.held <- &heldCancel{proxy: p, conn: c, request: request}:
		default:
			p.untrack(c)
		}
		return
	}
	defer p.untrack(c)

	up, err := net.Dial("tcp", p.upstream)
	if err != nil || !p.track(up) {
		return
	}
	defer p.untrack(up)
	if _, err := up.Write(head[:]); err != nil {
		return
	}
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		_, _ = io.Copy(up, c)
		p.untrack(up)
	}()
	_, _ = io.Copy(c, up)
}

// Next returns the next CancelRequest pggat sent for backend pid. Other cancels, such as
// those pgx sends when a test context ends, are delivered unchanged.
func (p *cancelProxy) Next(ctx context.Context, t *testing.T, pid int) *heldCancel {
	t.Helper()
	for {
		select {
		case h := <-p.held:
			if len(h.request) >= 12 && binary.BigEndian.Uint32(h.request[8:12]) == uint32(pid) {
				return h
			}
			_ = h.deliver(ctx)
			h.proxy.untrack(h.conn)
		case <-ctx.Done():
			t.Fatalf("pggat did not forward a cancel request: %v", ctx.Err())
			return nil
		}
	}
}

// deliver sends the request to PostgreSQL and waits for its close, which follows the signal to the backend.
func (h *heldCancel) deliver(ctx context.Context) error {
	var d net.Dialer
	up, err := d.DialContext(ctx, "tcp", h.proxy.upstream)
	if err != nil {
		return err
	}
	defer up.Close()
	if deadline, ok := ctx.Deadline(); ok {
		if err := up.SetDeadline(deadline); err != nil {
			return err
		}
	}
	if _, err := up.Write(h.request); err != nil {
		return err
	}
	_, err = io.Copy(io.Discard, up)
	return err
}

// Forward delivers the cancel and reports success to pggat by closing cleanly.
func (h *heldCancel) Forward(ctx context.Context, t *testing.T) {
	t.Helper()
	if err := h.deliver(ctx); err != nil {
		t.Fatalf("deliver cancel: %v", err)
	}
	h.proxy.untrack(h.conn)
}

// Reset delivers the cancel but resets pggat's connection, so pggat cannot tell it was delivered.
func (h *heldCancel) Reset(ctx context.Context, t *testing.T) {
	t.Helper()
	if err := h.deliver(ctx); err != nil {
		t.Fatalf("deliver cancel: %v", err)
	}
	_ = h.conn.SetLinger(0)
	h.proxy.untrack(h.conn)
}

// background runs f on its own goroutine. Cleanup cancels its context and waits for it, so the
// connection it uses is not closed while still in use.
func background[T any](ctx context.Context, t *testing.T, f func(ctx context.Context) T) <-chan T {
	t.Helper()
	ctx, cancel := context.WithCancel(ctx)
	done := make(chan T, 1)
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		done <- f(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		<-finished
	})
	return done
}

// cancelRace runs a blocked query on the only backend and has pggat forward a cancel for it.
// The proxy holds that cancel while the query finishes normally. It returns the backend pid,
// the held cancel, the result of the client's CancelRequest, and a second client.
func cancelRace(ctx context.Context, t *testing.T, addr string, lockKey int64) (int, *heldCancel, <-chan error, *pgx.Conn, *pgx.Conn) {
	t.Helper()
	// Drop cancels left by an earlier test.
	for drained := false; !drained; {
		select {
		case h := <-cancels.held:
			cancels.untrack(h.conn)
		default:
			drained = true
		}
	}
	// Startup pairs with the backend, so connect both clients before it is held.
	canceled := lifecycleConnect(ctx, t, addr)
	next := lifecycleConnect(ctx, t, addr)
	observer := lifecycleConnect(ctx, t, primaryAddr)

	var pid int
	if err := canceled.QueryRow(ctx, "SELECT pg_backend_pid()").Scan(&pid); err != nil {
		t.Fatal(err)
	}
	if _, err := observer.Exec(ctx, "SELECT pg_advisory_lock($1)", lockKey); err != nil {
		t.Fatal(err)
	}
	done := background(ctx, t, func(ctx context.Context) error {
		_, err := canceled.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", lockKey)
		return err
	})
	waitForAdvisoryLockWait(ctx, t, observer, pid)

	cancelSent := background(ctx, t, func(ctx context.Context) error {
		return canceled.PgConn().CancelRequest(ctx)
	})
	held := cancels.Next(ctx, t, pid)
	// Runs before the background cleanups, so a failed test does not wait for pggat's cancel timeout.
	t.Cleanup(func() { held.proxy.untrack(held.conn) })

	// The query finishes before its cancel reaches PostgreSQL.
	if _, err := observer.Exec(ctx, "SELECT pg_advisory_unlock($1)", lockKey); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("query finished before its cancel was delivered: %v", err)
		}
	case <-ctx.Done():
		t.Fatalf("query did not finish while its cancel was held: %v", ctx.Err())
	}
	return pid, held, cancelSent, next, observer
}

func cancelSentResult(ctx context.Context, t *testing.T, cancelSent <-chan error) {
	t.Helper()
	select {
	case err := <-cancelSent:
		if err != nil {
			t.Fatalf("cancel request: %v", err)
		}
	case <-ctx.Done():
		t.Fatalf("cancel request did not finish: %v", ctx.Err())
	}
}

var cancelRaceRoutes = []struct {
	name string
	addr *string
}{
	{"transaction", &cancelSingleAddr},
	{"hybrid", &cancelHybridAddr},
}

// TestInFlightCancelDoesNotReachNextClient checks that a cancel pggat is still forwarding
// for one client does not interrupt the next client's query on the same backend.
// Contract after PgBouncer test/test_cancel.py (test_cancel_race) at 7d38761c.
func TestInFlightCancelDoesNotReachNextClient(t *testing.T) {
	for _, tc := range cancelRaceRoutes {
		t.Run(tc.name, func(t *testing.T) {
			ctx := tctx(t)
			const lockKey = 0x63616e63
			pid, held, cancelSent, next, observer := cancelRace(ctx, t, *tc.addr, lockKey)

			if _, err := observer.Exec(ctx, "SELECT pg_advisory_lock($1)", lockKey); err != nil {
				t.Fatal(err)
			}
			unlocked := false
			defer func() {
				if !unlocked {
					_, _ = observer.Exec(ctx, "SELECT pg_advisory_unlock($1)", lockKey)
				}
			}()
			type result struct {
				pid int
				err error
			}
			done := background(ctx, t, func(ctx context.Context) result {
				var r result
				r.err = next.QueryRow(ctx, "SELECT pg_backend_pid() FROM (SELECT pg_advisory_xact_lock($1)) s", lockKey).Scan(&r.pid)
				return r
			})

			// Without pinning the next query reaches the backend now. With pinning this only
			// bounds the wait, since the query cannot start until the cancel is delivered.
			_ = advisoryLockWaitWithin(ctx, t, observer, pid, time.Second)
			held.Forward(ctx, t)
			cancelSentResult(ctx, t, cancelSent)

			for !advisoryLockWaitWithin(ctx, t, observer, pid, 10*time.Millisecond) {
				select {
				case r := <-done:
					t.Fatalf("next query ended before its lock was released: pid %d, %v", r.pid, r.err)
				case <-ctx.Done():
					t.Fatalf("next query never waited on the backend: %v", ctx.Err())
				default:
				}
			}
			if _, err := observer.Exec(ctx, "SELECT pg_advisory_unlock($1)", lockKey); err != nil {
				t.Fatal(err)
			}
			unlocked = true
			var r result
			select {
			case r = <-done:
			case <-ctx.Done():
				t.Fatalf("next query did not finish: %v", ctx.Err())
			}
			if r.err != nil {
				t.Fatalf("next query: %v", r.err)
			}
			if r.pid != pid {
				t.Fatalf("backend %d replaced by %d after a confirmed cancel", pid, r.pid)
			}
		})
	}
}

// TestUnconfirmedCancelReplacesBackend checks that a backend is not reused when pggat cannot
// confirm whether its forwarded cancel was delivered, since it could arrive during the next query.
func TestUnconfirmedCancelReplacesBackend(t *testing.T) {
	for _, tc := range cancelRaceRoutes {
		t.Run(tc.name, func(t *testing.T) {
			ctx := tctx(t)
			pid, held, cancelSent, next, _ := cancelRace(ctx, t, *tc.addr, 0x756e6366)
			held.Reset(ctx, t)
			cancelSentResult(ctx, t, cancelSent)

			var after int
			if err := next.QueryRow(ctx, "SELECT pg_backend_pid()").Scan(&after); err != nil {
				t.Fatalf("next query: %v", err)
			}
			if after == pid {
				t.Fatalf("backend %d reused after an unconfirmed cancel", pid)
			}
		})
	}
}

// TestHybridCancelRequest checks that a cancel reaches a query running on the hybrid primary
// and that the client keeps its backend afterwards.
func TestHybridCancelRequest(t *testing.T) {
	ctx := tctx(t)
	const lockKey = 0x68796272
	client := lifecycleConnect(ctx, t, hybridSingleAddr)
	observer := lifecycleConnect(ctx, t, primaryAddr)

	var pid int
	if err := client.QueryRow(ctx, "SELECT pg_backend_pid()").Scan(&pid); err != nil {
		t.Fatal(err)
	}
	if _, err := observer.Exec(ctx, "SELECT pg_advisory_lock($1)", lockKey); err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = observer.Exec(ctx, "SELECT pg_advisory_unlock($1)", lockKey) }()

	done := background(ctx, t, func(ctx context.Context) error {
		_, err := client.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", lockKey)
		return err
	})
	waitForAdvisoryLockWait(ctx, t, observer, pid)
	if err := client.PgConn().CancelRequest(ctx); err != nil {
		t.Fatalf("cancel request: %v", err)
	}
	var pgErr *pgconn.PgError
	select {
	case err := <-done:
		if !errors.As(err, &pgErr) || pgErr.Code != "57014" {
			t.Fatalf("canceled query: got %v, want SQLSTATE 57014", err)
		}
	case <-ctx.Done():
		t.Fatalf("cancel did not reach the hybrid primary: %v", ctx.Err())
	}

	var after int
	if err := client.QueryRow(ctx, "SELECT pg_backend_pid()").Scan(&after); err != nil {
		t.Fatalf("query after cancel: %v", err)
	}
	if after != pid {
		t.Fatalf("backend %d replaced by %d after a canceled query", pid, after)
	}
}

// advisoryLockWaitWithin reports whether backend pid waits on an advisory lock within d.
func advisoryLockWaitWithin(ctx context.Context, t *testing.T, observer *pgx.Conn, pid int, d time.Duration) bool {
	t.Helper()
	deadline := time.Now().Add(d)
	for {
		var waiting bool
		err := observer.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM pg_stat_activity WHERE pid = $1 AND wait_event_type = 'Lock' AND wait_event = 'advisory')`,
			pid,
		).Scan(&waiting)
		if err != nil {
			t.Fatalf("observe backend %d: %v", pid, err)
		}
		if waiting {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		select {
		case <-ctx.Done():
			t.Fatalf("observe backend %d: %v", pid, ctx.Err())
		case <-time.After(10 * time.Millisecond):
		}
	}
}
