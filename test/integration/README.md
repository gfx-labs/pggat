# Integration Tests

End-to-end tests that run real clients (pgx v5) through pggat against a real PostgreSQL.

Everything runs in-process. No Docker is needed:
- PostgreSQL is started with [embedded-postgres](https://github.com/fergusstrange/embedded-postgres)
  and seeded from `test/fixtures/init-primary.sql`.
- pggat is started from the Gatfiles in `test/configs/` (transaction, session, hybrid).
  Each Gatfile is rewritten to point at the embedded server and listen on a free port.

## Running

```bash
make integration
# or
go test -race -tags integration ./test/integration/...
```

Run a single test:

```bash
go test -tags integration ./test/integration -run TestTransactionPooling
```

Skip the stress tests:

```bash
go test -tags integration -short ./test/integration/...
```

Benchmarks:

```bash
go test -tags integration ./test/integration -run '^$' -bench .
```

## Postgres binaries

The first run downloads PostgreSQL, extracts it, and runs `initdb` once into
`~/.embedded-postgres-go` (override with `PGTEST_CACHE`). Later runs copy that
template data directory, so startup takes well under a second.

Set `PGTEST_LOG=1` to print PostgreSQL server output.

The embedded server is PostgreSQL 18. Only PostgreSQL 18 and later send
`ParameterStatus` for `search_path`. On earlier versions pggat is never told about
`SET search_path` at runtime, so transaction pooling cannot carry it to another
server connection. The `search_path` tests (`search_path_test.go`) need 18+ and
would fail on 17. Startup `search_path` values do not depend on this.

## Writing tests

Tests connect with `connURL(addr)` where `addr` is one of:

| Variable          | Target                          |
|-------------------|---------------------------------|
| `transactionAddr` | pggat, `transaction.Gatfile`    |
| `sessionAddr`     | pggat, `session.Gatfile`        |
| `hybridAddr`      | pggat, `hybrid.Gatfile`         |
| `primaryAddr`     | PostgreSQL directly             |

```go
func TestMyFeature(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, connURL(transactionAddr))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer conn.Close(ctx)
	// ...
}
```

Tests run in parallel and share one database, so write only inside transactions
that are rolled back, or use unique names.

Other packages can start their own server with `pgtest.StartT(t, "dbname")`.
