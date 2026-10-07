//go:build integration

package integration

import (
	"context"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgproto3"
)

// Extended protocol Flush through pggat, checked against PostgreSQL 18 run directly as a control.
// https://www.postgresql.org/docs/18/protocol-flow.html#PROTOCOL-FLOW-EXT-QUERY

const wireTimeout = 10 * time.Second

// wire is a raw protocol client. Reads have a deadline, so a pggat that waits for
// more input instead of answering fails the test rather than hanging it.
type wire struct {
	t    *testing.T
	conn net.Conn
	fe   *pgproto3.Frontend
}

func dialWire(ctx context.Context, t *testing.T, addr string) *wire {
	t.Helper()
	pc, err := pgx.Connect(ctx, connURL(addr))
	if err != nil {
		t.Fatal(err)
	}
	hc, err := pc.PgConn().Hijack()
	if err != nil {
		_ = pc.Close(ctx)
		t.Fatal(err)
	}
	w := &wire{t: t, conn: hc.Conn, fe: hc.Frontend}
	t.Cleanup(func() { _ = w.conn.Close() })
	return w
}

func (w *wire) send(msgs ...pgproto3.FrontendMessage) {
	w.t.Helper()
	for _, m := range msgs {
		w.fe.Send(m)
	}
	if err := w.conn.SetWriteDeadline(time.Now().Add(wireTimeout)); err != nil {
		w.t.Fatalf("set write deadline: %v", err)
	}
	if err := w.fe.Flush(); err != nil {
		w.t.Fatalf("send: %v", err)
	}
}

func render(msg pgproto3.BackendMessage) string {
	switch m := msg.(type) {
	case *pgproto3.ErrorResponse:
		return "Error:" + m.Code
	case *pgproto3.ReadyForQuery:
		return "Ready:" + string(m.TxStatus)
	case *pgproto3.CommandComplete:
		return "CommandComplete:" + string(m.CommandTag)
	case *pgproto3.DataRow:
		cols := make([]string, len(m.Values))
		for i, v := range m.Values {
			cols[i] = string(v)
		}
		return "DataRow:" + strings.Join(cols, ",")
	default:
		return strings.TrimPrefix(fmt.Sprintf("%T", msg), "*pgproto3.")
	}
}

// next reads one message, skipping asynchronous ones.
func (w *wire) next() (pgproto3.BackendMessage, error) {
	for {
		_ = w.conn.SetReadDeadline(time.Now().Add(wireTimeout))
		msg, err := w.fe.Receive()
		if err != nil {
			return nil, err
		}
		switch msg.(type) {
		case *pgproto3.NoticeResponse, *pgproto3.ParameterStatus, *pgproto3.NotificationResponse:
			continue
		}
		return msg, nil
	}
}

// expect reads exactly len(want) messages and compares them in order.
func (w *wire) expect(stage string, want ...string) {
	w.t.Helper()
	got := make([]string, 0, len(want))
	for range want {
		msg, err := w.next()
		if err != nil {
			w.t.Fatalf("%s: after %v waiting for %v: %v", stage, got, want[len(got):], err)
		}
		got = append(got, render(msg))
	}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		w.t.Fatalf("%s: got %v, want %v", stage, got, want)
	}
}

func parse(name, sql string) pgproto3.FrontendMessage {
	return &pgproto3.Parse{Name: name, Query: sql}
}

func bind(portal, stmt string, params ...string) pgproto3.FrontendMessage {
	b := &pgproto3.Bind{DestinationPortal: portal, PreparedStatement: stmt}
	for _, p := range params {
		b.Parameters = append(b.Parameters, []byte(p))
	}
	return b
}

func describe(which byte, name string) pgproto3.FrontendMessage {
	return &pgproto3.Describe{ObjectType: which, Name: name}
}

func execute(portal string, maxRows uint32) pgproto3.FrontendMessage {
	return &pgproto3.Execute{Portal: portal, MaxRows: maxRows}
}

