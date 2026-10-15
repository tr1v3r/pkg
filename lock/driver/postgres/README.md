# PostgreSQL lock driver

`github.com/tr1v3r/pkg/lock/driver/postgres` implements `lock.Driver` with a shared,
permanent PostgreSQL **12+** table. Production code depends only on `database/sql`
and the lock core. It works with both fixed-TTL `TryRun`/`Run` gates and
owner-checked `TryLock`/`Lock` leases, including explicit/optional automatic renewal.
PostgreSQL 12 is the syntax-compatibility minimum; use a maintained server release
for production deployments.

## Setup

The caller chooses a PostgreSQL SQL client and owns its connection pool. For
example, using pgx's `database/sql` adapter:

```go
import (
    "context"
    "database/sql"
    "os"
    "time"

    _ "github.com/jackc/pgx/v5/stdlib"
    "github.com/tr1v3r/pkg/lock"
    postgresdriver "github.com/tr1v3r/pkg/lock/driver/postgres"
)

func run(ctx context.Context) error {
    db, err := sql.Open("pgx", os.Getenv("DATABASE_URL"))
    if err != nil {
        return err
    }
    defer db.Close() // application-owned; normally shared until shutdown

    driver, err := postgresdriver.New(db, "myapp:gates:",
        postgresdriver.WithTable("public.pkg_lock_leases")) // optional, default table
    if err != nil {
        return err
    }
    // Explicit setup only. Prefer your normal serialized migrations in production.
    if err := driver.EnsureSchema(ctx); err != nil {
        return err
    }
    locker, err := lock.New(driver)
    if err != nil {
        return err
    }
    executed, err := locker.TryRun(ctx, "event:123", time.Minute,
        func(work context.Context) error { return processEvent(work) })
    _ = executed // false,nil means skipped; true means the callback ran
    return err
}
```

`processEvent` above is application code. For renewable leases, use the same
`driver` with `lock.WithAutoRenew()` or call `Lease.Renew` explicitly; always release
finished leases with a separate live cleanup context. See the [core guide](../../README.md).
Gate callbacks never renew or release their marker, even on failure or panic.
Both modes share keys: choose distinct prefixes/keys when their domains differ.

## Schema and ownership

`New` does not connect, ping, run DDL, or close the pool. `EnsureSchema` explicitly
executes `CREATE TABLE IF NOT EXISTS`; the schema must already exist. Equivalent
migration for the default table:

```sql
CREATE TABLE public.pkg_lock_leases (
    key        text COLLATE pg_catalog."C" PRIMARY KEY,
    owner      text COLLATE pg_catalog."C" NOT NULL,
    expires_at timestamp with time zone NOT NULL
);
```

- Use a dedicated, ordinary logged table with this shape, bytewise `C` collation,
  and full timestamp precision. Do not attach triggers/rules, RLS policies, or
  external writers that alter the protocol. A table need not be per process;
  all cooperating processes must use the same database/table/namespace.
- `EnsureSchema` does **not** validate or migrate an existing table. Concurrent
  `CREATE TABLE IF NOT EXISTS` can still race in PostgreSQL's catalogs; provision
  it through serialized migrations rather than every contender's hot path.
- `WithTable("leases")` uses `public.leases`; `WithTable("locks.leases")` uses that
  explicit schema. Each component is an ASCII identifier, at most 63 bytes,
  quoted and case-sensitive. SQL fragments and temporary schemas are rejected.
  Fixed qualification avoids `search_path` or temporary-table shadowing.
- Prefixes are concatenated literally. Keys/owners must be nonempty UTF-8 without
  NUL; prefixes may be empty. PostgreSQL's B-tree index size limit still applies
  to long keys: use short business identifiers or application-level hashes.
- The runtime role needs `SELECT`, `INSERT`, `UPDATE`, and `DELETE` on the table;
  `SELECT FOR UPDATE` participates in owner/expiry checks. Grant DDL separately
  if using `EnsureSchema`, or omit the call for a restricted runtime role.

