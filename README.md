# pggat

![image](https://github.com/user-attachments/assets/a4e7881f-fc5a-4349-b141-3148aff0f09f)

pggat is a Postgres pooler similar to PgBouncer. Its primary difference in functionality is that it supports load balancing to rdwr/rd replicas. This allows for pseudo-horizontal scaling via balancing to read-replicas.

the name comes from [this song, as it is a banger](https://www.youtube.com/watch?v=-DqCc2DJ0sg)

## Architecture

pggat is built on the [Caddy](https://caddyserver.com/) framework, leveraging its module system, configuration management, and runtime capabilities. This provides pggat with:

- A Robust plugin architecture through Caddy modules
- Familiar Caddyfile configuration format (adapted as Gatfile)
- A possible path to integration with Caddy down the line

## Features

### Connection Pooling
- Transaction pooling mode with prepared statement support
- Session pooling mode for full feature compatibility
- Basic and hybrid pooling implementations
- Connection warm-up and idle management
- Automatic reconnection with exponential backoff

### Load Balancing
- Primary/replica routing
- Read/write splitting
- Query latency-based routing
- Replication lag-aware routing
- Parameter-based routing decisions

### Authentication
- Plaintext password authentication
- MD5 password authentication
- SCRAM-SHA-256 authentication
- Client certificate authentication
- Pass-through authentication modes

### SSL/TLS
- Self-signed certificate generation
- X.509 certificate support
- Client certificate verification
- Optional SSL enforcement
- Configurable TLS modes

### Service Discovery
- CloudNativePG operator integration
- Zalando Postgres Operator support
- Google Cloud SQL discovery
- DigitalOcean managed database discovery
- Static configuration support
- Dynamic cluster updates

### Protocol Support
- PostgreSQL wire protocol v3.0
- Extended query protocol
- Simple query protocol
- COPY protocol support
- Prepared statements
- Parameter status synchronization

### Monitoring & Observability
- OpenTelemetry tracing integration
- Prometheus metrics export
- Query performance tracking
- Connection statistics
- Packet-level tracing

### Request Routing
- Database-based routing
- User-based routing
- SSL requirement matching
- Local address matching
- Startup parameter matching
- Boolean logic combinators (AND, OR, NOT)

### Request Modification
- Database rewriting
- Username rewriting
- Password rewriting
- Parameter rewriting
- Error message handling

### Configuration
- Gatfile configuration format (Using Caddyfile internals)
- Runtime configuration reloading (Via Caddy)
- Environment variable support
- Command-line interface

## Pooling Modes
There are currently two pooling modes which compromise between balancing and feature support. Most apps should work out of the box with transaction pooling.

### Transaction Pooling (default)
Send each transaction to a new node. This mode supports all postgres features that do not rely on session state (plus a few exceptions noted below).

This is similar to PgBouncer's transaction pooling except we additionally support protocol level prepared statements and all parameters (they may change at unexpected times, but clients should be able to handle this)

Using LISTEN commands in this mode will lead to undefined behavior (you may not receive the notifications you want, and you may receive notifications you did not ask for).

### Session Pooling
Send each session to a new node. This mode supports all postgres features, but will not balance as well unless clients make new sessions often.

## Unsupported features
One day these will maybe be supported
- Reserve pool (for serving long-stalled clients)
- Auth methods other than plaintext, MD5, and SASL-SCRAM-SHA256
- GSSAPI
- Timeouts (other than transaction idle timeout)
- Statement Pooling (probably won't add, the benefit over transaction pooling is negligible and compatibility suffers greatly)
- pgbouncer stats database (probably won't add, a lot of work for something that can be done more easily by other means like prometheus)