func query(sql string) pgproto3.FrontendMessage { return &pgproto3.Query{String: sql} }

var (
	flushMsg pgproto3.FrontendMessage = &pgproto3.Flush{}
	syncMsg  pgproto3.FrontendMessage = &pgproto3.Sync{}
)

// withFlushTable creates a table for COPY and drops it on cleanup.
func withFlushTable(t *testing.T, name string) {
	ctx := tctx(t)
	conn, err := pgx.Connect(ctx, connURL(primaryAddr))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	if _, err := conn.Exec(ctx, "CREATE TABLE "+name+" (i integer NOT NULL)"); err != nil {
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
		if _, err := conn.Exec(ctx, "DROP TABLE "+name); err != nil {
			t.Error(err)
		}
	})
}

func TestExtendedProtocolFlush(t *testing.T) {
	t.Parallel()

	const simpleSelect1 = "RowDescription|DataRow:1|CommandComplete:SELECT 1|Ready:I"

	scenarios := []struct {
		name string
		run  func(t *testing.T, w *wire)
	}{
		{"parse describe flush", func(t *testing.T, w *wire) {
			w.send(parse("fl_s1", "SELECT $1::int"), describe('S', "fl_s1"), flushMsg)
			w.expect("parse+describe", "ParseComplete", "ParameterDescription", "RowDescription")
			w.send(bind("fl_p1", "fl_s1", "7"), describe('P', "fl_p1"), execute("fl_p1", 0), flushMsg)
			w.expect("bind+execute", "BindComplete", "RowDescription", "DataRow:7", "CommandComplete:SELECT 1")
			w.send(syncMsg)
			w.expect("sync", "Ready:I")
			w.send(query("SELECT 1"))
			w.expect("simple query after pipeline", strings.Split(simpleSelect1, "|")...)
		}},
		{"execute results before sync", func(t *testing.T, w *wire) {
			w.send(parse("", "SELECT generate_series(1, 3)"), bind("", ""), execute("", 0), flushMsg)
			w.expect("rows", "ParseComplete", "BindComplete", "DataRow:1", "DataRow:2", "DataRow:3", "CommandComplete:SELECT 3")
			w.send(syncMsg)
			w.expect("sync", "Ready:I")
		}},
		{"suspended portal progresses across flushes", func(t *testing.T, w *wire) {
			w.send(parse("", "SELECT generate_series(1, 3)"), bind("", ""), execute("", 2), flushMsg)
			w.expect("first batch", "ParseComplete", "BindComplete", "DataRow:1", "DataRow:2", "PortalSuspended")
			w.send(execute("", 2), flushMsg)
			w.expect("second batch", "DataRow:3", "CommandComplete:SELECT 1")
			w.send(syncMsg)
			w.expect("sync", "Ready:I")
		}},
		{"close and empty query", func(t *testing.T, w *wire) {
			w.send(parse("fl_s2", "SELECT 1"), flushMsg)
			w.expect("parse", "ParseComplete")
			w.send(&pgproto3.Close{ObjectType: 'S', Name: "fl_s2"}, flushMsg)
			w.expect("close", "CloseComplete")
			w.send(parse("", ""), bind("", ""), execute("", 0), flushMsg)
			w.expect("empty", "ParseComplete", "BindComplete", "EmptyQueryResponse")
			w.send(syncMsg)
			w.expect("sync", "Ready:I")
		}},
		{"pipeline ordering across flush and sync", func(t *testing.T, w *wire) {
			w.send(parse("", "SELECT 1"), bind("", ""), execute("", 0), flushMsg)
			w.expect("batch 1", "ParseComplete", "BindComplete", "DataRow:1", "CommandComplete:SELECT 1")
			w.send(parse("", "SELECT 2"), bind("", ""), describe('P', ""), execute("", 0), syncMsg)
			w.expect("batch 2", "ParseComplete", "BindComplete", "RowDescription", "DataRow:2", "CommandComplete:SELECT 1", "Ready:I")
		}},
		{"error then flush then sync", func(t *testing.T, w *wire) {
			w.send(parse("", "SELECT 1/g FROM generate_series(0, 1) g"), bind("", ""), execute("", 0), flushMsg)
			w.expect("error", "ParseComplete", "BindComplete", "Error:22012")
			// The server discards these until Sync. The next message is the Sync reply.
			w.send(parse("fl_s3", "SELECT 1"), describe('S', "fl_s3"), flushMsg, syncMsg)
			w.expect("discarded", "Ready:I")
			w.send(parse("fl_s4", "SELECT $1::int"), describe('S', "fl_s4"), flushMsg)
			w.expect("recovery", "ParseComplete", "ParameterDescription", "RowDescription")
			w.send(syncMsg)
			w.expect("sync", "Ready:I")
			w.send(query("SELECT 1"))
			w.expect("simple query", strings.Split(simpleSelect1, "|")...)
		}},
		{"parse error then sync", func(t *testing.T, w *wire) {
			w.send(parse("", "SELEC 1"), flushMsg)
			w.expect("error", "Error:42601")
			w.send(syncMsg)
			w.expect("sync", "Ready:I")
		}},
		{"error in transaction", func(t *testing.T, w *wire) {
			w.send(query("BEGIN"))
			w.expect("begin", "CommandComplete:BEGIN", "Ready:T")
			w.send(parse("", "SELECT 1/g FROM generate_series(0, 1) g"), bind("", ""), execute("", 0), flushMsg)
			w.expect("error", "ParseComplete", "BindComplete", "Error:22012")
			w.send(syncMsg)
			w.expect("sync", "Ready:E")
			w.send(query("ROLLBACK"))
			w.expect("rollback", "CommandComplete:ROLLBACK", "Ready:I")
		}},
		{"sync and flush alone", func(t *testing.T, w *wire) {
			w.send(syncMsg)
			w.expect("sync only", "Ready:I")
			w.send(flushMsg, syncMsg)
			w.expect("flush then sync", "Ready:I")
			w.send(flushMsg, flushMsg, syncMsg)
			w.expect("flushes then sync", "Ready:I")
			w.send(query("SELECT 1"))
			w.expect("query", strings.Split(simpleSelect1, "|")...)
		}},
		{"large result streaming", func(t *testing.T, w *wire) {
			const rows = 200000
			w.send(
				parse("", fmt.Sprintf("SELECT g, repeat('x', 100) FROM generate_series(1, %d) g", rows)),
				bind("", ""), execute("", 0), flushMsg,
			)
			w.expect("start", "ParseComplete", "BindComplete")
			for i := 1; i <= rows; i++ {
				msg, err := w.next()
				if err != nil {
					t.Fatalf("row %d: %v", i, err)
				}
				row, ok := msg.(*pgproto3.DataRow)
				if !ok || string(row.Values[0]) != fmt.Sprint(i) {
					t.Fatalf("row %d: got %s", i, render(msg))
				}
			}
			w.expect("complete", fmt.Sprintf("CommandComplete:SELECT %d", rows))
			w.send(syncMsg)
			w.expect("sync", "Ready:I")
		}},
	}

	targets := []struct {
		name string
		addr *string
	}{
		{"direct", &primaryAddr},
		{"transaction", &transactionAddr},
		{"session", &sessionAddr},
		{"hybrid", &hybridAddr},
	}
	for _, target := range targets {
		for _, s := range scenarios {
			t.Run(target.name+"/"+s.name, func(t *testing.T) {
				t.Parallel()
				w := dialWire(tctx(t), t, *target.addr)
				s.run(t, w)
			})
		}
	}
}

