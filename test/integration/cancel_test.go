//go:build integration

package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// TestHybridCancel checks that a client's cancel request ends its query on
// the server, in every hybrid mode. pgx sends a cancel request when a query's
// context is canceled. The hybrid pool must forward it to the server
// connection the client is paired with; if it forwards it to the wrong pool,
// the cancel is dropped and the query keeps running in PostgreSQL, holding
// its locks and its pooled server connection.
func TestHybridCancel(t *testing.T) {
	t.Parallel()
	// ro is not covered: hybrid.Gatfile configures no replica, so an ro
	// client has no pool to acquire from.
	for _, mode := range []string{"", "wo"} {
		name := mode
		if name == "" {
			name = "rw"
		}
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			url := connURL(hybridAddr)
			if mode != "" {
				url = connURL(hybridAddr, "hybrid.mode="+mode)
			}

			conn, err := pgx.Connect(ctx, url)
			if err != nil {
				t.Fatalf("connect: %v", err)
			}
			defer conn.Close(ctx)

			observer, err := pgx.Connect(ctx, connURL(primaryAddr))
			if err != nil {
				t.Fatalf("connect observer: %v", err)
			}
			defer observer.Close(ctx)

			marker := fmt.Sprintf("hybrid-cancel-%s-%d", name, time.Now().UnixNano())
			running := func() int {
				var n int
				if err := observer.QueryRow(ctx, `SELECT count(*) FROM pg_stat_activity
WHERE state = 'active' AND query LIKE '%' || $1 || '%' AND pid <> pg_backend_pid()`, marker).Scan(&n); err != nil {
					t.Fatalf("observe: %v", err)
				}
				return n
			}
			t.Cleanup(func() {
				_, _ = observer.Exec(context.Background(), `SELECT pg_terminate_backend(pid) FROM pg_stat_activity
WHERE query LIKE '%' || $1 || '%' AND pid <> pg_backend_pid()`, marker)
			})

			// Cancel the query from the client while it runs on the server.
			qctx, cancel := context.WithCancel(ctx)
			done := make(chan error, 1)
			go func() {
				_, err := conn.Exec(qctx, "SELECT pg_sleep(30) /* "+marker+" */")
				done <- err
			}()
			deadline := time.Now().Add(5 * time.Second)
			for running() == 0 {
				if time.Now().After(deadline) {
					t.Fatal("query never started on the server")
				}
				time.Sleep(20 * time.Millisecond)
			}
			cancel()
			if err := <-done; err == nil {
				t.Fatal("canceled query returned no error")
			}

			// The server must end the query, not just the client give up on it.
			deadline = time.Now().Add(5 * time.Second)
			for running() > 0 {
				if time.Now().After(deadline) {
					t.Fatal("query still running on the server after its client canceled it")
				}
				time.Sleep(50 * time.Millisecond)
			}
		})
	}
}
