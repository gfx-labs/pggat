# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

pggat is a PostgreSQL connection pooler similar to PgBouncer, with the key differentiator being built-in support for load balancing across read-write and read-only replica nodes. It's built on top of the Caddy web server framework, leveraging Caddy's module system, configuration management, and lifecycle handling.

**Repository**: `github.com/gfx-labs/pggat`
**Language**: Go 1.19+ (go.mod specifies 1.19)
**Build System**: Go modules
**Architecture**: Modular, based on Caddy v2

## Core Architecture

### Key Components

pggat's architecture consists of several layers:

1. **Caddy Integration Layer** (`cmd/`, `lib/gat/`)
   - pggat is implemented as a Caddy application module
   - Uses Caddy's configuration adapters, module system, and lifecycle management
   - The main entry point is `cmd/pggat/main.go` which imports the Caddy command framework

2. **Protocol Layer** (`lib/fed/`)
   - `fed` (Frontend/Backend) package implements PostgreSQL wire protocol v3.0
   - `fed.Conn` wraps connections with middleware support for packet interception
   - Middleware pattern allows features like prepared statement tracking, parameter synchronization, and tracing
   - Key middlewares:
     - `eqp` (Extended Query Protocol): Tracks prepared statements
     - `ps` (Parameter Status): Synchronizes session parameters between client and server
     - `unterminate`: Prevents connection termination to enable pooling

3. **Pooling Layer** (`lib/bouncer/`, `lib/gat/handlers/pool/`)
   - Two pooling modes:
     - **Transaction Pooling** (default): Each transaction goes to a new node, supports protocol-level prepared statements
     - **Session Pooling**: Each session goes to a node, full PostgreSQL feature support
   - Two pool implementations:
     - `basic`: Single pool for all connections
     - `hybrid`: Separate pools for read-write (primary) and read-only (replica) nodes
   - Connection scheduling via `lib/rob/` (Rob scheduler - round-robin with penalties)

4. **Load Balancing** (`lib/gat/handlers/pool/spool/`)
   - `spool.Pool` manages server connections with auto-scaling
   - Recipe-based connection management with min/max connections per recipe
   - Support for penalty-based routing (latency, replication lag)
   - Critics evaluate server health and apply penalties

5. **Configuration** (`lib/gat/gatcaddyfile/`)
   - Custom Caddyfile adapter called "Gatfile"
   - Supports both Caddyfile syntax and native JSON configuration
   - Default config file name: `Gatfile` (searched in current directory)

6. **Authentication** (`lib/auth/`)
   - Supports: Cleartext, MD5, SASL-SCRAM-SHA-256
   - Credential management and encoding/decoding for auth flows

7. **Discovery** (`lib/gat/handlers/discovery/`)
   - Dynamic discovery of PostgreSQL nodes
   - Providers: AWS RDS, Google Cloud SQL

### Critical Implementation Details

#### Middleware Chain

The `fed.Conn` middleware chain is bidirectional and processes packets in both directions:
- **Read path**: PreRead -> ReadPacket (forwards through middleware stack)
- **Write path**: WritePacket (backwards through middleware stack) -> PostWrite

This allows middleware to intercept, modify, or generate packets on both client and server sides.

#### Connection Pairing

In transaction pooling mode:
1. Client connects and authenticates with pggat
2. For each transaction, pggat:
   - Acquires a server connection from the pool
   - Synchronizes session parameters (tracked parameters only)
   - Routes transaction packets
   - Returns server to pool when transaction ends (COMMIT/ROLLBACK)

In session pooling mode:
1. Client connects and authenticates
2. pggat acquires a dedicated server connection
3. All client traffic routes to that server until disconnect

#### Parameter Synchronization

pggat tracks "important" session parameters (configurable via `TrackedParameters`):
- When switching servers (transaction pooling), only tracked parameters are synced
- Client receives `ParameterStatus` messages for all parameters it cares about
- Server is SET for tracked parameters that differ from client expectations
- This balances feature support with pooling efficiency

#### Prepared Statements

Transaction pooling mode supports protocol-level prepared statements via the `eqp` middleware:
- Statements are re-prepared on each new server connection
- Statement handles are tracked and mapped
- Clients see consistent statement names across server switches