// PostgreSQL ignores Flush and Sync while a COPY FROM STDIN is waiting for data.
// The COPY's CommandComplete follows CopyDone, and its own Flush or Sync releases it.
func TestExtendedProtocolCopyIn(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		name string
		run  func(t *testing.T, w *wire, table string)
	}{
		{"started by flush", func(t *testing.T, w *wire, table string) {
			w.send(parse("", "COPY "+table+" FROM STDIN"), bind("", ""), execute("", 0), flushMsg)
			w.expect("start", "ParseComplete", "BindComplete", "CopyInResponse")
			w.send(flushMsg, syncMsg)
			w.send(&pgproto3.CopyData{Data: []byte("1\n2\n")}, &pgproto3.CopyDone{}, flushMsg)
			w.expect("copy done", "CommandComplete:COPY 2")
			w.send(syncMsg)
			w.expect("sync", "Ready:I")
			w.send(query("SELECT count(*) FROM " + table))
			w.expect("count", "RowDescription", "DataRow:2", "CommandComplete:SELECT 1", "Ready:I")
		}},
		{"started by sync", func(t *testing.T, w *wire, table string) {
			w.send(parse("", "COPY "+table+" FROM STDIN"), bind("", ""), execute("", 0), syncMsg)
			w.expect("start", "ParseComplete", "BindComplete", "CopyInResponse")
			w.send(&pgproto3.CopyData{Data: []byte("1\n")}, &pgproto3.CopyDone{}, flushMsg)
			w.expect("copy done", "CommandComplete:COPY 1")
			w.send(syncMsg)
			w.expect("sync", "Ready:I")
			w.send(query("SELECT count(*) FROM " + table))
			w.expect("count", "RowDescription", "DataRow:1", "CommandComplete:SELECT 1", "Ready:I")
		}},
		{"copy fail then sync", func(t *testing.T, w *wire, table string) {
			w.send(parse("", "COPY "+table+" FROM STDIN"), bind("", ""), execute("", 0), flushMsg)
			w.expect("start", "ParseComplete", "BindComplete", "CopyInResponse")
			w.send(&pgproto3.CopyFail{Message: "stop"}, flushMsg)
			w.expect("fail", "Error:57014")
			w.send(syncMsg)
			w.expect("sync", "Ready:I")
			w.send(query("SELECT count(*) FROM " + table))
			w.expect("count", "RowDescription", "DataRow:0", "CommandComplete:SELECT 1", "Ready:I")
		}},
		{"invalid data then sync", func(t *testing.T, w *wire, table string) {
			w.send(parse("", "COPY "+table+" FROM STDIN"), bind("", ""), execute("", 0), flushMsg)
			w.expect("start", "ParseComplete", "BindComplete", "CopyInResponse")
			w.send(&pgproto3.CopyData{Data: []byte("x\n")}, &pgproto3.CopyDone{}, flushMsg)
			w.expect("fail", "Error:22P02")
			w.send(syncMsg)
			w.expect("sync", "Ready:I")
		}},
	}

	targets := []struct {
		name string
		addr *string
	}{
		{"direct", &primaryAddr},
		{"transaction", &transactionAddr},
		{"session", &sessionAddr},
		{"hybrid", &hybridAddr},
	}
	for _, target := range targets {
		for i, s := range scenarios {
			table := fmt.Sprintf("fl_copy_%s_%d", target.name, i)
			t.Run(target.name+"/"+s.name, func(t *testing.T) {
				t.Parallel()
				withFlushTable(t, table)
				w := dialWire(tctx(t), t, *target.addr)
				s.run(t, w, table)
			})
		}
	}
}

