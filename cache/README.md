# cache

A concurrency-safe, generic, in-memory TTL cache for Go 1.24 and later. The
package uses only the Go standard library and does not depend on other packages
in this repository.

## Usage

```go
package main

import (
    "fmt"
    "time"

    "github.com/tr1v3r/pkg/cache"
)

func main() {
    c := cache.New[string, int]()
    defer c.Close()

    if err := c.Set("answer", 42, time.Minute); err != nil {
        panic(err)
    }
    if value, ok := c.Get("answer"); ok {
        fmt.Println(value) // 42
    }
    c.Delete("answer")
}
```

Construct instances with `New[K comparable, V any](opts ...Option[K, V]) *Cache[K, V]`.
The zero value is not supported, and a cache instance must not be copied. Share
the pointer returned by `New` instead.

Background expiration cleanup defaults to `DefaultCleanupBatchSize` (128 entries)
and `DefaultCleanupBatchBudget` (1 ms) per pass. Configure these limits with
`WithCleanupBatchSize[K, V](n)` and `WithCleanupBatchBudget[K, V](d)`; a nonpositive
value disables the respective limit. Options receive the entire cache and are
applied before its cleaner starts. Use the same key and value type arguments
for the cache and its options:

```go
c := cache.New[string, int](
    cache.WithCleanupBatchSize[string, int](256),
    cache.WithCleanupBatchBudget[string, int](2*time.Millisecond),
)
defer c.Close()
```

## API and TTL semantics

| Method | Behavior |
| --- | --- |
| `Set(key K, value V, ttl time.Duration) error` | Inserts or replaces a value and its deadline. A successful overwrite starts a new TTL. |
| `Get(key K) (V, bool)` | Returns a live value and `true`; otherwise returns the zero value of `V` and `false`. Reads do not extend TTL. |
| `Delete(key K)` | Removes a key if present. Missing keys are harmless. |
| `Close()` | Closes the cache, clears all entries, and waits for the cleanup goroutine to exit. Idempotent. |

TTL must be strictly positive. On an open cache, zero or negative TTL returns
`ErrInvalidTTL` without changing the existing value or its deadline. There is no
special TTL for entries that never expire.

Keys must be equal to themselves. `Set` rejects non-reflexive keys, such as
floating-point NaNs or comparable structs containing NaNs, with `ErrInvalidKey`.
This prevents entries that Go maps cannot look up or delete from retaining values
after expiration. Interface-typed keys must also contain comparable dynamic
values, as with a regular Go map; uncomparable dynamic values panic.
Validation order is closed cache, TTL, key, then capacity for new keys; rejected
writes store nothing and leave existing entries unchanged.

Expiration is exact at the read boundary: an entry is expired when
`now >= deadline`. `Get` loads a published snapshot and checks its immutable deadline, returning a miss
for an expired entry even if the background worker has not removed it yet.
It does not physically delete expired entries. Cleanup scheduling can delay
memory reclamation, but it cannot make an expired entry readable.

After `Close`:

- `Set` returns `ErrClosed`.
- `Get` always returns a miss.
- `Delete` is a no-op.
- Further calls to `Close` are safe.

Use `errors.Is` with `cache.ErrInvalidTTL`, `cache.ErrInvalidKey`,
`cache.ErrClosed`, or `cache.ErrCapacity` to check the exported errors.

## Entry capacity

Capacity is unlimited by default. Set an optional entry limit at construction:

```go
c := cache.New[string, int](cache.WithMaxEntries[string, int](10_000))
defer c.Close()

if err := c.Set("answer", 42, time.Minute); err != nil {
    if errors.Is(err, cache.ErrCapacity) {
        // Continue without caching this value, or record the rejection.
    } else {
        // Handle invalid input or a closed cache.
    }
}
```

`WithMaxEntries[K, V](n)` limits retained entries, including expired entries
awaiting physical cleanup. A nonpositive value disables the limit. Repeated
options use the last value. Configure capacity before storing entries; changing
it after construction is unsupported, including by applying an option directly.

