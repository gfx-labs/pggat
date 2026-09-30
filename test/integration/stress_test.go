//go:build integration

package integration

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestHighConcurrency tests many concurrent connections and queries
func TestHighConcurrency(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	connString := connURL(transactionAddr, "pool_max_conns=50")

	pool, err := pgxpool.New(context.Background(), connString)
	if err != nil {
		t.Fatalf("Failed to create connection pool: %v", err)
	}
	defer pool.Close()

	numWorkers := 100
	queriesPerWorker := 100
	var successCount atomic.Int64
	var errorCount atomic.Int64
	var wg sync.WaitGroup

	start := time.Now()

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < queriesPerWorker; j++ {
				var result int
				err := pool.QueryRow(
					context.Background(),
					"SELECT COUNT(*) FROM posts WHERE user_id = $1",
					(workerID%3)+1,
				).Scan(&result)

				if err != nil {
					errorCount.Add(1)
					t.Logf("Worker %d query %d failed: %v", workerID, j, err)
				} else {
					successCount.Add(1)
				}
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	totalQueries := int64(numWorkers * queriesPerWorker)
	successRate := float64(successCount.Load()) / float64(totalQueries) * 100
	qps := float64(totalQueries) / duration.Seconds()

	t.Logf("Duration: %v", duration)
	t.Logf("Total queries: %d", totalQueries)
	t.Logf("Successful: %d", successCount.Load())
	t.Logf("Errors: %d", errorCount.Load())
	t.Logf("Success rate: %.2f%%", successRate)
	t.Logf("Queries per second: %.2f", qps)

	if successRate < 95.0 {
		t.Errorf("Success rate too low: %.2f%%", successRate)
	}
}

// TestTransactionStress tests many concurrent transactions
func TestTransactionStress(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	connString := connURL(transactionAddr)

	pool, err := pgxpool.New(context.Background(), connString)
	if err != nil {
		t.Fatalf("Failed to create connection pool: %v", err)
	}
	defer pool.Close()

	numTransactions := 100
	var successCount atomic.Int64
	var errorCount atomic.Int64
	var wg sync.WaitGroup

	for i := 0; i < numTransactions; i++ {
		wg.Add(1)
		go func(txID int) {
			defer wg.Done()

			ctx := context.Background()
			tx, err := pool.Begin(ctx)
			if err != nil {
				errorCount.Add(1)
				t.Logf("Failed to begin transaction %d: %v", txID, err)
				return
			}
			defer tx.Rollback(ctx) // Safe to call even after commit

			// Do some work in transaction
			var count int
			err = tx.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
			if err != nil {
				errorCount.Add(1)
				t.Logf("Transaction %d query failed: %v", txID, err)
				return
			}

			// Simulate some processing time
			time.Sleep(10 * time.Millisecond)

			err = tx.Commit(ctx)
			if err != nil {
				errorCount.Add(1)
				t.Logf("Transaction %d commit failed: %v", txID, err)
				return
			}

			successCount.Add(1)
		}(i)
	}

	wg.Wait()

	successRate := float64(successCount.Load()) / float64(numTransactions) * 100
	t.Logf("Transactions: %d, Success: %d, Errors: %d, Success rate: %.2f%%",
		numTransactions, successCount.Load(), errorCount.Load(), successRate)

	if successRate < 95.0 {
		t.Errorf("Transaction success rate too low: %.2f%%", successRate)
	}
}

// TestConnectionChurn tests rapid connection open/close cycles
func TestConnectionChurn(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	connString := connURL(transactionAddr)

	numCycles := 100
	var errorCount atomic.Int64
	var wg sync.WaitGroup

	for i := 0; i < numCycles; i++ {
		wg.Add(1)
		go func(cycleID int) {
			defer wg.Done()

			conn, err := pgx.Connect(context.Background(), connString)
			if err != nil {
				errorCount.Add(1)
				t.Logf("Cycle %d: Failed to connect: %v", cycleID, err)
				return
			}

			var result int
			err = conn.QueryRow(context.Background(), "SELECT 1").Scan(&result)
			if err != nil {
				errorCount.Add(1)
				t.Logf("Cycle %d: Query failed: %v", cycleID, err)
			}

			err = conn.Close(context.Background())
			if err != nil {
				errorCount.Add(1)
				t.Logf("Cycle %d: Close failed: %v", cycleID, err)
			}
		}(i)
	}

	wg.Wait()

	if errorCount.Load() > 0 {
		t.Errorf("Encountered %d errors during connection churn", errorCount.Load())
	}
}

// TestPreparedStatementStress tests many prepared statements
func TestPreparedStatementStress(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	connString := connURL(sessionAddr)

	conn, err := pgx.Connect(context.Background(), connString)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close(context.Background())

	// Prepare multiple statements
	numStatements := 20
	for i := 0; i < numStatements; i++ {
		stmtName := fmt.Sprintf("stmt_%d", i)
		_, err := conn.Prepare(
			context.Background(),
			stmtName,
			"SELECT $1::int + $2::int",
		)
		if err != nil {
			t.Fatalf("Failed to prepare statement %s: %v", stmtName, err)
		}
	}

	// Execute them many times
	numExecutions := 1000
	var errorCount int
	for i := 0; i < numExecutions; i++ {
		stmtName := fmt.Sprintf("stmt_%d", i%numStatements)
		var result int
		err := conn.QueryRow(
			context.Background(),
			stmtName,
			i, i+1,
		).Scan(&result)
		if err != nil {
			errorCount++
			t.Logf("Execution %d failed: %v", i, err)
		} else if result != 2*i+1 {
			t.Errorf("Execution %d: expected %d, got %d", i, 2*i+1, result)
		}
	}

	if errorCount > 0 {
		t.Errorf("Encountered %d errors during prepared statement execution", errorCount)
	}
}

// TestLongTransaction tests a transaction that takes a while
func TestLongTransaction(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping long test in short mode")
	}

	connString := connURL(transactionAddr)

	conn, err := pgx.Connect(context.Background(), connString)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close(context.Background())

	tx, err := conn.Begin(context.Background())
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	// Do multiple queries within the transaction over time
	for i := 0; i < 5; i++ {
		var count int64
		err = tx.QueryRow(context.Background(), "SELECT COUNT(*) FROM users").Scan(&count)
		if err != nil {
			t.Errorf("Query %d failed: %v", i, err)
			tx.Rollback(context.Background())
			return
		}
		t.Logf("Query %d: user count = %d", i, count)
		time.Sleep(100 * time.Millisecond)
	}

	err = tx.Commit(context.Background())
	if err != nil {
		t.Errorf("Failed to commit long transaction: %v", err)
	}
}

