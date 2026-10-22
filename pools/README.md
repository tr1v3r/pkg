# pools

[English](README.md) | [简体中文](README.zh-CN.md)

A standard-library-only, context-aware concurrency limiter for Go 1.24+.
The pool controls admission; callers own goroutines, results, errors, and completion.
It does not maintain workers, queue jobs, or start background goroutines.

## Installation

```sh
go get github.com/tr1v3r/pkg/pools
```

## Quick start

```go
package main

import (
    "context"
    "fmt"
    "log"
    "sync"

    "github.com/tr1v3r/pkg/pools"
)

func main() {
    p := pools.NewPool(2)
    var wg sync.WaitGroup
    results := make([]int, 5)
    for i := range results {
        if err := p.Acquire(context.Background()); err != nil {
            log.Fatal(err)
        }
        wg.Add(1)
        go func() {
            defer wg.Done()
            defer p.Release()
            results[i] = (i + 1) * (i + 1)
        }()
    }
    wg.Wait()
    fmt.Println(results) // [1 4 9 16 25]
}
```

Acquire before spawning workers to bound their creation. Each worker writes to
its own result slot; read results after `wg.Wait`. Shared maps and appends need
separate synchronization. For request-scoped work, pass its context to `Acquire`.
If submission fails partway through, join already-started workers before returning;
the [batch example](examples/batch/example.go) demonstrates this with `defer wg.Wait()`.

## API

```go
type Pool interface {
    Acquire(ctx context.Context) error
    Release()
    Num() int
    Size() int
}
```

- `NewPool(size)`: positive capacity bounds concurrency; negative capacity disables
  the limit; zero panics. Use the constructor; the zero value is not a public API.
- `Acquire(ctx)`: blocks for capacity or returns a context error. Pass a non-nil
  context. `context.Background()` explicitly opts into waiting without a deadline.
  An already-cancelled context is rejected even when capacity is available or unlimited.
- Call `Release()` exactly once after each successful acquisition, usually with
  `defer`. On acquisition failure, do not release. Cancellation after success does
  not release capacity or stop the work; propagate the context to the work separately.
- If cancellation races with capacity becoming available, either outcome may win.
  The return value determines whether the caller owns a token.
- `Release()` is a no-op in unlimited mode. In bounded mode, it panics when no token
  is held. Tokens have no owner identity: releasing another worker's token cannot
  be detected, so correct pairing remains the caller's responsibility.
- `Num()` reports held bounded-pool tokens, excluding blocked requests. `Size()`
  reports the capacity. Both return zero in unlimited mode. Neither tracks completion.
- No `Close` is needed. Use `sync.WaitGroup` for joining work, and call `Add` before
  starting a worker and before beginning the completion wait for an idle group.
  The pool does not recover panics or aggregate errors.

## Cancellation

Inside an error-returning function with a pool `p` and parent context `ctx`:

```go
ctx, cancel := context.WithTimeout(ctx, time.Second)
defer cancel()
if err := p.Acquire(ctx); err != nil {
    return err // No token acquired; do not Release.
}
defer p.Release()
// Perform work here.
```

Import `context` and `time`. `Acquire` waits in the calling goroutine without
spawning an internal waiter. For asynchronous execution, launch and manage your
own goroutine; acquiring before launching usually provides better backpressure.

## Examples and validation

Run from `pools/`:

| Example | Command |
| --- | --- |
| [Bounded batch and ordered results](examples/batch/example.go) | `go run ./examples/batch` |
| [Acquisition timeout](examples/cancellation/example.go) | `go run ./examples/cancellation` |
| [Caller-owned completion notification](examples/completion/example.go) | `go run ./examples/completion` |

```sh
GOWORK=off go test -race ./...
GOWORK=off go vet ./...
```

## Breaking migration

| Removed API | Replacement |
| --- | --- |
| `Wait()` | Handle the error from `Acquire(ctx)`; use `context.Background()` for unbounded waiting. |
| `WaitContext(ctx)` | `Acquire(ctx)` |
| `Done()` | `Release()` |
| `AsyncWait()` | Call `Acquire(ctx)` directly, or manage your own goroutine and its result. |
| `WaitAll()` | Caller-owned `sync.WaitGroup`. |
| `AsyncWaitAll()` | Caller-owned goroutine that waits on a `sync.WaitGroup` and closes a channel. |

Custom implementations and mocks must implement the new `Pool` interface.
Zero capacity now panics. Unlimited mode does not track completion, and neither
does bounded mode: move all batch completion tracking to the caller. Unmatched
bounded releases now panic instead of blocking. `Num` and `Size` remain available.
