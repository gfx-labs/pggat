//go:build integration

package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// TestTransactionPooling tests basic transaction pooling functionality
func TestTransactionPooling(t *testing.T) {
	t.Parallel()
	connString := connURL(transactionAddr)

	conn, err := pgx.Connect(context.Background(), connString)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close(context.Background())

	// Test simple query
	var result int
	err = conn.QueryRow(context.Background(), "SELECT 1+1").Scan(&result)
	if err != nil {
		t.Errorf("Query failed: %v", err)
	}
	if result != 2 {
		t.Errorf("Expected 2, got %d", result)
	}

	// Test transaction
	tx, err := conn.Begin(context.Background())
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	var count int64
	err = tx.QueryRow(context.Background(), "SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		t.Errorf("Query in transaction failed: %v", err)
	}
	if count < 1 {
		t.Errorf("Expected at least 1 user, got %d", count)
	}

	err = tx.Commit(context.Background())
	if err != nil {
		t.Errorf("Failed to commit transaction: %v", err)
	}
}

// TestSessionPooling tests session pooling with session state
func TestSessionPooling(t *testing.T) {
	t.Parallel()
	connString := connURL(sessionAddr)

	conn, err := pgx.Connect(context.Background(), connString)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close(context.Background())

	// Set a session variable
	_, err = conn.Exec(context.Background(), "SET application_name = 'integration_test'")
	if err != nil {
		t.Fatalf("Failed to set session variable: %v", err)
	}

	// Verify it persists
	var appName string
	err = conn.QueryRow(context.Background(), "SHOW application_name").Scan(&appName)
	if err != nil {
		t.Errorf("Failed to read session variable: %v", err)
	}
	if appName != "integration_test" {
		t.Errorf("Expected 'integration_test', got '%s'", appName)
	}

	// Test prepared statements
	_, err = conn.Prepare(context.Background(), "get_user", "SELECT username FROM users WHERE id = $1")
	if err != nil {
		t.Fatalf("Failed to prepare statement: %v", err)
	}

	var username string
	err = conn.QueryRow(context.Background(), "get_user", 1).Scan(&username)
	if err != nil {
		t.Errorf("Failed to execute prepared statement: %v", err)
	}
	if username == "" {
		t.Errorf("Expected non-empty username")
	}
}

// TestConcurrentConnections tests multiple concurrent connections
func TestConcurrentConnections(t *testing.T) {
	t.Parallel()
	connString := connURL(transactionAddr)

	numConns := 10
	errors := make(chan error, numConns)

	for i := 0; i < numConns; i++ {
		go func(id int) {
			conn, err := pgx.Connect(context.Background(), connString)
			if err != nil {
				errors <- fmt.Errorf("connection %d failed: %w", id, err)
				return
			}
			defer conn.Close(context.Background())

			var result int
			err = conn.QueryRow(context.Background(), "SELECT $1::int", id).Scan(&result)
			if err != nil {
				errors <- fmt.Errorf("query %d failed: %w", id, err)
				return
			}
			if result != id {
				errors <- fmt.Errorf("connection %d: expected %d, got %d", id, id, result)
				return
			}

			errors <- nil
		}(i)
	}

	// Collect results
	for i := 0; i < numConns; i++ {
		select {
		case err := <-errors:
			if err != nil {
				t.Error(err)
			}
		case <-time.After(10 * time.Second):
			t.Fatalf("Timeout waiting for connection %d", i)
		}
	}
}

// TestResetQueryTimeout tests that reset query timeout works correctly
func TestResetQueryTimeout(t *testing.T) {
	t.Parallel()
	connString := connURL(sessionAddr)

	conn, err := pgx.Connect(context.Background(), connString)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close(context.Background())

	// Execute a query
	var result int
	err = conn.QueryRow(context.Background(), "SELECT 1").Scan(&result)
	if err != nil {
		t.Errorf("Query failed: %v", err)
	}

	// Close connection (should trigger reset query)
	err = conn.Close(context.Background())
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Reopen and verify connection pool is working
	conn2, err := pgx.Connect(context.Background(), connString)
	if err != nil {
		t.Fatalf("Failed to reconnect: %v", err)
	}
	defer conn2.Close(context.Background())

	err = conn2.QueryRow(context.Background(), "SELECT 2").Scan(&result)
	if err != nil {
		t.Errorf("Query after reconnect failed: %v", err)
	}
	if result != 2 {
		t.Errorf("Expected 2, got %d", result)
	}
}

// TestLongRunningQuery tests handling of long queries
func TestLongRunningQuery(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping long-running test in short mode")
	}

	connString := connURL(transactionAddr)

	conn, err := pgx.Connect(context.Background(), connString)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close(context.Background())

	// Run a query that takes 1 second
	start := time.Now()
	var result bool
	err = conn.QueryRow(context.Background(), "SELECT true FROM pg_sleep(1)").Scan(&result)
	duration := time.Since(start)

	if err != nil {
		t.Errorf("Long query failed: %v", err)
	}
	if duration < time.Second {
		t.Errorf("Query finished too quickly: %v", duration)
	}
	if !result {
		t.Error("Expected true result")
	}
}

// TestDirectPostgres tests direct connection to PostgreSQL (baseline)
func TestDirectPostgres(t *testing.T) {
	t.Parallel()
	connString := connURL(primaryAddr)

	conn, err := pgx.Connect(context.Background(), connString)
	if err != nil {
		t.Fatalf("Failed to connect directly to postgres: %v", err)
	}
	defer conn.Close(context.Background())

	var version string
	err = conn.QueryRow(context.Background(), "SELECT version()").Scan(&version)
	if err != nil {
		t.Errorf("Query failed: %v", err)
	}
	t.Logf("PostgreSQL version: %s", version)
}

// TestHybridPooling tests reads and writes through the hybrid pool
func TestHybridPooling(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, connURL(hybridAddr))
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close(ctx)

	var count int64
	if err := conn.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if count < 1 {
		t.Errorf("Expected at least 1 user, got %d", count)
	}

	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, "INSERT INTO posts (user_id, title) VALUES (1, 'hybrid')"); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
}
