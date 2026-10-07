//go:build integration

package integration

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Protocol recovery regressions through pggat against real PostgreSQL.
// Scenarios independently authored in Go/pgx, inspired by the ISC-licensed
// PgBouncer test suite; pinned provenance is recorded in REGRESSIONS.md.

// withCopyTable creates pr_copy(i int) on the primary and drops it on cleanup.
func withCopyTable(t *testing.T) {
	ctx := tctx(t)
	conn, err := pgx.Connect(ctx, connURL(primaryAddr))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	if _, err := conn.Exec(ctx, "CREATE TABLE pr_copy (i integer NOT NULL)"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		conn, err := pgx.Connect(ctx, connURL(primaryAddr))
		if err != nil {
			t.Error(err)
			return
		}
		defer conn.Close(ctx)
		if _, err := conn.Exec(ctx, "DROP TABLE pr_copy"); err != nil {
			t.Error(err)
		}
	})
}

// Malformed COPY data must surface the server error, leave an explicit
// transaction in the failed state until ROLLBACK, and leave the same
// connection able to COPY and read afterwards. Regular pgx CopyFrom readers
// cannot reproduce PgBouncer's late-CopyDone race; this covers only the
// observable error/rollback/recovery contract.
// Source: pgbouncer test/test_copy.py test_copy_stdin_error_after_copy_done_
// simple (L34-42) @7d38761c.
func TestCopyFromErrorInTransaction(t *testing.T) {
	t.Parallel()
	withCopyTable(t)
	ctx := tctx(t)

	conn, err := pgx.Connect(ctx, connURL(transactionAddr))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	pgc := conn.PgConn()

	if _, err := pgc.Exec(ctx, "BEGIN").ReadAll(); err != nil {
		t.Fatal(err)
	}
	// "\n" is an empty row: invalid integer input text, SQLSTATE 22P02.
	_, err = pgc.CopyFrom(ctx, strings.NewReader("\n"), "COPY pr_copy (i) FROM STDIN")
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "22P02" {
		t.Fatalf("COPY error = %v, want SQLSTATE 22P02", err)
	}
	if s := pgc.TxStatus(); s != 'E' {
		t.Fatalf("TxStatus after failed COPY = %q, want 'E'", s)
	}
	if _, err := pgc.Exec(ctx, "ROLLBACK").ReadAll(); err != nil {
		t.Fatalf("ROLLBACK in failed transaction: %v", err)
	}
	if s := pgc.TxStatus(); s != 'I' {
		t.Fatalf("TxStatus after ROLLBACK = %q, want 'I'", s)
	}
	if _, err := pgc.CopyFrom(ctx, strings.NewReader("1\n2\n"), "COPY pr_copy (i) FROM STDIN"); err != nil {
		t.Fatalf("COPY after recovery: %v", err)
	}
	var n int
	if err := conn.QueryRow(ctx, "SELECT count(*) FROM pr_copy").Scan(&n); err != nil {
		t.Fatalf("read after recovery: %v", err)
	}
	if n != 2 {
		t.Fatalf("rows after recovery = %d, want 2", n)
	}
}

// Two clients using the same prepared statement name with different SQL must
// each get their own statement's result, whether they share one backend
// (single-server pool: pggat must re-prepare under the colliding name) or
// migrate to another backend (pggat must re-prepare the client's SQL there).
// Source: pgbouncer test/test_prepared.py test_discard_or_deallocate_all
// (L53-95) @7d38761c.
func TestPreparedStatementNameCollisionAcrossBackends(t *testing.T) {
	ctx := tctx(t)

	value := func(t *testing.T, conn *pgx.Conn, want int, stage string) (pid int) {
		t.Helper()
		var got int
		if err := conn.QueryRow(ctx, "mystmt").Scan(&got, &pid); err != nil {
			t.Fatalf("%s: %v", stage, err)
		}
		if got != want {
			t.Fatalf("%s: mystmt returned %d, want %d", stage, got, want)
		}
		return pid
	}

	// Singleton pool: every statement runs on the one backend pid.
	t.Run("same backend", func(t *testing.T) {
		a, err := pgx.Connect(ctx, connURL(singleServerAddr))
		if err != nil {
			t.Fatal(err)
		}
		defer a.Close(ctx)
		b, err := pgx.Connect(ctx, connURL(singleServerAddr))
		if err != nil {
			t.Fatal(err)
		}
		defer b.Close(ctx)

		if _, err := a.Prepare(ctx, "mystmt", "SELECT 1::int AS v, pg_backend_pid() AS pid"); err != nil {
			t.Fatal(err)
		}
		pidA := value(t, a, 1, "a first")

		if _, err := b.Prepare(ctx, "mystmt", "SELECT 2::int AS v, pg_backend_pid() AS pid"); err != nil {
			t.Fatal(err)
		}
		if pidB := value(t, b, 2, "b reuses backend"); pidB != pidA {
			t.Fatalf("b ran on backend %d, want the single backend %d", pidB, pidA)
		}

		// The backend now holds b's SQL under the shared name; a's next run
		// must re-prepare a's SQL and still return a's value.
		if pid := value(t, a, 1, "a after b"); pid != pidA {
			t.Fatalf("a ran on backend %d, want the single backend %d", pid, pidA)
		}
		value(t, b, 2, "b last")
	})

	// Shared pool: holding a's backend forces its next statement elsewhere.
	t.Run("migrated backend", func(t *testing.T) {
		a, err := pgx.Connect(ctx, connURL(transactionAddr))
		if err != nil {
			t.Fatal(err)
		}
		defer a.Close(ctx)
		b, err := pgx.Connect(ctx, connURL(transactionAddr))
		if err != nil {
			t.Fatal(err)
		}
		defer b.Close(ctx)

		if _, err := a.Prepare(ctx, "mystmt", "SELECT 1::int AS v, pg_backend_pid() AS pid"); err != nil {
			t.Fatal(err)
		}
		pidBefore := value(t, a, 1, "a before swap")

		// Hold that backend so a's next statement must run on another one.
		held, release := occupy(ctx, t, transactionAddr, pidBefore)
		defer func() {
			_ = held.Rollback(ctx)
			release()
		}()

		pidAfter := value(t, a, 1, "a after swap")
		if pidAfter == pidBefore {
			t.Fatalf("a stayed on backend %d despite the backend being occupied", pidBefore)
		}

		if _, err := b.Prepare(ctx, "mystmt", "SELECT 2::int AS v, pg_backend_pid() AS pid"); err != nil {
			t.Fatal(err)
		}
		value(t, b, 2, "b")
		value(t, a, 1, "a last")
	})
}

