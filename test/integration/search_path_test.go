//go:build integration

package integration

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// Needs PostgreSQL 18+, which reports search_path in ParameterStatus (see README.md).

const defaultPath = `"$user", public`

// Reports the backend pid, the setting, and the schema that holds the unqualified table sp_which.
const stateSQL = `SELECT pg_backend_pid(), current_setting('search_path'),
	coalesce((SELECT n.nspname FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE c.oid = to_regclass('sp_which')), '')`

// Each schema holds an empty sp_which, so resolving it shows which schema search_path selects.
var schemas = []string{
	"sp_startup", "sp_runtime", "Sp Odd's", "sp_leak_startup", "sp_leak_set", "sp_base", "sp_tx",
}

type state struct {
	pid          int
	path, schema string
}

type rower interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func tctx(t *testing.T) context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func stateOf(ctx context.Context, t *testing.T, q rower) state {
	t.Helper()
	var s state
	if err := q.QueryRow(ctx, stateSQL).Scan(&s.pid, &s.path, &s.schema); err != nil {
		t.Fatal(err)
	}
	return s
}

func createSchemas(t *testing.T) {
	ctx := tctx(t)
	conn, err := pgx.Connect(ctx, connURL(primaryAddr))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		defer conn.Close(ctx)
		for _, name := range schemas {
			if _, err := conn.Exec(ctx, "DROP SCHEMA IF EXISTS "+pgx.Identifier{name}.Sanitize()+" CASCADE"); err != nil {
				t.Error(err)
			}
		}
	})
	for _, name := range schemas {
		id := pgx.Identifier{name}.Sanitize()
		for _, sql := range []string{"CREATE SCHEMA " + id, "CREATE TABLE " + id + ".sp_which (id int)"} {
			if _, err := conn.Exec(ctx, sql); err != nil {
				t.Fatal(err)
			}
		}
	}
}

// occupy opens default clients, each in a transaction, until one runs on backend pid.
// It returns that client's transaction and a release func. While held, nobody else can use pid.
func occupy(ctx context.Context, t *testing.T, addr string, pid int) (pgx.Tx, func()) {
	t.Helper()
	type held struct {
		conn *pgx.Conn
		tx   pgx.Tx
	}
	var all []held
	release := func() {
		for _, h := range all {
			_ = h.tx.Rollback(ctx)
			_ = h.conn.Close(ctx)
		}
		all = nil
	}
	for range 5 {
		for range 12 {
			conn, err := pgx.Connect(ctx, connURL(addr))
			if err != nil {
				release()
				t.Fatal(err)
			}
			tx, err := conn.Begin(ctx)
			if err != nil {
				_ = conn.Close(ctx)
				release()
				t.Fatal(err)
			}
			all = append(all, held{conn, tx})
			var got int
			if err := tx.QueryRow(ctx, "SELECT pg_backend_pid()").Scan(&got); err != nil {
				release()
				t.Fatal(err)
			}
			if got == pid {
				return tx, release
			}
		}
		// The backend may still be returning to the pool.
		release()
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("could not reach backend %d through %s", pid, addr)
	return nil, nil
}

type mode struct {
	name, addr string
	// pinned: a client stays on one backend for its whole session.
	pinned bool
}

type client struct {
	t    *testing.T
	ctx  context.Context
	mode mode
	conn *pgx.Conn
	pid  int // backend of the last statement
}

func newClient(t *testing.T, ctx context.Context, m mode, query ...string) *client {
	t.Helper()
	conn, err := pgx.Connect(ctx, connURL(m.addr, query...))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(ctx) })
	return &client{t: t, ctx: ctx, mode: m, conn: conn}
}

// elsewhere occupies the client's last backend so its next statement must run on another one.
func (c *client) elsewhere() func() {
	if c.mode.pinned || c.pid == 0 {
		return func() {}
	}
	_, release := occupy(c.ctx, c.t, c.mode.addr, c.pid)
	return release
}

// moved records the backend of a statement and checks it switched (or stayed, if pinned).
func (c *client) moved(pid int) {
	c.t.Helper()
	if c.pid != 0 && (pid == c.pid) == !c.mode.pinned {
		c.t.Fatalf("backend %d -> %d with pinned=%v", c.pid, pid, c.mode.pinned)
	}
	c.pid = pid
}

func (c *client) check(desc string, got state, path, schema string) {
	c.t.Helper()
	// A startup value may keep "a,b" where SET reports "a, b".
	norm := func(s string) string { return strings.ReplaceAll(s, ", ", ",") }
	if norm(got.path) != norm(path) || got.schema != schema {
		c.t.Fatalf("%s (backend %d): search_path %q resolves to %q, want %q resolving to %q",
			desc, got.pid, got.path, got.schema, path, schema)
	}
}