// COPY TO STDOUT through the extended protocol, ended by Flush or Sync, and failing midway.
func TestExtendedProtocolCopyOut(t *testing.T) {
	t.Parallel()

	const copy3 = "COPY (SELECT generate_series(1, 3)) TO STDOUT"
	scenarios := []struct {
		name string
		run  func(t *testing.T, w *wire)
	}{
		{"ended by flush", func(t *testing.T, w *wire) {
			w.send(parse("", copy3), bind("", ""), execute("", 0), flushMsg)
			w.expect("copy", "ParseComplete", "BindComplete", "CopyOutResponse",
				"CopyData", "CopyData", "CopyData", "CopyDone", "CommandComplete:COPY 3")
			w.send(syncMsg)
			w.expect("sync", "Ready:I")
		}},
		{"ended by sync", func(t *testing.T, w *wire) {
			w.send(parse("", copy3), bind("", ""), execute("", 0), syncMsg)
			w.expect("copy", "ParseComplete", "BindComplete", "CopyOutResponse",
				"CopyData", "CopyData", "CopyData", "CopyDone", "CommandComplete:COPY 3", "Ready:I")
			w.send(query("SELECT 1"))
			w.expect("query", "RowDescription", "DataRow:1", "CommandComplete:SELECT 1", "Ready:I")
		}},
		{"error midway then flush then sync", func(t *testing.T, w *wire) {
			w.send(parse("", "COPY (SELECT 1/(g - 2) FROM generate_series(1, 3) g) TO STDOUT"),
				bind("", ""), execute("", 0), flushMsg)
			w.expect("copy", "ParseComplete", "BindComplete", "CopyOutResponse", "CopyData", "Error:22012")
			w.send(flushMsg, syncMsg)
			w.expect("sync", "Ready:I")
			w.send(query("SELECT 1"))
			w.expect("query", "RowDescription", "DataRow:1", "CommandComplete:SELECT 1", "Ready:I")
		}},
	}

	targets := []struct {
		name string
		addr *string
	}{
		{"direct", &primaryAddr},
		{"transaction", &transactionAddr},
		{"session", &sessionAddr},
		{"hybrid", &hybridAddr},
	}
	for _, target := range targets {
		for _, s := range scenarios {
			t.Run(target.name+"/"+s.name, func(t *testing.T) {
				t.Parallel()
				s.run(t, dialWire(tctx(t), t, *target.addr))
			})
		}
	}
}

