//go:build integration

package integration

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// A write inside a transaction the client declared read only must fail with the
// server's ERROR 25006 and leave the connection usable, not trigger a hybrid replica fallback.
func TestHybridReadOnlyTransaction(t *testing.T) {
	t.Parallel()
	// pgx sends Exec without arguments as a simple Query, so each statement under test takes arguments.
	// QueryExecModeExec sends them as an unnamed Parse/Bind/Execute, QueryExecModeSimpleProtocol as one Query.
	modes := []struct {
		name string
		mode pgx.QueryExecMode
	}{
		{"simple", pgx.QueryExecModeSimpleProtocol},
		{"extended", pgx.QueryExecModeExec},
	}
	begins := []struct {
		name string
		sql  []string
	}{
		{"begin read only", []string{"BEGIN READ ONLY"}},
		{"set transaction read only", []string{"BEGIN", "SET TRANSACTION READ ONLY"}},
	}
	for _, b := range begins {
		for _, m := range modes {
			t.Run(b.name+"/"+m.name, func(t *testing.T) {
				t.Parallel()
				ctx := tctx(t)
				conn, err := pgx.Connect(ctx, connURL(hybridAddr))
				if err != nil {
					t.Fatal(err)
				}
				defer conn.Close(ctx)

				for _, sql := range b.sql {
					if _, err := conn.Exec(ctx, sql); err != nil {
						t.Fatalf("%s: %v", sql, err)
					}
				}

				_, err = conn.Exec(ctx, "INSERT INTO users (username, email) VALUES ($1, $2)", m.mode, "read_only", "read_only@example.com")
				var pgErr *pgconn.PgError
				if !errors.As(err, &pgErr) || pgErr.Severity != "ERROR" || pgErr.Code != "25006" {
					t.Fatalf("write in read only transaction: got %v, want ERROR 25006", err)
				}

				var n int
				err = conn.QueryRow(ctx, "SELECT $1::int", m.mode, 1).Scan(&n)
				if !errors.As(err, &pgErr) || pgErr.Code != "25P02" {
					t.Fatalf("query in failed transaction: got %v, want ERROR 25P02", err)
				}

				if _, err := conn.Exec(ctx, "ROLLBACK"); err != nil {
					t.Fatalf("rollback: %v", err)
				}

				var readOnly string
				if err := conn.QueryRow(ctx, "SELECT current_setting($1)", m.mode, "transaction_read_only").Scan(&readOnly); err != nil {
					t.Fatalf("after rollback: %v", err)
				}
				if readOnly != "off" {
					t.Errorf("transaction_read_only after rollback = %q, want off", readOnly)
				}
			})
		}
	}
}
