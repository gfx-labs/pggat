# Integration Tests

This directory contains integration tests for pggat with real PostgreSQL databases.

## Overview

The integration test suite sets up a complete testing environment with:
- **PostgreSQL primary database** (port 5432)
- **PostgreSQL replica database** (port 5433) - simulates read replica
- **Pggat with transaction pooling** (port 6432)
- **Pggat with session pooling** (port 6433)
- **Pggat with hybrid pooling** (port 6434) - primary + replica

## Quick Start

Run all integration tests:
```bash
make integration
```

This will:
1. Build pggat Docker image
2. Start PostgreSQL primary and replica
3. Start three pggat instances with different configurations
4. Run the integration test suite
5. Clean up all containers

## Development Workflow

### Start Services Only (for manual testing)

```bash
# Start all services in background
make integration-up

# Check logs
make integration-logs

# When done, tear down
make integration-down
```

### Run Tests Against Running Services

```bash
# Start services
make integration-up

# Run tests manually
go test -v ./test/integration/...

# Or run specific tests
go test -v ./test/integration -run TestTransactionPooling

# Or run benchmarks
go test -v ./test/integration -bench=. -benchtime=10s

# Tear down when done
make integration-down
```

### Interactive Shell

Get a shell in the test container to debug:
```bash
make integration-shell
```

From the shell you can:
```bash
# Run all tests
go test -v ./test/integration/...

# Run specific test
go test -v ./test/integration -run TestHighConcurrency

# Run with short mode (skip long tests)
go test -v -short ./test/integration/...

# Connect to databases directly
psql postgres://postgres:postgres@postgres-primary:5432/testdb
psql postgres://postgres:postgres@pggat-transaction:6432/testdb
```

## Test Structure

### Basic Tests (`basic_test.go`)

Core functionality tests:
- `TestTransactionPooling` - Basic transaction pooling
- `TestSessionPooling` - Session pooling with state
- `TestConcurrentConnections` - Multiple concurrent connections
- `TestResetQueryTimeout` - Reset query timeout handling
- `TestLongRunningQuery` - Long-running query handling
- `TestDirectPostgres` - Direct PostgreSQL baseline

### Stress Tests (`stress_test.go`)

Performance and load tests:
- `TestHighConcurrency` - 100 workers x 100 queries
- `TestTransactionStress` - 100 concurrent transactions
- `TestConnectionChurn` - Rapid connect/disconnect cycles
- `TestPreparedStatementStress` - Many prepared statements
- `TestLongTransaction` - Multi-second transaction
- `BenchmarkSimpleQuery` - Query performance benchmark
- `BenchmarkTransaction` - Transaction performance benchmark

Run only stress tests:
```bash
go test -v ./test/integration -run Stress
```

Skip stress tests (short mode):
```bash
go test -v -short ./test/integration/...
```

## Configuration Files

Test configurations are in `test/configs/`:

### `transaction.Gatfile`
- Basic pool with transaction pooling
- Min 2 connections, max 20
- 10s acquire timeout
- 60s idle timeout

### `session.Gatfile`
- Basic pool with session pooling
- `DISCARD ALL` reset query with 15s timeout
- Min 2 connections, max 20
- 10s acquire timeout
- 300s idle timeout (5 minutes)

### `hybrid.Gatfile`
- Hybrid pool with separate primary and replica pools
- Each pool: min 2, max 10 connections
- Session pooling mode
- `DISCARD ALL` reset query with 15s timeout

## Database Fixtures

Test databases are initialized with fixtures in `test/fixtures/`:

### Schema
- `users` table (id, username, email, created_at)
- `posts` table (id, user_id, title, content, created_at)
- `sessions` table (id, user_id, data, expires_at)

### Test Data
- 3 users: alice, bob, charlie
- 4 posts from various users
- Test functions for advisory locks

## Adding New Tests

1. Create test file in `test/integration/`:
```go
package integration

import (
    "context"
    "testing"
    "github.com/jackc/pgx/v4"
)

func TestMyFeature(t *testing.T) {
    connString := fmt.Sprintf(
        "postgres://%s:%s@%s:6432/testdb?sslmode=disable",
        postgresUser, postgresPassword, pggatTransactionHost,
    )

    conn, err := pgx.Connect(context.Background(), connString)
    if err != nil {
        t.Fatalf("Failed to connect: %v", err)
    }
    defer conn.Close(context.Background())

    // Your test logic here
}
```

2. For long-running tests, check short mode:
```go
func TestLongRunning(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping long test in short mode")
    }
    // Test logic
}
```

3. Run your new test:
```bash
make integration-up
go test -v ./test/integration -run TestMyFeature
```

## Environment Variables

Tests use these environment variables (set automatically by docker-compose):
- `POSTGRES_PASSWORD` - Postgres password (default: postgres)
- `POSTGRES_USER` - Postgres user (default: postgres)
- `PGGAT_TRANSACTION_HOST` - Transaction pooling host
- `PGGAT_SESSION_HOST` - Session pooling host
- `PGGAT_HYBRID_HOST` - Hybrid pooling host
- `POSTGRES_PRIMARY_HOST` - Primary database host
- `POSTGRES_REPLICA_HOST` - Replica database host

## Connecting to Services Manually

While services are running (`make integration-up`):

### Direct PostgreSQL
```bash
psql postgres://postgres:postgres@localhost:5432/testdb
```

### Via Pggat Transaction Pool
```bash
psql postgres://postgres:postgres@localhost:6432/testdb
```

### Via Pggat Session Pool
```bash
psql postgres://postgres:postgres@localhost:6433/testdb
```

### Via Pggat Hybrid Pool
```bash
psql postgres://postgres:postgres@localhost:6434/testdb
```

## Troubleshooting

### Services won't start
```bash
# Check logs
make integration-logs

# Or check specific service
docker compose -f docker-compose.integration.yml logs pggat-transaction
```

### Tests hang or timeout
- Check if services are healthy: `docker compose -f docker-compose.integration.yml ps`
- Increase timeouts in test code
- Check pggat logs for errors

### Connection refused errors
- Services might not be fully started
- Wait a few seconds and retry
- Check healthcheck status: `docker compose -f docker-compose.integration.yml ps`

### Clean slate
```bash
# Remove everything and start fresh
make integration-down
docker system prune -f
make integration-up
```

## CI/CD Integration

To run integration tests in CI:

```yaml
# Example GitHub Actions
- name: Run integration tests
  run: |
    make integration
```

The test suite exits with code 0 on success, non-zero on failure.

## Performance Expectations

Typical performance metrics from stress tests:

- **High Concurrency**: 10,000 queries in ~10-15 seconds (600-1000 QPS)
- **Transaction Stress**: 100 transactions in ~2-3 seconds
- **Connection Churn**: 100 cycles in ~2-3 seconds
- **Simple Query Benchmark**: ~2000-5000 ops/sec
- **Transaction Benchmark**: ~1000-2000 ops/sec

Actual numbers depend on hardware and system load.
