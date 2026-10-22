# pools

[English](README.md) | [简体中文](README.zh-CN.md)

A small, standard-library-only token pool for limiting concurrent work in Go.
You launch the goroutines; the pool controls admission and tracks completion.
It does not maintain reusable workers or a job queue.

## Installation

Requires Go 1.24 or later.

```sh
go get github.com/tr1v3r/pkg/pools
```

## Quick start

```go
package main

import (
    "fmt"

    "github.com/tr1v3r/pkg/pools"
)

func main() {
    p := pools.NewPool(2)
    inputs := []int{1, 2, 3, 4, 5}
    results := make([]int, len(inputs))
    for i, value := range inputs {
        p.Wait() // Acquire before launching the worker.
        go func() {
            defer p.Done()
            results[i] = value * value
        }()
    }
    p.WaitAll()
    fmt.Println(results) // [1 4 9 16 25]
}
```

Acquiring in the submitting goroutine bounds worker creation as well as active
work. Each worker writes to a separate result slot, and results are read only
after completion. Protect shared maps, appends, and other shared state separately.

## API

| Method | Behavior |
| --- | --- |
| `NewPool(size)` | Positive: bounded; negative: unlimited with completion tracking; zero: panics. |
| `Wait()` | Blocks until a token is acquired. |
| `WaitContext(ctx)` | Acquires a token or returns a context error; call `Done` only on success. |
| `AsyncWait()` (deprecated) | Starts an acquisition in a new goroutine; the returned channel closes once acquired. |
| `Done()` | Returns one token and marks that unit of work complete. |
| `Num()` | Snapshot of currently held bounded-pool tokens; zero in unlimited mode. Excludes blocked attempts. |
| `Size()` | Token channel capacity; zero in unlimited mode. |
| `WaitAll()` | Blocks until registered work completes, subject to the ordering rules below. |
| `AsyncWaitAll()` | Runs `WaitAll` in a new goroutine and closes the returned channel when it finishes. |

## Ordering and limitations

- Pair every successful acquisition with exactly one `Done()`, usually deferred
  inside the worker. An unmatched `Done()` may block or consume another worker's token.
- `WaitAll` follows `sync.WaitGroup` ordering rules. Finish submitting a batch
  before waiting. Do not put `Wait()` inside newly launched workers and immediately
  call `WaitAll()`: it can observe no registered work and return too early.
- `AsyncWait()` returning does not mean its acquisition has been registered or
  completed. Receive from its signal before starting the associated work or
  relying on a completion wait to cover it.
- Async signals cannot cancel operations. Abandoning an `AsyncWait()` signal can
  leave a pending goroutine or an acquired token without a matching `Done()`.
  A timeout around `AsyncWaitAll()` also leaves its waiter running until completion.
- A size of **zero** panics at construction instead of creating an unusable pool.
- A **negative** size disables the concurrency limit but retains completion tracking.
  Every successful acquisition still requires `Done()`, and `WaitAll()` waits for it.
- `WaitContext` rejects an already-cancelled context before registering work. A
  cancelled pending acquisition rolls back its completion count and retains no token.
  If cancellation races with available capacity, either success or cancellation is
  possible: use the return value to decide whether to call `Done()`. Cancellation
  after success does not release the token or stop the worker.
- Reuse a pool for a new batch after the previous completion
  wait returns. There is no `Close` method or persistent worker to shut down.
- The pool does not collect results/errors, recover panics, or cancel work.
  `Num()` is an observation, not a synchronization barrier.

## Runnable examples

Run from this module directory (`pools/`):

| Example | Command | Demonstrates |
| --- | --- | --- |
| [Batch](examples/batch/example.go) | `go run ./examples/batch` | Bounded workers and ordered results with `WaitAll`. |
| [Async acquisition](examples/async-acquire/example.go) | `go run ./examples/async-acquire` | Work while waiting for a token, then receive the signal and return the token. |
| [Async completion](examples/async-wait/example.go) | `go run ./examples/async-wait` | Continue other work before receiving the batch completion signal. |

Validate this module independently of the repository workspace:

```sh
GOWORK=off go test -race ./...
GOWORK=off go vet ./...
```

## Cancellable acquisition

```go
ctx, cancel := context.WithTimeout(context.Background(), time.Second)
defer cancel()
if err := p.WaitContext(ctx); err != nil {
    return err // No Done call on failure.
}
defer p.Done()
// Perform work. Pass ctx to the work separately if it should also be cancellable.
```

The fragment assumes a pool `p` and an enclosing function returning `error`;
import `context` and `time`. `WaitContext` runs in the calling goroutine.
Prefer it over `AsyncWait`, whose pending acquisition cannot be cancelled.

## Compatibility changes

Zero capacity now panics, and negative capacity now tracks completion. The new
`WaitContext(context.Context) error` method extends `Pool`; custom implementations
and mocks must implement it. `AsyncWait` remains available but is deprecated.
