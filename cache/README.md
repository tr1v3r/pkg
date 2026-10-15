# cache

A concurrency-safe, generic, in-memory TTL cache for Go 1.24 and later. The
package uses only the Go standard library and does not depend on other packages
in this module.

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

Construct instances with `New[K comparable, V any]() *Cache[K, V]`. The zero
value is not supported, and a cache instance must not be copied. Share the
pointer returned by `New` instead.

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
Validation order is closed cache, TTL, then key; rejected writes store nothing.

Expiration is exact at the read boundary: an entry is expired when
`now >= deadline`. `Get` checks the deadline under a read lock and returns a miss
for an expired entry even if the background worker has not removed it yet.
It does not physically delete expired entries. Cleanup scheduling can delay
memory reclamation, but it cannot make an expired entry readable.

After `Close`:

- `Set` returns `ErrClosed`.
- `Get` always returns a miss.
- `Delete` is a no-op.
- Further calls to `Close` are safe.

Use `errors.Is` with `cache.ErrInvalidTTL`, `cache.ErrInvalidKey`, or
`cache.ErrClosed` to check the exported errors.

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

A map provides key lookup, and an indexed minimum heap orders expiration
deadlines. Each stored key has one heap node. Overwrites update that node with
`heap.Fix`; deletion removes it with `heap.Remove`. Repeated refreshes therefore
do not accumulate stale expiration nodes.

The background worker schedules a timer for the earliest deadline rather than
polling at a fixed interval. `Set` and `Delete` wake it to reconsider the next
deadline.

### Complexity

For `n` stored entries, including expired entries awaiting cleanup:

| Operation | Complexity |
| --- | --- |
| `Get` | O(1), using the usual average-case map lookup assumption |
| `Set` | O(log n) heap maintenance, plus amortized map operations |
| `Delete` | O(log n) for an existing key; average O(1) for a missing key |
| Background expiration | O(log n) per removed entry |
| Stored entries and expiration bookkeeping | O(n) |

These bounds describe data-structure work, not lock contention or scheduling
latency. Removed entries release their key/value references, but Go maps and
heap backing arrays can retain capacity from earlier peaks until `Close`;
allocated capacity need not shrink with the current number of entries.

## Concurrency and limits

- Methods are safe to call concurrently on the same cache pointer.
- Synchronization protects the container, not mutable data stored in values.
  Maps, slices, pointers, and other reference-containing values are not cloned.
  Callers must synchronize access to shared mutable values themselves.
- There is no capacity limit, size-based eviction, LRU/LFU policy, or automatic
  bound on application memory usage. Choose TTLs and key cardinality accordingly.
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
remain compatible with Go 1.24. CI runs tests on Go 1.24 and the stable release.

Run the representative microbenchmarks with:

```sh
go test ./cache -run '^$' -bench . -benchmem
```

They cover Get hits, misses, parallel reads, overwrites at several cache sizes,
and Set/Delete pairs. Setup and population are excluded from the timed region.
Keys are reused, so retained entries do not grow with the benchmark iteration
count; the Set/Delete benchmark measures one pair per iteration. Long TTLs keep
ordinary benchmark runs focused on these operations rather than expiration.
Results depend on the machine, Go version, and workload; no performance numbers
are claimed here.
