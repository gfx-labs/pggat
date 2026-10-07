//go:build integration

package integration

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Bound cleanup even if the connection is busy.
func lifecycleConnect(ctx context.Context, t *testing.T, addr string, query ...string) *pgx.Conn {
	t.Helper()
	conn, err := pgx.Connect(ctx, connURL(addr, query...))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = conn.Close(ctx)
	})
	return conn
}

// waitForAdvisoryLockWait polls PostgreSQL until backend pid is blocked on an advisory lock.
func waitForAdvisoryLockWait(ctx context.Context, t *testing.T, observer *pgx.Conn, pid int) {
	t.Helper()
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
			return
		}
		select {
		case <-ctx.Done():
			t.Fatalf("backend %d never waited on the advisory lock: %v", pid, ctx.Err())
		case <-time.After(10 * time.Millisecond):
		}
	}
}

// TestCancelRequestTargetsOnlyActiveClient checks that a CancelRequest sent to pggat
// interrupts the sender's running query and nothing else. Both clients share the one
// backend of singleServerAddr, so a cancel routed by backend instead of by client would hit
// the other client's query.
// Contract after PgBouncer test/test_cancel.py (test_cancel_race) at 7d38761c.
func TestCancelRequestTargetsOnlyActiveClient(t *testing.T) {
	ctx := tctx(t)
	const lockKey = 0x6c696665

	// Startup pairs with the backend, so connect both clients before it is held.
	idle := lifecycleConnect(ctx, t, singleServerAddr)
	active := lifecycleConnect(ctx, t, singleServerAddr)
	observer := lifecycleConnect(ctx, t, primaryAddr)

	var pid int
	if err := idle.QueryRow(ctx, "SELECT pg_backend_pid()").Scan(&pid); err != nil {
		t.Fatal(err)
	}

	if _, err := observer.Exec(ctx, "SELECT pg_advisory_lock($1)", lockKey); err != nil {
		t.Fatal(err)
	}
	locked := true
	unlock := func() {
		if !locked {
			return
		}
		locked = false
		if _, err := observer.Exec(ctx, "SELECT pg_advisory_unlock($1)", lockKey); err != nil {
			t.Fatal(err)
		}
	}
	defer unlock()

	block := func() <-chan error {
		done := make(chan error, 1)
		go func() {
			_, err := active.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", lockKey)
			done <- err
		}()
		waitForAdvisoryLockWait(ctx, t, observer, pid)
		return done
	}
	result := func(done <-chan error) error {
		select {
		case err := <-done:
			return err
		case <-ctx.Done():
			t.Fatalf("blocked query did not return: %v", ctx.Err())
			return nil
		}
	}

	// The idle client last ran on this backend but owns nothing on it now.
	done := block()
	if err := idle.PgConn().CancelRequest(ctx); err != nil {
		t.Fatalf("idle cancel request: %v", err)
	}
	// This checks idle-client routing, not cancellations already in flight to PostgreSQL.
	unlock()
	if err := result(done); err != nil {
		t.Fatalf("idle client's cancel interrupted another client's query: %v", err)
	}

	if _, err := observer.Exec(ctx, "SELECT pg_advisory_lock($1)", lockKey); err != nil {
		t.Fatal(err)
	}
	locked = true

	done = block()
	if err := active.PgConn().CancelRequest(ctx); err != nil {
		t.Fatalf("active cancel request: %v", err)
	}
	var pgErr *pgconn.PgError
	if err := result(done); !errors.As(err, &pgErr) || pgErr.Code != "57014" {
		t.Fatalf("canceled query: got %v, want SQLSTATE 57014", err)
	}

	// The canceled client keeps working, on the same backend.
	var after int
	if err := active.QueryRow(ctx, "SELECT pg_backend_pid()").Scan(&after); err != nil {
		t.Fatalf("query after cancel: %v", err)
	}
	if after != pid {
		t.Fatalf("backend %d replaced by %d after a canceled query", pid, after)
	}
}

// TestDisconnectInTransactionRollsBack checks that a client leaving inside a write
// transaction has its work rolled back and its locks released, and that the backend
// returns to the pool instead of being dropped.
// Contract after PgCat tests/ruby/misc_spec.rb (client disconnect in transaction) at 5b038813.
func TestDisconnectInTransactionRollsBack(t *testing.T) {
	const lockKey = 0x64697363

	for _, tc := range []struct {
		name  string
		leave func(ctx context.Context, conn *pgx.Conn) error
	}{
		// Terminate message, as sent by a client closing normally.
		{"terminate", func(ctx context.Context, conn *pgx.Conn) error { return conn.Close(ctx) }},
		// The socket closes without a Terminate message.
		{"abrupt", func(_ context.Context, conn *pgx.Conn) error { return conn.PgConn().Conn().Close() }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := tctx(t)
			title := fmt.Sprintf("lifecycle-disconnect-%s-%d", tc.name, time.Now().UnixNano())

			leaving := lifecycleConnect(ctx, t, singleServerAddr)
			tx, err := leaving.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			var pid int
			if err := tx.QueryRow(ctx, "SELECT pg_backend_pid()").Scan(&pid); err != nil {
				t.Fatal(err)
			}
			if _, err := tx.Exec(ctx, "INSERT INTO posts (user_id, title) VALUES (1, $1)", title); err != nil {
				t.Fatal(err)
			}
			if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", lockKey); err != nil {
				t.Fatal(err)
			}

			if err := tc.leave(ctx, leaving); err != nil {
				t.Fatalf("disconnect: %v", err)
			}

			// The pool has one backend, so this waits until the abandoned transaction is gone.
			next := lifecycleConnect(ctx, t, singleServerAddr)
			var (
				after    int
				lockFree bool
				rows     int
			)
			err = next.QueryRow(ctx,
				"SELECT pg_backend_pid(), pg_try_advisory_xact_lock($1), (SELECT count(*) FROM posts WHERE title = $2)",
				lockKey, title,
			).Scan(&after, &lockFree, &rows)
			if err != nil {
				t.Fatal(err)
			}
			if rows != 0 {
				t.Errorf("write from the disconnected transaction is visible")
			}
			if !lockFree {
				t.Errorf("transaction lock held after disconnect")
			}
			if after != pid {
				t.Errorf("backend %d replaced by %d after a client disconnect", pid, after)
			}
		})
	}
}
