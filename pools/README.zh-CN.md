# pools

[English](README.md) | [简体中文](README.zh-CN.md)

仅依赖标准库的 Go 并发限制器，支持通过 context 取消等待，需要 Go 1.24+。
池只负责控制准入；goroutine、结果、错误和完成等待均由调用方管理。
不维护 worker 或任务队列，也不启动后台 goroutine。

## 安装

```sh
go get github.com/tr1v3r/pkg/pools
```

## 快速开始

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


先获取令牌再启动 worker，能够约束 worker 的创建数量。每个 worker 写入独立结果槽位，
在 `wg.Wait` 后读取；共享 map 和并发 append 仍需另行同步。请求处理时应传入请求的 context。
如果提交途中获取失败，应等待已启动的 worker 再返回；
[批量示例](examples/batch/example.go) 用 `defer wg.Wait()` 演示了这一点。

## API

```go
type Pool interface {
    Acquire(ctx context.Context) error
    Release()
    Num() int
    Size() int
}
```

- `NewPool(size)`：正数限制并发；负数不限并发；零直接 panic。
  必须通过构造函数创建，零值不是公开 API。
- `Acquire(ctx)`：等待可用容量，或返回 context 错误。不能传 nil context。
  `context.Background()` 表示显式选择无截止时间的等待。
  已取消的 context 会直接被拒绝，即使当前有容量或处于无限并发模式。
- 每次成功获取后必须恰好调用一次 `Release()`，通常使用 `defer`；获取失败时不能释放。
  成功后的取消不会归还令牌，也不会停止任务；任务本身需要另行接收 context。
- 取消与容量可用同时发生时，两种结果都可能出现；以返回值决定是否需要释放令牌。
- 无限并发模式下 `Release()` 不做任何操作。有界模式下，无令牌可释放时 panic。
  令牌没有所有者标识，无法识别误释放其他 worker 的令牌，调用方仍须保证正确配对。
- `Num()` 返回有界池已持有令牌数，不含等待请求；`Size()` 返回容量。
  无限并发模式下两者都返回零。它们不表示任务完成状态。
- 不需要 `Close`。通过 `sync.WaitGroup` 等待任务，在启动 worker 前调用 `Add`；
  对空闲的 WaitGroup，开始完成等待前应先登记任务。池不恢复 panic，也不汇总错误。

## 取消等待

在返回 `error` 的函数中，假设已有池 `p` 和上游 context `ctx`：

```go
ctx, cancel := context.WithTimeout(ctx, time.Second)
defer cancel()
if err := p.Acquire(ctx); err != nil {
    return err // 未获取令牌，不调用 Release。
}
defer p.Release()
// 执行任务。
```

需导入 `context` 和 `time`。`Acquire` 在当前 goroutine 中等待，不额外启动等待 goroutine。
需要异步执行时由调用方启动并管理 goroutine；通常先获取再启动，可以对任务提交施加背压。

## 示例与验证

在 `pools/` 目录运行：

| 示例 | 命令 |
| --- | --- |
| [有界批量处理与有序结果](examples/batch/example.go) | `go run ./examples/batch` |
| [获取令牌超时](examples/cancellation/example.go) | `go run ./examples/cancellation` |
| [调用方管理完成通知](examples/completion/example.go) | `go run ./examples/completion` |

```sh
GOWORK=off go test -race ./...
GOWORK=off go vet ./...
```

## Breaking change 迁移

| 移除的 API | 替代方案 |
| --- | --- |
| `Wait()` | 调用 `Acquire(ctx)` 并处理错误；无限期等待使用 `context.Background()`。 |
| `WaitContext(ctx)` | `Acquire(ctx)`。 |
| `Done()` | `Release()`。 |
| `AsyncWait()` | 直接调用 `Acquire(ctx)`，或自行管理 goroutine 及返回结果。 |
| `WaitAll()` | 调用方使用 `sync.WaitGroup`。 |
| `AsyncWaitAll()` | 调用方启动 goroutine 等待 `sync.WaitGroup`，完成后关闭 channel。 |

自定义实现和 mock 需适配新的 `Pool` 接口。零容量现在会 panic。
无限并发和有界模式都不再跟踪任务完成，批次完成统一由调用方管理。
有界池无令牌可释放时现在会 panic，而非阻塞。保留 `Num` 和 `Size`。