// query runs on a different backend than the previous statement and checks the state there.
func (c *client) query(desc, path, schema string) {
	c.t.Helper()
	release := c.elsewhere()
	defer release()
	got := stateOf(c.ctx, c.t, c.conn)
	c.moved(got.pid)
	c.check(desc, got, path, schema)
}

// set runs a statement and records its backend. Both statements share one simple query, so one backend.
func (c *client) set(sql string) {
	c.t.Helper()
	results, err := c.conn.PgConn().Exec(c.ctx, sql+"; SELECT pg_backend_pid()").ReadAll()
	if err != nil {
		c.t.Fatal(err)
	}
	pid, err := strconv.Atoi(string(results[len(results)-1].Rows[0][0]))
	if err != nil {
		c.t.Fatal(err)
	}
	c.pid = pid
}

func TestSearchPath(t *testing.T) {
	createSchemas(t)

	for _, m := range []mode{
		{"transaction", transactionAddr, false},
		{"session", sessionAddr, true},
		{"hybrid", hybridAddr, false},
	} {
		t.Run(m.name, func(t *testing.T) {
			// First, so no earlier subtest has changed this pool's backends.
			t.Run("defaults do not leak", func(t *testing.T) {
				for _, tc := range []struct {
					name, startup, set, path, schema string
				}{
					{"startup parameter", "search_path=sp_leak_startup,public", "", "sp_leak_startup, public", "sp_leak_startup"},
					{"SET", "", "SET search_path TO sp_leak_set, public", "sp_leak_set, public", "sp_leak_set"},
				} {
					t.Run(tc.name, func(t *testing.T) {
						ctx := tctx(t)
						var q []string
						if tc.startup != "" {
							q = append(q, tc.startup)
						}
						owner := newClient(t, ctx, m, q...)
						if tc.set != "" {
							owner.set(tc.set)
						}
						owner.query("owner", tc.path, tc.schema)
						if m.pinned {
							_ = owner.conn.Close(ctx) // frees the backend
						}

						// A client that never set search_path, on the owner's backend.
						other, release := occupy(ctx, t, m.addr, owner.pid)
						defer release()
						owner.check("other client", stateOf(ctx, t, other), defaultPath, "")
						if !m.pinned {
							release()
							owner.query("owner after", tc.path, tc.schema)
						}
					})
				}
			})

			t.Run("startup parameter", func(t *testing.T) {
				c := newClient(t, tctx(t), m, "search_path=sp_startup,public")
				c.query("first", "sp_startup, public", "sp_startup")
				c.query("switched", "sp_startup, public", "sp_startup")
			})

			t.Run("SET and RESET", func(t *testing.T) {
				c := newClient(t, tctx(t), m)
				c.set("SET search_path TO sp_runtime, public")
				c.query("SET", "sp_runtime, public", "sp_runtime")
				c.set(`SET search_path TO "Sp Odd's", public`)
				c.query("quoted SET", `"Sp Odd's", public`, "Sp Odd's")
				c.set("RESET search_path")
				c.query("RESET", defaultPath, "")
			})

			for _, tc := range []struct {
				name, stmt  string
				commit      bool
				inTx, after string
			}{
				{"SET LOCAL ends with commit", "SET LOCAL search_path TO sp_tx, public", true, "sp_tx", "sp_base"},
				{"SET undone by rollback", "SET search_path TO sp_tx, public", false, "sp_tx", "sp_base"},
				{"SET kept by commit", "SET search_path TO sp_tx, public", true, "sp_tx", "sp_tx"},
			} {
				t.Run(tc.name, func(t *testing.T) {
					ctx := tctx(t)
					c := newClient(t, ctx, m)
					c.set("SET search_path TO sp_base, public")

					release := c.elsewhere()
					tx, err := c.conn.Begin(ctx)
					if err != nil {
						release()
						t.Fatal(err)
					}
					first := stateOf(ctx, t, tx)
					release()
					c.moved(first.pid)
					c.check("before", first, "sp_base, public", "sp_base")

					if _, err := tx.Exec(ctx, tc.stmt); err != nil {
						t.Fatal(err)
					}
					in := stateOf(ctx, t, tx)
					if in.pid != first.pid {
						t.Fatalf("transaction moved from backend %d to %d", first.pid, in.pid)
					}
					c.check("in transaction", in, tc.inTx+", public", tc.inTx)

					if tc.commit {
						err = tx.Commit(ctx)
					} else {
						err = tx.Rollback(ctx)
					}
					if err != nil {
						t.Fatal(err)
					}
					c.query("after", tc.after+", public", tc.after)
				})
			}
		})
	}
}
