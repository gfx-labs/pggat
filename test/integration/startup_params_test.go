//go:build integration

package integration

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgproto3"
)

type startupSettings struct {
	pid                             int
	timezone, appName, intervalType string
}

func readStartupSettings(ctx context.Context, t *testing.T, conn *pgx.Conn) startupSettings {
	t.Helper()
	var s startupSettings
	err := conn.QueryRow(ctx,
		"SELECT pg_backend_pid(), current_setting('timezone'), current_setting('application_name'), current_setting('intervalstyle')",
	).Scan(&s.pid, &s.timezone, &s.appName, &s.intervalType)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// A rejected startup setting must preserve the backend PID and defaults.
func TestInvalidStartupParameterKeepsBackend(t *testing.T) {
	for _, tc := range []struct{ name, addr string }{
		{"transaction", singleServerAddr},
		{"session", sessionSingleAddr},
		{"hybrid", hybridSingleAddr},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := tctx(t)

			first := lifecycleConnect(ctx, t, tc.addr)
			want := readStartupSettings(ctx, t, first)
			// Session pooling holds the backend until the client leaves.
			if err := first.Close(ctx); err != nil {
				t.Fatal(err)
			}

			// Parameter order varies per attempt, so different settings may be applied before the bad one fails.
			for i := range 8 {
				bad, err := pgx.Connect(ctx, connURL(tc.addr, "application_name=leaked", "intervalstyle=sql_standard", "timezone=Not/AZone"))
				if bad != nil {
					_ = bad.Close(ctx)
				}
				var pgErr *pgconn.PgError
				if !errors.As(err, &pgErr) || pgErr.Code != "22023" {
					t.Fatalf("attempt %d: got %v, want SQLSTATE 22023", i, err)
				}

				next := lifecycleConnect(ctx, t, tc.addr)
				got := readStartupSettings(ctx, t, next)
				if got != want {
					t.Fatalf("attempt %d: settings after rejected startup %+v, want %+v", i, got, want)
				}
				if err := next.Close(ctx); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

// An idle client's Terminate must not wait for another client's backend.
func TestTerminateDoesNotWaitForBackend(t *testing.T) {
	ctx := tctx(t)

	// Startup pairs with the backend, so connect both clients before it is held.
	leaving := lifecycleConnect(ctx, t, singleServerAddr)
	holder := lifecycleConnect(ctx, t, singleServerAddr)

	tx, err := holder.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var pid int
	if err := tx.QueryRow(ctx, "SELECT pg_backend_pid()").Scan(&pid); err != nil {
		t.Fatal(err)
	}

	front := leaving.PgConn().Frontend()
	front.Send(&pgproto3.Terminate{})
	if err := front.Flush(); err != nil {
		t.Fatal(err)
	}

	// pggat closes its end after Terminate. The deadline only bounds a failing run.
	sock := leaving.PgConn().Conn()
	if err := sock.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	_, err = sock.Read(make([]byte, 1))
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		t.Fatal("pggat kept the terminated client open while the only backend was busy")
	}
	if !errors.Is(err, io.EOF) {
		t.Fatalf("read after Terminate: got %v, want EOF", err)
	}

	// The holder's transaction was not disturbed.
	var after int
	if err := tx.QueryRow(ctx, "SELECT pg_backend_pid()").Scan(&after); err != nil || after != pid {
		t.Fatalf("holder's transaction: pid %d, err %v, want pid %d", after, err, pid)
	}
	if err := tx.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
}