- A new key below capacity is inserted normally.
- At capacity, a new key reclaims at most one expired heap root before insertion.
  If the earliest entry is still live, `Set` returns `ErrCapacity` without changing
  any existing entry. Remaining overdue entries are left to background cleanup.
- An existing key can be replaced at capacity, including one that has expired but
  has not been physically removed. Replacements start a new TTL and use one slot.
- `Delete` and background expiration free slots for future writes.
- Unexpired entries are never evicted to admit new keys; there is no LRU/LFU policy.

Admission and insertion share the existing writer mutex, so concurrent writers
cannot oversubscribe the configured entry limit. Reads have no capacity-related
bookkeeping. Applications choose whether a rejected cache write should affect
business behavior; the cache does not silently discard writes.

This is an **entry count limit, not a byte or process-memory limit**. Replacing a
value with a larger one is allowed. Values may reference arbitrarily large or
shared objects, in-flight readers can retain old snapshots, and GC controls when
unreachable objects are reclaimed. A count limit does not establish a bound on
process RSS.

## Lifecycle and implementation

See [architecture and lifecycle diagrams](docs/README.md) for interactive,
source-linked views of the storage/scheduler architecture, cache shutdown, and
entry expiration. The standalone HTML files and their editable JSON sources live
in `docs/`.

Every call to `New` starts one background cleanup goroutine. **Always call
`Close`**, normally with `defer` immediately after construction. Closing is
synchronous: when it returns, the cache's cleanup goroutine has exited and its
entries have been cleared. A closed cache cannot be reopened; create a new one
when needed.

A `sync.Map` provides key lookup, and an indexed minimum heap orders expiration
deadlines. Each stored key has one current heap node. Overwrites create a new
node, replace the old heap slot, repair ordering with `heap.Fix`, then publish
the new node through `sync.Map.Store`. Published values and deadlines are never
mutated. Deletion removes the node with `heap.Remove`; repeated refreshes do
not accumulate stale expiration nodes.

The background worker schedules a timer for the earliest deadline rather than
polling at a fixed interval. `Set` and `Delete` wake it only when the earliest
deadline changes, including transitions to or from an empty heap. Changes at the
same deadline need no notification. The worker uses Go 1.24's synchronous timer
semantics: `Reset` directly replaces the previous deadline without a
stop-and-drain sequence. When the heap is empty, it stops the timer and waits only for a change or shutdown. The timer
is also stopped when the worker exits.

### Complexity

For `n` stored entries, including expired entries awaiting cleanup:

| Operation | Complexity |
| --- | --- |
| `Get` | Amortized O(1) index lookup |
| `Set` | O(log n) heap maintenance, plus amortized map operations |
| `Delete` | O(log n) for an existing key; average O(1) for a missing key |
| Background expiration | O(log n) per removed entry |
| Stored entries and expiration bookkeeping | O(n) |

These bounds describe data-structure work, not lock contention or scheduling
latency. The heap backing array can retain capacity from earlier peaks until
`Close`. Replaced snapshots can remain reachable by in-flight readers until
those reads finish, and unreachable snapshots are reclaimed by Go GC.

## Concurrency and limits

A single heap mutex serializes `Set`, `Delete`, cleanup, and `Close` so index
updates cannot race with expiration or reinsertion. `Get` uses `sync.Map.Load`
and immutable snapshot fields without taking that mutex. Only heap indexes
are mutable, and readers never inspect them. This does not promise a lock-free
implementation of `sync.Map` itself or a fixed scheduling latency.

Every overwrite allocates a new node instead of mutating one already published.
This trades extra allocation, GC work, and index memory for reader concurrency
and simpler synchronization. See the [selection experiment and decision](docs/experiments/syncmap-2026-10-15/README.md)
for measured costs, workload limits, raw data, and a frozen reproducer.

- Methods are safe to call concurrently on the same cache pointer.
- Synchronization protects the container, not mutable data stored in values.
  Maps, slices, pointers, and other reference-containing values are not cloned.
  Callers must synchronize access to shared mutable values themselves.