// BenchmarkSimpleQuery benchmarks simple query performance
func BenchmarkSimpleQuery(b *testing.B) {
	connString := connURL(transactionAddr)

	pool, err := pgxpool.New(context.Background(), connString)
	if err != nil {
		b.Fatalf("Failed to create connection pool: %v", err)
	}
	defer pool.Close()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var result int
		err := pool.QueryRow(context.Background(), "SELECT 1").Scan(&result)
		if err != nil {
			b.Fatalf("Query failed: %v", err)
		}
	}
}

// BenchmarkTransaction benchmarks transaction performance
func BenchmarkTransaction(b *testing.B) {
	connString := connURL(transactionAddr)

	pool, err := pgxpool.New(context.Background(), connString)
	if err != nil {
		b.Fatalf("Failed to create connection pool: %v", err)
	}
	defer pool.Close()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		tx, err := pool.Begin(context.Background())
		if err != nil {
			b.Fatalf("Failed to begin transaction: %v", err)
		}

		var count int
		err = tx.QueryRow(context.Background(), "SELECT COUNT(*) FROM users").Scan(&count)
		if err != nil {
			b.Fatalf("Query failed: %v", err)
		}

		err = tx.Commit(context.Background())
		if err != nil {
			b.Fatalf("Commit failed: %v", err)
		}
	}
}