// A relay must not hold back data the server has already sent. PostgreSQL flushes its output when it
// emits a WARNING, so rows produced before it are visible to a direct client while a later row blocks on
// a real advisory lock. The statement below produces rows 1 and 2, a warning at row 3 (unlocking an
// advisory lock this session never took), and blocks at row 5. Direct PostgreSQL defines how many rows a
// client sees during the stall, so that count is measured, not hardcoded. Each pggat mode must then deliver
// that prefix, in order, while the lock is still held. Reads use the normal wire timeout, so a relay that
// holds the rows back fails by timeout. Serial, so the direct and pggat runs see the same load.
func TestExtendedProtocolStalledStreamDelivery(t *testing.T) {
	const (
		rows    = 5
		lockKey = 0x666c0010
		warnKey = 0x666c0011
	)
	sel := fmt.Sprintf("SELECT g, CASE WHEN g = 3 THEN pg_advisory_unlock(%d)::text WHEN g = %d THEN pg_advisory_xact_lock(%d)::text ELSE 'x' END FROM generate_series(1, %d) g",
		warnKey, rows, lockKey, rows)
	rowText := func(i int) (string, string) {
		switch i {
		case 3:
			return "3", "false"
		case rows:
			return fmt.Sprint(i), ""
		}
		return fmt.Sprint(i), "x"
	}

	cases := []struct {
		name string
		sql  string
		// payload returns the payload of a counted message, or false for other messages.
		payload func(pgproto3.BackendMessage) (string, bool)
		want    func(i int) string
	}{
		{
			name: "rows",
			sql:  sel,
			payload: func(m pgproto3.BackendMessage) (string, bool) {
				row, ok := m.(*pgproto3.DataRow)
				if !ok {
					return "", false
				}
				return string(row.Values[0]) + "," + string(row.Values[1]), true
			},
			want: func(i int) string {
				a, b := rowText(i)
				return a + "," + b
			},
		},
		{
			name: "copy out",
			sql:  "COPY (" + sel + ") TO STDOUT",
			payload: func(m pgproto3.BackendMessage) (string, bool) {
				data, ok := m.(*pgproto3.CopyData)
				if !ok {
					return "", false
				}
				return string(data.Data), true
			},
			want: func(i int) string {
				a, b := rowText(i)
				return a + "\t" + b + "\n"
			},
		},
	}

	// run executes the statement against addr with the lock held. A negative prefix means measure it with a
	// quiet read, otherwise read exactly that many counted messages. It returns the prefix length.
	run := func(t *testing.T, c int, addr string, prefix int) int {
		ctx := tctx(t)
		tc := cases[c]

		observer := lifecycleConnect(ctx, t, primaryAddr)
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

		w := dialWire(ctx, t, addr)
		w.send(parse("", tc.sql), bind("", ""), execute("", 0), flushMsg)
		for {
			var waiting bool
			err := observer.QueryRow(ctx,
				`SELECT EXISTS (SELECT 1 FROM pg_locks WHERE locktype = 'advisory' AND NOT granted AND objid = $1::oid)`, lockKey,
			).Scan(&waiting)
			if err != nil {
				t.Fatal(err)
			}
			if waiting {
				break
			}
			select {
			case <-ctx.Done():
				t.Fatalf("execute never waited on the advisory lock: %v", ctx.Err())
			case <-time.After(10 * time.Millisecond):
			}
		}

		got := 0
		check := func(msg pgproto3.BackendMessage) {
			p, ok := tc.payload(msg)
			if !ok {
				switch msg.(type) {
				case *pgproto3.ParseComplete, *pgproto3.BindComplete, *pgproto3.CopyOutResponse, *pgproto3.CopyDone,
					*pgproto3.NoticeResponse, *pgproto3.ParameterStatus:
					return
				}
				t.Fatalf("unexpected %s after %d messages", render(msg), got)
			}
			got++
			if want := tc.want(got); p != want {
				t.Fatalf("message %d = %q, want %q", got, p, want)
			}
		}

		if prefix < 0 {
			// Direct PostgreSQL: whatever arrives before a quiet second is what it sends while blocked.
			for {
				if err := w.conn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
					t.Fatal(err)
				}
				msg, err := w.fe.Receive()
				if err != nil {
					if ne, ok := err.(net.Error); !ok || !ne.Timeout() {
						t.Fatal(err)
					}
					break
				}
				check(msg)
			}
			prefix = got
		} else {
			for got < prefix {
				if err := w.conn.SetReadDeadline(time.Now().Add(wireTimeout)); err != nil {
					t.Fatal(err)
				}
				msg, err := w.fe.Receive()
				if err != nil {
					t.Fatalf("after %d of %d messages that direct PostgreSQL delivers while the Execute is blocked: %v", got, prefix, err)
				}
				check(msg)
			}
		}

		unlock()
		for {
			msg, err := w.next()
			if err != nil {
				t.Fatalf("after unlock, %d messages: %v", got, err)
			}
			if _, ok := msg.(*pgproto3.CommandComplete); ok {
				break
			}
			check(msg)
		}
		if got != rows {
			t.Fatalf("received %d messages, want %d", got, rows)
		}
		w.send(syncMsg)
		w.expect("sync", "Ready:I")
		return prefix
	}

	for c := range cases {
		t.Run(cases[c].name, func(t *testing.T) {
			prefix := run(t, c, primaryAddr, -1)
			if prefix == 0 || prefix == rows {
				t.Fatalf("direct PostgreSQL delivered %d of %d messages while blocked, the case does not measure buffering", prefix, rows)
			}
			t.Logf("direct PostgreSQL delivered %d of %d messages while blocked", prefix, rows)
			for _, target := range []struct{ name, addr string }{
				{"transaction", transactionAddr},
				{"session", sessionAddr},
				{"hybrid", hybridAddr},
			} {
				t.Run(target.name, func(t *testing.T) {
					run(t, c, target.addr, prefix)
				})
			}
		})
	}
}