- Capacity is unlimited unless configured with `WithMaxEntries`. There is no
  byte-based limit or eviction of unexpired entries. Choose TTLs and entry limits
  according to the workload.
- There is no loader, singleflight/request deduplication, persistence, Redis
  integration, or distributed coordination.
- TTL expiration is a cache visibility guarantee, not a real-time callback or
  an exact memory-reclamation deadline.

## Examples and benchmarks

Executable external-package examples cover basic usage, invalid TTLs,
expiration, and closing:

```sh
go test ./cache
go test -race ./cache
```

On Go 1.25+, additional `testing/synctest` tests use virtual time to verify
cleanup scheduling, exact deadline boundaries, rescheduling, and shutdown.
These tests are version-gated; the package, core tests, examples, and benchmarks
remain compatible with Go 1.24. CI runs tests on Go 1.24, 1.25, 1.26, and 1.27.
Go 1.24 additionally runs
real-time scheduler smoke tests with generous watchdogs; exact virtual-time
assertions remain specific to Go 1.25+.

A dedicated CI stress job runs on all four Go versions with 1, 2, and 4 CPUs,
repeating each setting 10 times under the race detector:

```sh
go test -race -count=10 -cpu=1,2,4 -timeout=5m -run '^TestCacheConcurrent(Cleanup|Capacity|CapacityProtection|HotKey)Stress$' ./cache
```

The cleanup test uses the real worker and default cleanup limits, 128 workers, two
32,768-entry overdue backlogs, and 16 concurrent callers of `Close`. It checks
mixed reads/writes/refreshes/deletes, strict expiry, physical backlog reclamation,
map/heap consistency, and shutdown. It accepts valid scheduling orders rather
than requiring a writer to land in a particular cleanup window. Short-mode tests
skip this workload. The timeout is a hang watchdog; this is a correctness stress
test and does not establish QPS capacity or P99 latency. The capacity stress test
adds 128 workers with a 64-entry limit and checks admission, short TTLs, deletes,
map/heap consistency, and concurrent shutdown.

The capacity-protection stress test checks exact admission counts and persistent
long-TTL sentinels through repeated full-cache rejection, deletion, expiration,
and refill waves. It covers default cleanup, one-entry passes, tiny time budgets,
and unlimited passes with the real cleaner. The hot-key stress test distinguishes
value generations and checks visibility after completed refreshes, deletes, and
reinsertions; reads overlapping a write accept any valid writer in that wave.
Repeated lifecycle tests also verify that each cache's cleaner has exited after
concurrent shutdown. These bounded workloads are regression tests, not long-running
soak tests or memory-usage guarantees.

Run the representative microbenchmarks with:

```sh
go test ./cache -run '^$' -bench . -benchmem
```

They cover Get hits, misses, parallel reads, overwrites at several cache sizes,
Set/Delete pairs, and entry-capacity admission/rejection/reclamation. Setup and
population are excluded from the timed region.
Keys are reused, so retained entries do not grow with the benchmark iteration
count; the Set/Delete benchmark measures one pair per iteration. Long TTLs keep
ordinary benchmark runs focused on these operations rather than expiration.

`BenchmarkCacheMixedParallel` adds 50%, 90%, and 99% read workloads, one hot key
or 1,024 keys, and short/long TTLs. The remaining operations are writes and deletes
(99% reads leaves only writes). Run it without race/coverage instrumentation:

```sh
go test ./cache -run '^$' -bench '^BenchmarkCacheMixedParallel$' -benchmem -cpu=1,2,4
```

Its ns/op and allocations describe average operation cost. Tail latency, writer
fairness, and steady-state GC/memory require separate workload measurements.
Results depend on the machine, Go version, and workload; no performance numbers
are claimed here.

See the [historical split-lock measurements](docs/performance.md) and
[sync.Map selection experiment](docs/experiments/syncmap-2026-10-15/README.md)
for the successive designs, measurement boundaries, and reproducible snapshots.