## Atomicity, expiry, and timeouts

```text
[Acquire]
  INSERT ... ON CONFLICT DO UPDATE
    live row -> false (including same owner)
    expired row -> lock conflict row -> check DB clock -> install fresh owner/TTL

[Renew / release]
  MATERIALIZED held-row CTE -> SELECT ... FOR UPDATE (key only)
    -> check held.owner AND held.expires_at against current DB clock
    -> extend / delete, or false for absent/expired/successor-owned rows
```

Checking expiry in a plain `UPDATE ... WHERE expires_at > clock_timestamp()` can
happen **before** waiting for a row lock. The materialized CTE deliberately checks
its locked row's values **after** that wait, including updated successor ownership.
Do not replace it with an unlocked read or a pre-lock expiry filter.

Expiry uses `pg_catalog.clock_timestamp()`, not transaction-start `now()`.
Positive TTLs round **up** to microseconds and bind as exact integer hour/minute/
second/microsecond intervals: no floating-point multiplication, calendar-day/DST
arithmetic, integer overflow, or PostgreSQL 12 oversized interval fields.

TTL is not a fresh validity promise starting at response receipt. In particular,
first-INSERT values are evaluated before a unique-conflict wait; if the competing
insert rolls back, that wait can consume this reservation's TTL. Network/commit
latency can also consume validity. The core conservatively counts from **before**
the request and rejects exhausted TTLs; do not extend validity from reply time.

All operations, including schema setup, check cancellation before dispatch. A
supplied deadline is preserved; without one the driver adds a one-second deadline.
Late successful replies are rejected. SQL/context errors retain their identity
through wrapping, including uncertain/aborted outcomes; they are never busy results.
No extra goroutine/select wrapper returns while a provider call is still running.
A failed/canceled request can nevertheless have committed, or still commit, on the
server: cancellation is not proof of rollback.

**`database/sql` forwards contexts; it cannot force a noncompliant provider to
interrupt I/O.** Use a PostgreSQL SQL driver and connection configuration that honor
contexts and bound network I/O (the integration suite uses pgx stdlib). Custom
context watchers/dialers must retain those guarantees. Do not assume an auxiliary
cancel request alone bounds a blocked socket. Pool acquisition/row-lock waits can
consume the operation budget; failures do not prove a write never committed.

Database clock jumps, pauses, network partitions, replication/failover settings,
and administrative writes remain part of the safety model. This adapter does not
provide fencing, strict gate mutual exclusion, exactly-once work, or stronger
consistency/durability than the database itself. No external transaction is accepted:
operations use the pool's own single-statement transactions, never a caller's
long-lived `*sql.Tx`.

## Expired-row maintenance

Expiry is logical: expired rows cannot renew or block new acquisition even if not
physically deleted. Unique, never-reused gate keys otherwise accumulate, so arrange
periodic cleanup, for example:

```sql
DELETE FROM public.pkg_lock_leases
WHERE expires_at <= clock_timestamp();
```

For large tables, use bounded batches, a suitable expiry index, and an operational
statement/lock timeout. Reserve the table for this protocol; no background cleaner
is started by the driver. Never truncate a live lock table or delete unexpired rows.

## Tests and examples

```sh
go test -race ./lock/driver/postgres
# Explicit opt-in; each integration fixture creates/drops only its own table:
PKG_LOCK_POSTGRES_DSN='postgres://user:password@127.0.0.1:5432/testdb?sslmode=disable' \
  go test -race -tags=integration ./lock/driver/postgres
```

[`example_test.go`](example_test.go) has executable `Output` examples for all six
exported operations/options. Its SQL mock is test-only plumbing, not an in-memory
PostgreSQL substitute: real atomic SQL, expiry, contention, row-lock/deadline races,
and core integration are verified against PostgreSQL in the opt-in tests and CI.