// DEALLOCATE ALL must invalidate the client's protocol-level named statements
// exactly like DISCARD ALL: after it, executing the name on a fresh backend
// must fail with SQLSTATE 26000, not be silently re-prepared by pggat.
// Sequential (no t.Parallel): occupy needs the pool uncontended, and parallel
// tests are paused while sequential tests run.
// Source: pgbouncer test/test_prepared.py test_discard_or_deallocate_all
// (L52-95) @7d38761c.
func TestDeallocateAllInvalidatesStatementAcrossMigration(t *testing.T) {
	ctx := tctx(t)

	conn, err := pgx.Connect(ctx, connURL(transactionAddr))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	pgc := conn.PgConn()

	// Raw protocol prepare/execute, bypassing pgx's statement cache.
	if _, err := pgc.Prepare(ctx, "dv", "SELECT 41::int", nil); err != nil {
		t.Fatal(err)
	}
	rr := pgc.ExecPrepared(ctx, "dv", nil, nil, nil)
	var row string
	for rr.NextRow() {
		row = string(rr.Values()[0])
	}
	if _, err := rr.Close(); err != nil {
		t.Fatal(err)
	}
	if row != "41" {
		t.Fatalf("dv returned %q, want 41", row)
	}

	// One simple-Query packet: transaction pooling dispatches it whole, so the
	// PID came from the same backend that ran the DEALLOCATE.
	results, err := pgc.Exec(ctx, "DEALLOCATE ALL; SELECT pg_backend_pid()").ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 || len(results[1].Rows) != 1 {
		t.Fatalf("want one pid row after DEALLOCATE ALL, got %d results", len(results))
	}
	if tag := results[0].CommandTag.String(); tag != "DEALLOCATE ALL" {
		t.Fatalf("CommandComplete tag = %q, want DEALLOCATE ALL", tag)
	}
	pid, err := strconv.Atoi(string(results[1].Rows[0][0]))
	if err != nil {
		t.Fatal(err)
	}

	// Hold the backend that ran the DEALLOCATE so the next Bind must run on a
	// fresh backend, where a stale statement cache would silently re-prepare
	// dv instead of failing.
	held, release := occupy(ctx, t, transactionAddr, pid)
	defer func() {
		_ = held.Rollback(ctx)
		release()
	}()

	_, err = pgc.ExecPrepared(ctx, "dv", nil, nil, nil).Close()
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "26000" {
		t.Fatalf("exec dv after DEALLOCATE ALL on fresh backend: err = %v, want SQLSTATE 26000", err)
	}
}

// A Parse that fails on the server (SQLSTATE 42P01) must leave the connection
// usable, and once the missing table exists the same statement name prepares
// and runs.
// Source: pgbouncer test/test_prepared.py test_prepared_failed_prepare
// (L487-493) @7d38761c.
func TestFailedPrepareRecoversOnSameConnection(t *testing.T) {
	t.Parallel()
	ctx := tctx(t)

	conn, err := pgx.Connect(ctx, connURL(transactionAddr))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)

	_, err = conn.Prepare(ctx, "prstmt", "SELECT * FROM pr_missing")
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "42P01" {
		t.Fatalf("prepare error = %v, want SQLSTATE 42P01", err)
	}
	if _, err := conn.Exec(ctx, "CREATE TABLE pr_missing (a int)"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		c, err := pgx.Connect(ctx, connURL(primaryAddr))
		if err != nil {
			t.Error(err)
			return
		}
		defer c.Close(ctx)
		if _, err := c.Exec(ctx, "DROP TABLE pr_missing"); err != nil {
			t.Error(err)
		}
	})
	if _, err := conn.Prepare(ctx, "prstmt", "SELECT count(*) FROM pr_missing"); err != nil {
		t.Fatalf("prepare after recovery: %v", err)
	}
	var n int
	if err := conn.QueryRow(ctx, "prstmt").Scan(&n); err != nil {
		t.Fatalf("execute after recovery: %v", err)
	}
	if n != 0 {
		t.Fatalf("prstmt returned %d, want 0", n)
	}
}
