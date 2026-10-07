//go:build integration

package integration

import (
	"testing"
	"time"
)

// TestIdleTerminatedBackendIsReplaced checks that a client keeps working after the
// pooled backend it last used is terminated while idle. singleServerAddr has one
// backend, so the client's next query must be sent to a new one.
// Contract after Odyssey test/pg_regress/tests/broken_conn/sql/server_closed.sql at 456131c5.
func TestIdleTerminatedBackendIsReplaced(t *testing.T) {
	ctx := tctx(t)

	client := lifecycleConnect(ctx, t, singleServerAddr)
	observer := lifecycleConnect(ctx, t, primaryAddr)

	var pid int
	if err := client.QueryRow(ctx, "SELECT pg_backend_pid()").Scan(&pid); err != nil {
		t.Fatal(err)
	}

	var terminated bool
	if err := observer.QueryRow(ctx, "SELECT pg_terminate_backend($1)", pid).Scan(&terminated); err != nil {
		t.Fatal(err)
	}
	if !terminated {
		t.Fatalf("backend %d was not terminated", pid)
	}
	// pg_terminate_backend only signals. Wait until the backend is gone.
	for {
		var alive bool
		if err := observer.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_stat_activity WHERE pid = $1)", pid).Scan(&alive); err != nil {
			t.Fatal(err)
		}
		if !alive {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatalf("backend %d never exited: %v", pid, ctx.Err())
		case <-time.After(10 * time.Millisecond):
		}
	}

	var after int
	if err := client.QueryRow(ctx, "SELECT pg_backend_pid()").Scan(&after); err != nil {
		t.Fatalf("query after idle backend was terminated: %v", err)
	}
	if after == pid {
		t.Fatalf("query ran on terminated backend %d", pid)
	}
}
