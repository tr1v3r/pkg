# lock

Backend-independent fixed-TTL execution gates and owner-checked lease locks for
Go 1.24+. The core uses only standard-library packages. Redis is the first driver,
not part of the public abstraction. A file or database driver can implement the
same atomic contract; neither is included in this version.

## Choose the behavior

| API | Purpose | After work finishes | Renewal |
| --- | --- | --- | --- |
| `TryRun` / `Run` | Suppress repeated execution within a fixed TTL window | Keep the reservation until TTL, including on callback error or panic | Never |
| `TryLock` / `Lock` | Hold an owner-checked lease for cooperative protected work | Caller explicitly releases the lease | Explicit, or opt-in automatic |

Both modes share the driver and key namespace, so a gate and a lease using the
same key contend. Reserve distinct prefixes/keys when their domains should differ.

## Fixed-TTL execution

Given a configured `lock.Driver` (Redis setup is shown below):

```go
locker, err := lock.New(driver)
if err != nil {
    return err
}
executed, err := locker.TryRun(ctx, "event:123", time.Minute, func(work context.Context) error {
    return processEvent(work) // application function
})
if err != nil {
    // executed == true means the callback ran and returned this exact error.
    // executed == false means failure occurred before callback invocation.
    return err
}
if !executed {
    return nil // another caller already reserved this key; no callback ran
}
// No Release: the marker remains until its original TTL expires.
```

`TryRun` makes one acquisition attempt. `Run` retries only contention, every 50ms
by default (`WithRetryInterval`), until acquisition or context cancellation. Use a
caller deadline to bound waiting; backend and callback errors are never retried.

- `(false, nil)`: the key was busy; callback was not invoked (`TryRun` only).
- `(false, err)`: invalid arguments, cancellation, backend failure, or a successful
  acquisition reply received after its operation deadline/local TTL; no callback ran.
- `(true, err)`: callback was invoked once and returned this exact error, including
  nil. A callback returning `ErrNotAcquired` is still a callback failure, not a retry.

Keys must be nonempty, TTLs positive, and callbacks non-nil (`ErrInvalidKey`,
`ErrInvalidTTL`, `ErrNilFunc`). Every attempt gets a fresh owner token. Acquisition
requests are bounded by `WithOperationTimeout`; the request context is canceled
before invoking the callback, which receives the **original caller context**.

Neither `WithOperationTimeout` nor the reservation TTL sets a callback timeout.
Work may continue past TTL and overlap a later execution; callers must cooperate
with their own context. Reservations are not renewed, deleted, or rolled back on
callback success, error, panic, or cancellation. Even an uncertain/late acquisition
is left to expire without cleanup. `WithAutoRenew` has no effect on these APIs.
No lease handles, lifecycle workers, or result caching are created by them. This is
bounded-window execution suppression, **not** strict mutual exclusion, exactly-once
execution, durable idempotency, or shared-result singleflight.

## Lease usage

```go
import (
    "context"
    "time"

    goredis "github.com/redis/go-redis/v9"
    "github.com/tr1v3r/pkg/lock"
    redisdriver "github.com/tr1v3r/pkg/lock/driver/redis"
)

func run(ctx context.Context, client *goredis.Client) error {
    // Construct client with ContextTimeoutEnabled: true and socket deadlines
    // enabled. New rejects known clients that would ignore operation deadlines.
    driver, err := redisdriver.New(client, "myapp:locks:")
    if err != nil {
        return err
    }
    locker, err := lock.New(driver,
        lock.WithAutoRenew(), // opt-in; omit for explicit Renew only
        lock.WithOperationTimeout(time.Second),
    )
    if err != nil {
        return err
    }
    lease, err := locker.Lock(ctx, "resource:123", 10*time.Second)
    if err != nil {
        return err
    }
    defer func() {
        // Acquisition-context cancellation stops renewal, but cleanup needs a
        // fresh, bounded context. Handle/log the error in production.
        cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Second)
        defer cancel()
        _ = lease.Release(cleanup)
    }()

    // Pass lease.Context() to work that cooperatively stops on cancellation.
    // Consult lease.Err() to distinguish expiry/loss from a closed handle.
    return performWork(lease.Context()) // application function
}
```

A typical client is created as:

```go
client := goredis.NewClient(&goredis.Options{
    Addr: "127.0.0.1:6379",
    ContextTimeoutEnabled: true,
})
defer client.Close() // the caller owns client resources
```

## API and lifecycle

- `New(driver, options...)`: validates the driver and configuration; it does not
  start workers or take ownership of backend connections.
- `TryRun(ctx, key, ttl, fn)` / `Run(ctx, key, ttl, fn)`: fixed-TTL execution as
  described above; neither method creates a lease or uses renewal/release.
- `TryLock(ctx, key, ttl)`: one attempt; contention is `ErrNotAcquired`, while a
  backend error is a separate failure. Empty keys and nonpositive TTLs are rejected.
- `Lock(ctx, key, ttl)`: retries contention only, every 50ms by default. Change the
  interval with `WithRetryInterval`; context cancellation interrupts the wait.
- Every acquisition attempt uses a fresh cryptographic owner token, including
  attempts by the same Locker after a previous lease expired or was released.
- `Lease.Renew(ctx)`: renews by the original TTL. No new acquisition, no revival
  of a closed/expired handle. A canceled wait before a request does not abandon
  an otherwise-valid lease; an error after dispatch fails closed.
- `WithAutoRenew()`: off by default. Enables renewal around TTL/3 after acquisition
  and each successful renewal. Explicit Renew remains available in this mode.
  This option affects lease acquisition only, never `TryRun` or `Run`.