// Responses released by a Flush must reach the client while a later Execute in the same
// write is blocked on a real lock. PostgreSQL does not send an Execute's own responses
// before it finishes, so the early Flush is what releases ParseComplete and BindComplete.
// Each target has its own advisory lock key, so the parallel subtests do not block each other.
func TestExtendedProtocolFlushEarlyResponses(t *testing.T) {
	t.Parallel()

	targets := []struct {
		name string
		addr *string
		key  int
	}{
		{"direct", &primaryAddr, 0x666c0001},
		{"transaction", &transactionAddr, 0x666c0002},
		{"session", &sessionAddr, 0x666c0003},
		{"hybrid", &hybridAddr, 0x666c0004},
	}
	for _, target := range targets {
		t.Run(target.name, func(t *testing.T) {
			t.Parallel()
			ctx := tctx(t)

			observer := lifecycleConnect(ctx, t, primaryAddr)
			if _, err := observer.Exec(ctx, "SELECT pg_advisory_lock($1)", target.key); err != nil {
				t.Fatal(err)
			}
			locked := true
			unlock := func() {
				if !locked {
					return
				}
				locked = false
				if _, err := observer.Exec(ctx, "SELECT pg_advisory_unlock($1)", target.key); err != nil {
					t.Fatal(err)
				}
			}
			defer unlock()

			w := dialWire(ctx, t, *target.addr)
			w.send(
				parse("", "SELECT pg_advisory_xact_lock($1::bigint)::text"),
				bind("", "", fmt.Sprint(target.key)),
				flushMsg,
				execute("", 0),
				flushMsg,
			)
			w.expect("before lock release", "ParseComplete", "BindComplete")

			// The Execute must be blocked on the lock, not finished.
			for {
				var waiting bool
				err := observer.QueryRow(ctx,
					`SELECT EXISTS (SELECT 1 FROM pg_locks WHERE locktype = 'advisory' AND NOT granted AND objid = $1::oid)`,
					target.key,
				).Scan(&waiting)
				if err != nil {
					t.Fatal(err)
				}
				if waiting {
					break
				}
				select {
				case <-ctx.Done():
					t.Fatalf("execute never waited on the advisory lock: %v", ctx.Err())
				case <-time.After(10 * time.Millisecond):
				}
			}

			unlock()
			w.expect("after lock release", "DataRow:", "CommandComplete:SELECT 1")
			w.send(syncMsg)
			w.expect("sync", "Ready:I")
		})
	}
}

