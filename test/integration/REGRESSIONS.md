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
| `TestDisconnectInTransactionRollsBack` | Graceful and abrupt disconnects roll back writes, release transaction locks, and return the same backend to the pool. | PgCat [disconnect during a transaction](https://github.com/postgresml/pgcat/blob/5b038813eb14f181434ab7b5509e74d9b1fe123b/tests/ruby/misc_spec.rb#L180-L188), checked through PostgreSQL data and backend PIDs rather than a mocked query counter. |
| `TestInvalidStartupParameterKeepsBackend` | A rejected startup setting returns `22023` without replacing the healthy backend or leaking settings to its next client. Covers transaction, session, and hybrid primary pools. | Pggat-specific lifecycle defect found during the coverage audit. |
| `TestTerminateDoesNotWaitForBackend` | An idle client's Terminate closes promptly while the only backend belongs to another transaction. | Pggat-specific lifecycle defect found during the coverage audit. |

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

1. **Protocol Flush without Sync, confirmed.** A scratch diagnostic sent
   Parse, Describe and Flush without Sync. Direct PostgreSQL returned the
   description immediately. Pggat returned no messages before a five-second
   read deadline. `backends.eqp` waits for more client packets rather than
   reading the backend until Sync. Repair needs a separate protocol change,
   including response ordering, error draining, and backpressure coverage.
   The diagnostic is not part of the passing CI suite.
2. **Idle backend termination, confirmed.** Terminating a pooled backend while
   idle can make its next client receive `FATAL 57P01` and disconnect. Odyssey's
   [server-closed regression](https://github.com/yandex/odyssey/blob/456131c5ae1e292cf0b1353a2ac9295d2b6ed38a/test/pg_regress/tests/broken_conn/sql/server_closed.sql)
   motivates checkout-liveness coverage. Any repair must avoid retrying a query
   whose execution status is unknown.
3. **Real replica routing and replay.** The existing hybrid Gatfile has no
   replica and exercises primary fallback only. Add an actual standby fixture
   before claiming coverage for writes, `SELECT FOR UPDATE`, data-changing CTEs,
   multi-statement transactions, or partial results followed by a write error.
   Pgpool-II's [multi-statement transaction scenarios](https://github.com/pgpool/pgpool2/blob/a76292e80f388dea26893cc63eb270a6be9011c1/src/test/regression/tests/001.load_balance/sql/7.sql)
   are useful inputs, but its routing assertions are not transferable directly.
4. **In-flight cancellation across backend reuse.** PgBouncer's
   [cancel race](https://github.com/pgbouncer/pgbouncer/blob/7d38761c8f6c757238fde9f942cf9fe0cd272ae3/test/test_cancel.py#L80-L112)
   sends many simultaneous cancellations while another client reuses a backend.
   Idle-client cancellation coverage is not proof that an already forwarded
   cancellation cannot hit a later owner. This needs a distinct concurrency
   regression and cancellation lifecycle analysis.
5. **Late CopyDone and large pipelines.** The ordinary COPY recovery test does
   not cover PgBouncer's [late-CopyDone race](https://github.com/pgbouncer/pgbouncer/blob/7d38761c8f6c757238fde9f942cf9fe0cd272ae3/test/test_copy.py#L142-L178)
   or full-duplex streaming under socket backpressure. Avoid timing-only sleeps
   when adapting these scenarios.

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
