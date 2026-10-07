//go:build integration

package integration

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgproto3"
)

// A client's prepared statements must survive a move to another backend
// (transaction pooling) the same way they survive on one PostgreSQL session.
// When one of them no longer parses there, only requests that use it fail,
// and they get PostgreSQL's own error rather than 26000. Each case runs on
// PostgreSQL directly as a control and through pggat after a confirmed
// backend change.

// prTable creates table name on the primary and drops it (if present) on cleanup.
func prTable(t *testing.T, name string) (create, drop func()) {
	t.Helper()
	exec := func(sql string) {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		conn, err := pgx.Connect(ctx, connURL(primaryAddr))
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close(ctx)
		if _, err := conn.Exec(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() { exec("DROP TABLE IF EXISTS " + name) })
	return func() { exec("CREATE TABLE " + name + " (a int)") }, func() { exec("DROP TABLE " + name) }
}

// prPid returns the backend serving w's next transaction.
func prPid(t *testing.T, w *wire) int {
	t.Helper()
	w.send(query("SELECT pg_backend_pid()"))
	var pid int
	for {
		msg, err := w.next()
		if err != nil {
			t.Fatal(err)
		}
		switch m := msg.(type) {
		case *pgproto3.DataRow:
			pid, err = strconv.Atoi(string(m.Values[0]))
			if err != nil {
				t.Fatal(err)
			}
		case *pgproto3.ReadyForQuery:
			return pid
		case *pgproto3.ErrorResponse:
			t.Fatalf("pid query: %s", m.Code)
		}
	}
}

// prRun names each run's statements. Backends outlive clients, so names must not repeat.
var prRun int

func TestPreparedStatementRecoveryAfterBackendChange(t *testing.T) {
	const unrelated = 20
	ok := func(i int) string { return fmt.Sprintf("pr%d_ok%d", prRun, i) }
	bad := func() string { return fmt.Sprintf("pr%d_bad", prRun) }

	type target struct {
		name string
		// move forces w's next transaction onto another backend until the test ends.
		move func(t *testing.T, w *wire)
	}
	targets := []target{
		{"direct", func(*testing.T, *wire) {}},
		{"pggat moved", func(t *testing.T, w *wire) {
			ctx := tctx(t)
			before := prPid(t, w)
			held, release := occupy(ctx, t, transactionAddr, before)
			t.Cleanup(func() {
				_ = held.Rollback(ctx)
				release()
			})
			if after := prPid(t, w); after == before {
				t.Fatalf("client stayed on backend %d", before)
			}
		}},
	}
	addr := map[string]string{"direct": primaryAddr, "pggat moved": transactionAddr}

	cases := []struct {
		name string
		run  func(t *testing.T, w *wire, recreate func())
	}{
		// named and unnamed Bind of the stale statement, and statement Describe
		{"stale statement reports its own error", func(t *testing.T, w *wire, recreate func()) {
			w.send(bind("", bad()), execute("", 0), syncMsg)
			w.expect("bind unnamed portal", "Error:42P01", "Ready:I")
			w.send(bind("pr_portal", bad()), execute("pr_portal", 0), syncMsg)
			w.expect("bind named portal", "Error:42P01", "Ready:I")
			// PostgreSQL still holds the statement, so it describes the parameters before failing.
			// A new backend cannot parse it at all.
			w.send(describe('S', bad()), syncMsg)
			if msg, err := w.next(); err != nil {
				t.Fatal(err)
			} else if got := render(msg); got != "ParameterDescription" && got != "Error:42P01" {
				t.Fatalf("describe statement: got %s, want 42P01", got)
			} else if got == "ParameterDescription" {
				w.expect("describe statement", "Error:42P01")
			}
			w.expect("describe statement", "Ready:I")
			for i := range unrelated {
				w.send(bind("", ok(i)), execute("", 0), syncMsg)
				w.expect(ok(i), "BindComplete", fmt.Sprintf("DataRow:%d", i), "CommandComplete:SELECT 1", "Ready:I")
			}
			recreate()
			w.send(bind("", bad()), execute("", 0), syncMsg)
			w.expect("after table recreated", "BindComplete", "CommandComplete:SELECT 0", "Ready:I")
		}},
		// the failed request skips the rest of the pipeline until Sync, then the client recovers
		{"failure skips pipeline until sync", func(t *testing.T, w *wire, _ func()) {
			w.send(bind("", ok(0)), execute("", 0), bind("", bad()), execute("", 0), bind("", ok(1)), execute("", 0), syncMsg)
			w.expect("pipeline", "BindComplete", "DataRow:0", "CommandComplete:SELECT 1", "Error:42P01", "Ready:I")
			w.send(bind("", ok(1)), execute("", 0), syncMsg)
			w.expect("after sync", "BindComplete", "DataRow:1", "CommandComplete:SELECT 1", "Ready:I")
		}},
		// several Binds of one statement in a pipeline, with a portal Describe
		{"repeated binds in one pipeline", func(t *testing.T, w *wire, _ func()) {
			w.send(bind("p1", ok(2)), bind("p2", ok(2)), describe('P', "p2"), execute("p1", 0), execute("p2", 0), syncMsg)
			w.expect("pipeline", "BindComplete", "BindComplete", "RowDescription",
				"DataRow:2", "CommandComplete:SELECT 1", "DataRow:2", "CommandComplete:SELECT 1", "Ready:I")
		}},
		// the client replaces a statement name before using it on the new backend
		{"replaced statement name", func(t *testing.T, w *wire, _ func()) {
			w.send(&pgproto3.Close{ObjectType: 'S', Name: ok(3)}, parse(ok(3), "SELECT 33"), bind("", ok(3)), execute("", 0), syncMsg)
			w.expect("replace", "CloseComplete", "ParseComplete", "BindComplete", "DataRow:33", "CommandComplete:SELECT 1", "Ready:I")
			w.send(&pgproto3.Close{ObjectType: 'S', Name: bad()}, parse(bad(), "SELECT 44"), bind("", bad()), execute("", 0), syncMsg)
			w.expect("reparse stale name", "CloseComplete", "ParseComplete", "BindComplete", "DataRow:44", "CommandComplete:SELECT 1", "Ready:I")
		}},
		// the name is still taken on the client's session, even with the same SQL
		{"duplicate named parse", func(t *testing.T, w *wire, _ func()) {
			w.send(parse(ok(4), "SELECT 4"), syncMsg)
			w.expect("same sql", "Error:42P05", "Ready:I")
			w.send(parse(ok(5), "SELECT 55"), bind("", ok(5)), execute("", 0), syncMsg)
			w.expect("other sql", "Error:42P05", "Ready:I")
			w.send(bind("", ok(5)), execute("", 0), syncMsg)
			w.expect("original kept", "BindComplete", "DataRow:5", "CommandComplete:SELECT 1", "Ready:I")
			w.send(parse("", "SELECT 6"), bind("", ""), execute("", 0), syncMsg)
			w.expect("unnamed replaces", "ParseComplete", "BindComplete", "DataRow:6", "CommandComplete:SELECT 1", "Ready:I")
		}},
		// a Close skipped after an error leaves the statement in place
		{"skipped close keeps statement", func(t *testing.T, w *wire, _ func()) {
			w.send(query("BEGIN"))
			w.expect("begin", "CommandComplete:BEGIN", "Ready:T")
			w.send(bind("", bad()), &pgproto3.Close{ObjectType: 'S', Name: ok(7)}, syncMsg)
			w.expect("close skipped", "Error:42P01", "Ready:E")
			w.send(query("ROLLBACK"))
			w.expect("rollback", "CommandComplete:ROLLBACK", "Ready:I")
			w.send(bind("", ok(7)), execute("", 0), syncMsg)
			w.expect("statement kept", "BindComplete", "DataRow:7", "CommandComplete:SELECT 1", "Ready:I")
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, tg := range targets {
				t.Run(tg.name, func(t *testing.T) {
					prRun++
					ctx := tctx(t)
					create, drop := prTable(t, "pr_stale")
					create()

					w := dialWire(ctx, t, addr[tg.name])
					w.send(parse(bad(), "SELECT a FROM pr_stale"))
					for i := range unrelated {
						w.send(parse(ok(i), fmt.Sprintf("SELECT %d", i)))
					}
					w.send(syncMsg)
					want := make([]string, 0, unrelated+2)
					for range unrelated + 1 {
						want = append(want, "ParseComplete")
					}
					w.expect("prepare", append(want, "Ready:I")...)

					drop()
					tg.move(t, w)
					tc.run(t, w, create)

					w.send(query("SELECT 1"))
					w.expect("simple query", "RowDescription", "DataRow:1", "CommandComplete:SELECT 1", "Ready:I")
				})
			}
		})
	}
}