#### Code Generation

PostgreSQL packet types are code-generated:
- Source: `hack/packetgen/main.go`
- Output: `lib/fed/packets/v3.0/packets.go`
- Do not manually edit generated files
- Regenerate with: `go run hack/packetgen/main.go`

## Development Workflow

### Building

```bash
# Build the binary
go build -o pggat ./cmd/pggat

# Using Docker
docker build -t pggat .
```

### Testing

```bash
# Run all tests (requires PostgreSQL on localhost:5432)
go test ./...

# Run with race detector
CGO_ENABLED=1 go test -race ./...

# Coverage
go test -coverprofile=coverage.txt -covermode count ./...
go tool cover -func=coverage.txt
```

**Test Requirements**:
- PostgreSQL must be running on localhost:5432
- Default credentials: postgres/postgres
- Tests use environment variables: POSTGRES_HOST, POSTGRES_PORT, POSTGRES_PASSWORD

### Linting

```bash
# Run golangci-lint
golangci-lint run --timeout=15m

# Fix issues automatically
golangci-lint run --fix

# CI uses golangci-lint v2.5.0
```

**Important Linter Notes**:
- G115 (integer overflow) is disabled - intentional for PostgreSQL wire protocol compatibility
- Several gocritic checks are disabled (see `.golangci.yml`)
- Max cyclomatic complexity: 30
- Test functions and hack/ directory are excluded

### Running Locally

```bash
# Run with a Gatfile in current directory
go run ./cmd/pggat run

# Run with specific config
go run ./cmd/pggat run --config myconfig.Gatfile

# Run with config watching (auto-reload on changes)
go run ./cmd/pggat run --config myconfig.Gatfile --watch

# Validate config
go run ./cmd/pggat validate --config myconfig.Gatfile

# Convert Gatfile to JSON
go run ./cmd/pggat adapt --config myconfig.Gatfile --pretty
```

### Configuration Example

Example minimal Gatfile:
```caddyfile
:5433 {
    ssl self_signed

    pool /mydb {
        pool basic session

        address localhost:5432
        username postgres
        password postgres
        database mydb
    }
}
```

## Important Code Patterns

### Error Handling

pggat uses `lib/perror` for PostgreSQL-compatible error handling:
- Errors can be converted to PostgreSQL ErrorResponse packets
- PostgreSQL error codes are defined in `lib/perror/code.go`
- Use `perror.New()` to create errors that can be sent to clients

### Concurrency

- Most pools use `sync.RWMutex` for concurrent access
- Server connections are managed by goroutines
- Client connections each run in their own goroutine
- The Rob scheduler uses lock-free operations where possible

### Memory Management

- Custom slice utilities in `lib/util/slices/` for efficient resizing
- Buffer pooling in `lib/util/pools/` for packet handling
- Ring buffers in `lib/util/ring/` for connection queues

### Tracing

OpenTelemetry integration:
- Traces span client connections, server connections, and transactions
- Configured via environment variables
- Middleware `lib/fed/middlewares/tracing/` adds packet-level tracing

## Module Structure

```
lib/
├── auth/               # Authentication mechanisms
├── bouncer/           # Core bouncing logic (client<->server proxying)
├── fed/               # PostgreSQL protocol implementation
│   ├── codecs/        # Wire protocol encoding/decoding
│   ├── listeners/     # Connection listeners
│   ├── middlewares/   # Packet middleware (ps, eqp, tracing, etc.)
│   └── packets/       # Packet type definitions (GENERATED)
├── gat/               # Caddy integration and main app
│   ├── gatcaddyfile/  # Gatfile configuration adapter
│   ├── handlers/      # Route handlers (pool, discovery, rewrite, etc.)
│   ├── matchers/      # Request matchers (user, database, SSL, etc.)
│   ├── metrics/       # Prometheus metrics
│   └── ssl/           # TLS/SSL servers and clients
├── gsql/              # SQL parsing utilities
├── instrumentation/   # Prometheus instrumentation
├── perror/            # PostgreSQL error handling
├── rob/               # Round-robin scheduler with penalties
└── util/              # Utilities (maps, slices, strings, etc.)
```

## CI/CD

