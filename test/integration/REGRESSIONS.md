# Pooler regression scenarios

These tests exercise real pgx clients through pggat against embedded PostgreSQL.
They were independently written in Go after comparing upstream scenarios. No
upstream test code or fixtures were copied. Pggat's license is unchanged.

## Retained coverage

| Test | Contract | Scenario source |
|---|---|---|
| `TestHybridReadOnlyTransaction` | A client-requested read-only transaction returns `ERROR 25006`, stays failed until rollback, and does not disconnect. Tests simple and extended protocol. | Pggat-specific defect found during the comparison, not an upstream port. |
| `TestPreparedStatementNameCollisionAcrossBackends` | Two clients using the same statement name get their own SQL results on one physical backend, and a prepared statement works after a forced backend change. Backend PIDs are asserted. | PgBouncer [same-name statements](https://github.com/pgbouncer/pgbouncer/blob/7d38761c8f6c757238fde9f942cf9fe0cd272ae3/test/test_prepared.py#L53-L95) and [backend changes](https://github.com/pgbouncer/pgbouncer/blob/7d38761c8f6c757238fde9f942cf9fe0cd272ae3/test/test_prepared.py#L12-L49). |
| `TestDeallocateAllInvalidatesStatementAcrossMigration` | SQL `DEALLOCATE ALL` invalidates protocol-level statements even after the client changes backends. Executing the old name returns `26000` instead of silently recreating it. | PgBouncer [deallocation and discard](https://github.com/pgbouncer/pgbouncer/blob/7d38761c8f6c757238fde9f942cf9fe0cd272ae3/test/test_prepared.py#L52-L95). |
| `TestFailedPrepareRecoversOnSameConnection` | A failed Parse does not poison the connection or the subsequent use of that statement name. | PgBouncer [failed prepare](https://github.com/pgbouncer/pgbouncer/blob/7d38761c8f6c757238fde9f942cf9fe0cd272ae3/test/test_prepared.py#L487-L493). |
| `TestCopyFromErrorInTransaction` | Invalid COPY input reaches the client as a PostgreSQL error, leaves the transaction failed until rollback, and allows a later successful COPY on the same client. | PgBouncer [ordinary COPY failure](https://github.com/pgbouncer/pgbouncer/blob/7d38761c8f6c757238fde9f942cf9fe0cd272ae3/test/test_copy.py#L34-L42), extended here to transaction recovery. This does not reproduce the late-CopyDone race. |
| `TestCancelRequestTargetsOnlyActiveClient` | An active client receives `57014` and recovers on the same backend. An idle client's cancel does not interrupt another client using that backend. | PgBouncer [cancel](https://github.com/pgbouncer/pgbouncer/blob/7d38761c8f6c757238fde9f942cf9fe0cd272ae3/test/test_cancel.py#L9-L17) and [ownership race](https://github.com/pgbouncer/pgbouncer/blob/7d38761c8f6c757238fde9f942cf9fe0cd272ae3/test/test_cancel.py#L80-L112). This does not reproduce the in-flight race. |
| `TestInFlightCancelDoesNotReachNextClient` | A cancel pggat is still forwarding keeps its backend from the next client until PostgreSQL confirms it, so it cannot interrupt the next query. Covers transaction and hybrid pools. | PgBouncer [cancel race](https://github.com/pgbouncer/pgbouncer/blob/7d38761c8f6c757238fde9f942cf9fe0cd272ae3/test/test_cancel.py#L80-L112), made deterministic by a proxy that holds the forwarded CancelRequest. |
| `TestUnconfirmedCancelReplacesBackend` | A backend whose forwarded cancel was not confirmed is replaced rather than reused. Covers transaction and hybrid pools. | Pggat-specific cancellation lifecycle. |
| `TestHybridCancelRequest` | A cancel reaches a query on the hybrid primary, which returns `57014` and keeps its backend. | Pggat-specific defect: hybrid cancels were routed to the replica pool. |
| `TestDisconnectInTransactionRollsBack` | Graceful and abrupt disconnects roll back writes, release transaction locks, and return the same backend to the pool. | PgCat [disconnect during a transaction](https://github.com/postgresml/pgcat/blob/5b038813eb14f181434ab7b5509e74d9b1fe123b/tests/ruby/misc_spec.rb#L180-L188), checked through PostgreSQL data and backend PIDs rather than a mocked query counter. |
| `TestInvalidStartupParameterKeepsBackend` | A rejected startup setting returns `22023` without replacing the healthy backend or leaking settings to its next client. Covers transaction, session, and hybrid primary pools. | Pggat-specific lifecycle defect found during the coverage audit. |
| `TestTerminateDoesNotWaitForBackend` | An idle client's Terminate closes promptly while the only backend belongs to another transaction. | Pggat-specific lifecycle defect found during the coverage audit. |
| `TestIdleTerminatedBackendIsReplaced` | A terminated idle backend is replaced before its next client's query is sent. The client remains usable and receives a new backend PID. | Odyssey [server-closed regression](https://github.com/yandex/odyssey/blob/456131c5ae1e292cf0b1353a2ac9295d2b6ed38a/test/pg_regress/tests/broken_conn/sql/server_closed.sql). |
| `TestExtendedProtocolFlush` | Flush delivers Parse, Bind, Describe, Execute, Close, suspended-portal and error responses without requiring Sync. Covers ordered results and large output streams. | PostgreSQL 18 [extended-query protocol flow](https://www.postgresql.org/docs/18/protocol-flow.html#PROTOCOL-FLOW-EXT-QUERY), with direct-server controls. |
| `TestExtendedProtocolCopyIn` / `TestExtendedProtocolCopyOut` | COPY started by Flush or Sync completes or reports its error, then allows protocol recovery. Flush and Sync inside COPY IN are ignored as PostgreSQL specifies. | PostgreSQL 18 [COPY protocol flow](https://www.postgresql.org/docs/18/protocol-flow.html#PROTOCOL-COPY), with direct-server controls. |
| `TestExtendedProtocolFlushEarlyResponses` | Responses from an earlier Flush arrive while a later Execute is blocked on a real advisory lock. | PostgreSQL 18 protocol flow, with a direct-server control and observed lock state. |
| `TestExtendedProtocolFlushReleasesBackend` | A flushed pipeline releases its sole backend after Sync or client disconnect, without leaking responses to the next client. | Pggat-specific lifecycle coverage. |
| `TestExtendedProtocolStalledStreamDelivery` | Rows and COPY OUT data already delivered by a direct PostgreSQL control reach pooled clients before a later row's advisory lock is released. Checks payload and ordering in transaction, session, and hybrid modes. | Pggat-specific buffered-output defect, compared with a real PostgreSQL control. |
| `TestDecoderHasBufferedTypedPacket` | A partial next header or body does not suppress output flushing before a transport read. Covers unread current bodies and invalid lengths. | Pggat's decoder framing contract. |
| `TestPreparedStatementRecoveryAfterBackendChange` | A stale statement returns its own `42P01` after a confirmed backend change, unrelated statements still work, and recreation permits recovery. Covers Describe, repeated Bind, name replacement, and error-until-Sync behavior. | Pggat-specific internal re-prepare defect found during the coverage audit, with direct PostgreSQL controls. |

The original hybrid and deallocation regressions fail on the pre-fix code:

- The hybrid pool changes an ordinary `25006` error from the primary into
  `FATAL XX000` and closes the client. Only replica errors should request a retry
  on the primary.
- `DEALLOCATE ALL` leaves pggat's statement tracking intact, so changing backends
  silently recreates a statement the client already deallocated. Clear the
  prepared statements without discarding portals or queued protocol requests.

Rejected startup settings now count as client errors after PostgreSQL has sent
ReadyForQuery. Transport failures and FATAL errors still discard the backend.
The basic pool also handles an idle Terminate before acquiring a server. Both
new regression tests fail on the previous implementation.

The other original tests protect missing contracts that already worked when
this comparison was made.

Idle checkout now rejects pending input or a closed socket before pairing. It
never retries an application query that may have executed. This is a best-effort
snapshot, not a guarantee that the backend cannot fail immediately afterward.
Socket peeking is available on Linux, macOS, and the supported BSD targets.
Other platforms and custom transports can only inspect buffered input. TLS
probes also check internal buffered input, bound reads and writes, and reject
transport write failures. A TLS KeyUpdate requesting a response can cause a
conservative reconnect. TLS probe diagnostics use real TLS transports, but the
embedded PostgreSQL fixture still disables backend TLS.

Flush handling now reads the outstanding requests' responses without waiting
for Sync. Error responses stop the drain until Sync, COPY retains its outstanding
Execute, and standalone Sync returns an idle ReadyForQuery status. The wire
regressions compare against direct PostgreSQL and fail on the old implementation.
Before reading a packet that is not fully buffered, the relay flushes client output.
This delivers received rows and COPY data during a backend stall while still batching
packets that are already buffered.

Cancellation now pins a client's backend until PostgreSQL closes the cancel socket.
A failed write, reset, or missing confirmation retires the backend before reuse.
The cancel transport has a ten-second cap and also closes when its context is canceled.
A proxy holding real CancelRequests proves that a late cancel cannot reach the next
client, and a reset proves that an unconfirmed backend is replaced.

Prepared statements are restored when Bind or statement Describe first uses them.
An injected ParseComplete is hidden from the client, but its error is forwarded.
This prevents one stale statement from suppressing unrelated re-prepares in a batch.
Internal portal restoration drains through ReadyForQuery and reports server errors.

The single-server native JSON fixture uses the existing recipe min/max limits
of one. It prevents a test from accidentally passing by using another backend.
Its tests run serially. The migration case uses the existing `occupy` helper
instead, so it proves a backend change rather than relying on pool scheduling.

## Design differences

Pgpool-II classifies queries before selecting a server, including
[locking reads and data-changing CTEs](https://github.com/pgpool/pgpool2/blob/a76292e80f388dea26893cc63eb270a6be9011c1/src/protocol/pool_process_query.c#L1186-L1204).
Pggat's hybrid pool tries a replica and replays on the primary when PostgreSQL
rejects a write with `25006`. A SQL-classifier unit test would not protect that
path. It needs tests against an actual standby, including the response stream
seen by the client during replay.

## Remaining gaps

1. **Real replica routing and replay.** The existing hybrid Gatfile has no
   replica and exercises primary fallback only. Add an actual standby fixture
   before claiming coverage for writes, `SELECT FOR UPDATE`, data-changing CTEs,
   multi-statement transactions, or partial results followed by a write error.
   Pgpool-II's [multi-statement transaction scenarios](https://github.com/pgpool/pgpool2/blob/a76292e80f388dea26893cc63eb270a6be9011c1/src/test/regression/tests/001.load_balance/sql/7.sql)
   are useful inputs, but its routing assertions are not transferable directly.
2. **Late CopyDone and large pipelines.** The ordinary COPY recovery test does
   not cover PgBouncer's [late-CopyDone race](https://github.com/pgbouncer/pgbouncer/blob/7d38761c8f6c757238fde9f942cf9fe0cd272ae3/test/test_copy.py#L142-L178)
   or full-duplex streaming under socket backpressure. Avoid timing-only sleeps
   when adapting these scenarios.
3. **Stale duplicate statement names.** After migration, a duplicate named Parse
   whose cached original SQL no longer parses can return the original Parse's
   error instead of `42P05`. Bind and statement Describe now report the original
   error correctly, and valid cached names retain duplicate-name behavior.
   Exact duplicate-name error precedence for invalid cached SQL remains a follow-up.

## Source versions and licenses

| Project | Inspected commit | License source |
|---|---|---|
| PgBouncer | `7d38761c8f6c757238fde9f942cf9fe0cd272ae3` | [ISC](https://github.com/pgbouncer/pgbouncer/blob/7d38761c8f6c757238fde9f942cf9fe0cd272ae3/COPYRIGHT) |
| PgCat | `5b038813eb14f181434ab7b5509e74d9b1fe123b` | [MIT](https://github.com/postgresml/pgcat/blob/5b038813eb14f181434ab7b5509e74d9b1fe123b/LICENSE) |
| Odyssey | `456131c5ae1e292cf0b1353a2ac9295d2b6ed38a` | [BSD-3-Clause](https://github.com/yandex/odyssey/blob/456131c5ae1e292cf0b1353a2ac9295d2b6ed38a/LICENSE) |
| Pgpool-II | `a76292e80f388dea26893cc63eb270a6be9011c1` | [Custom permissive license](https://github.com/pgpool/pgpool2/blob/a76292e80f388dea26893cc63eb270a6be9011c1/COPYING) |

If upstream source is copied in a future change, retain its applicable copyright
and license notices. Referencing a scenario here does not make a differently
scoped test cover the original upstream regression.