// A pipeline that is flushed but not synced must leave nothing behind for the
// next client of the same backend. Serial: singleServerAddr has one backend.
func TestExtendedProtocolFlushReleasesBackend(t *testing.T) {
	ctx := tctx(t)

	probe := func(t *testing.T, stage string) {
		t.Helper()
		w := dialWire(ctx, t, singleServerAddr)
		w.send(query("SELECT 1"))
		w.expect(stage, "RowDescription", "DataRow:1", "CommandComplete:SELECT 1", "Ready:I")
	}

	t.Run("after sync", func(t *testing.T) {
		a := dialWire(ctx, t, singleServerAddr)
		a.send(parse("", "SELECT 1"), bind("", ""), execute("", 0), flushMsg)
		a.expect("a pipeline", "ParseComplete", "BindComplete", "DataRow:1", "CommandComplete:SELECT 1")
		a.send(syncMsg)
		a.expect("a sync", "Ready:I")
		probe(t, "b")
		a.send(query("SELECT 1"))
		a.expect("a again", "RowDescription", "DataRow:1", "CommandComplete:SELECT 1", "Ready:I")
	})

	t.Run("client disconnects after flush", func(t *testing.T) {
		a := dialWire(ctx, t, singleServerAddr)
		a.send(parse("fl_gone", "SELECT $1::int"), describe('S', "fl_gone"), flushMsg)
		a.expect("a pipeline", "ParseComplete", "ParameterDescription", "RowDescription")
		_ = a.conn.Close()
		probe(t, "b")
	})
}