- `Lease.Done()`, `Err()`, `Context()`: local invalidation notification/cause and a
  context for cooperative work. The first terminal cause is retained. Expiry,
  cancellation, ownership loss, and uncertain renewal errors invalidate the handle.
- `Lease.Deadline()`: conservative local deadline, measured from before the
  acquisition/renewal request, not from its response. It is not ownership proof.
- `Lease.Release(ctx)`: closes the local handle even if cleanup fails; stops its
  workers before owner-checked remote cleanup. Successful release is idempotent;
  missing/successor-owned keys return `ErrLeaseLost`. Failed cleanup can be retried
  with another context, but the handle can never renew again. `ErrClosed` denotes
  local closure, not proof of successful remote deletion; check Release's result.

```text
[Fixed-TTL execution: TryRun / Run]
  fresh owner -> atomic Driver.TryAcquire
                busy: TryRun skips / Run waits with caller context
                acquired -> fn(original ctx) -> preserve marker until TTL
                error/panic/cancel ----------> preserve marker until TTL
  no lease, renewal, cleanup, or lifecycle worker; TTL does not time out fn

[Lease acquire: TryLock / Lock]
  fresh owner -> atomic Driver.TryAcquire -> Lease
                 contention: no work; backend error: uncertain outcome

[Live lease]
  expiry/cancellation watcher ---------------------------> Done / Context closed
  explicit Renew / optional renewal worker -> bounded RPC -> update local deadline
                                            failure ----> permanent local loss

[Release]
  close local handle -> cancel requests -> wait for workers (caller-context bounded)
                     -> owner-checked Driver.Release -> success / lost / uncertain
```

Each lease has one expiry/cancellation watcher; opt-in automatic renewal adds one
separate worker. A slow renewal RPC cannot hold local loss notification hostage.
Release waits for workers, but its caller context bounds that wait. Otherwise
expiry or acquisition-context cancellation eventually ends the workers. Always
Release when finished rather than waiting for the TTL unnecessarily.

### Timeouts, cancellation, and uncertainty

Every core driver request has a finite deadline (1s by default, configurable with
`WithOperationTimeout`). Earlier caller deadlines win; renewal is additionally
bounded by the remaining local lease validity.
No untracked goroutines are used to return early while a remote request continues.

Some transports, including go-redis, cannot immediately interrupt an already
blocked socket read on plain context cancellation. An in-flight operation can
drain until its deadline even though Done/Context have already signaled local
loss. Redis drivers require deadline-capable known client configurations; custom
Client implementations must honor provided deadlines too.

Acquisition errors may mean the key was created remotely; it is left to expire.
For lease APIs, late successful acquisitions that already outlived local validity
are rejected and best-effort released. Gate APIs instead always leave their marker
to expire and never invoke the callback after a canceled/expired acquisition.
Renewal errors permanently abandon the handle even if
Redis might have renewed it; subsequent work must not rely on that uncertainty.
Cancellation does not automatically delete remote ownership; use Release with a
separate live cleanup context. A failed release can leave ownership until TTL.

## Driver contract

```go
type Driver interface {
    TryAcquire(ctx context.Context, key, owner string, ttl time.Duration) (bool, error)
    Renew(ctx context.Context, key, owner string, ttl time.Duration) (bool, error)
    Release(ctx context.Context, key, owner string) (bool, error)
}
```

Drivers must be concurrent-safe and honor operation deadlines. Acquisition of an
absent/expired key, owner-checked renewal, and owner-checked release must each be
atomic. Expired owners cannot revive; stale owners cannot touch successors. TTLs
must not be rounded down. False is contention/lost ownership; errors are failures
with potentially ambiguous outcomes. A file driver needs real cross-process
atomic coordination and expiration checks; plain file read/write is insufficient.

The Redis driver uses one `SET ... NX PX ...`, and Lua owner-check-plus-`PEXPIRE`
or owner-check-plus-`DEL`. Sub-millisecond TTLs round up, never down. Prefixes are
concatenated literally; reserve a namespace and avoid collisions with unrelated
keys. It accepts go-redis's normal/cluster clients through a small Do/Eval interface.

## Safety boundaries

Neither execution gates nor leases provide fencing, exactly-once execution, or a
complete business-idempotency solution. An expired or paused owner can continue executing
application code; cancellation cannot forcibly stop it or undo side effects.
For correctness-critical external mutations, the target resource needs fencing,
transactions, unique constraints, or another independent safety mechanism.

Redis asynchronous failover may lose an acknowledged lock; clock changes,
partitions and process pauses also affect the safety model. No Redlock/quorum
algorithm, reentrancy, fairness, multi-key transactions, or fencing tokens are
provided. Core abstractions do not strengthen a backend's actual guarantees.

## Executable examples

[`example_test.go`](example_test.go) demonstrates construction/options, both
execution and lease entry points, renewal/release, and lease state/context queries.
Examples also distinguish skipped execution from callback failure and show how to
cancel a wait. The test-only setup uses miniredis rather than an external server;
replace it with your configured client in an application. The Redis adapter has
its own [driver examples](driver/redis/example_test.go).

```sh
go test -run '^Example' ./lock/...
```

## Testing

```sh
go test -race ./lock/...
# Optional real Redis tests; unique namespace, no FLUSHDB:
PKG_LOCK_REDIS_ADDR=127.0.0.1:6379 go test -race -tags=integration ./lock/...
```

Default Redis tests use miniredis and exercise owner checks, expiry, Lua,
contention, rounding and failure responses. Core tests include a memory driver,
concurrency/cancellation scenarios, plus Go 1.25+ virtual-time tests for renewal,
expiry, conservative deadlines and stalled requests. Core/default tests also
run on Go 1.24; the virtual-time files are build-tagged for newer toolchains.