GitHub Actions workflow (`.github/workflows/ci.yml`):
- **test**: Run tests with race detector on Go 1.23
- **lint**: golangci-lint v2.5.0
- **coverage**: Generate coverage reports, upload to Codecov
- **build**: Build verification

GitLab CI (`.gitlab-ci.yml`) provides similar functionality with test, lint, and coverage stages.

## Special Notes

### Comments with "okay" or TODOs

Several files contain comments noting things are "okay" - these indicate intentional behavior that might look suspicious to linters or seem like errors but are working as intended.

Known TODOs:
- `lib/bouncer/backends/v0/accept.go`: Handle notice responses
- `lib/gat/ssl/servers/self_signed/server.go`: SSL server improvements
- `lib/gat/handlers/pgbouncer/module.go`: Remove InsecureSkipVerify hardcode

### Caddy Integration Quirks

pggat inherits Caddy's command structure but customizes help text to say "Pggat" instead of "Caddy". This is done in the command registration (see `cmd/commands.go`).

### Protocol Compatibility

- Only PostgreSQL wire protocol v3.0 is supported
- LISTEN/NOTIFY is not properly supported in transaction pooling mode
- Statement pooling is intentionally not implemented (negligible benefit, major compatibility issues)

### Cloud Provider Integration

Discovery handlers integrate with cloud provider APIs to automatically find PostgreSQL instances:
- **AWS RDS**: Automatic instance discovery via AWS API
- **Google Cloud SQL**: Uses project/instance discovery

### Permissions

The `.claude/settings.local.json` file shows allowed Bash commands for AI assistants. When working in this codebase:
- Go build/run/mod commands are pre-approved
- golangci-lint operations are pre-approved
- File operations should use dedicated tools (Read/Write/Edit) not Bash

## Common Tasks

### Adding a New Pool Type

1. Create package in `lib/gat/handlers/pool/pools/yourtype/`
2. Implement `pool.Pool` interface
3. Register factory in `lib/gat/handlers/pool/module.go`
4. Add Gatfile unmarshaller in `lib/gat/gatcaddyfile/pool.go`

### Adding a New Matcher

1. Create package in `lib/gat/matchers/yourmatcher.go`
2. Implement `gat.Matcher` interface
3. Register in Caddy module system
4. Add Gatfile unmarshaller in `lib/gat/gatcaddyfile/matcher.go`

### Adding Prometheus Metrics

1. Define metrics in `lib/instrumentation/prom/`
2. Hook into appropriate pool/handler in `lib/gat/metrics/`
3. Metrics automatically exported at `/metrics` endpoint

### Modifying Packet Types

1. Edit packet definitions in `hack/packetgen/main.go`
2. Run: `go run hack/packetgen/main.go`
3. Review generated code in `lib/fed/packets/v3.0/packets.go`
4. Never manually edit generated files

## Dependencies

Key external dependencies:
- `github.com/caddyserver/caddy/v2`: Core framework
- `github.com/jackc/pgx/v4`: PostgreSQL driver for testing
- `go.opentelemetry.io/otel`: Distributed tracing
- `github.com/prometheus/client_golang`: Metrics
- `k8s.io/client-go`: Kubernetes integration for discovery

Custom dependencies (gfx.cafe):
- `gfx.cafe/ghalliday1/scram`: SCRAM authentication
- `gfx.cafe/open/gotoprom`: Prometheus helpers
- `gfx.cafe/util/go`: General utilities

## Performance Considerations

- Transaction pooling is more efficient but has limitations (no session state)
- Session pooling has full compatibility but less connection reuse
- Connection limits are per-recipe (min/max connections)
- Rob scheduler applies penalties to slow/lagging servers automatically
- Prepared statements add overhead in transaction pooling (re-prepare on each server switch)

## Debugging Tips

1. Enable stat logging: Set `stat_log_period` in Gatfile global options
2. Use OpenTelemetry tracing: Configure OTEL environment variables
3. Check Prometheus metrics: Hit `/metrics` endpoint
4. Validate config: `pggat validate --config yourconfig.Gatfile`
5. Pretty-print adapted JSON: `pggat adapt --config yourconfig.Gatfile --pretty`
6. Watch mode for development: `pggat run --config yourconfig.Gatfile --watch`
