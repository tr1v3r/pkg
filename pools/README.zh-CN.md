# pools

[English](README.md) | [简体中文](README.zh-CN.md)

仅依赖 Go 标准库的轻量令牌池，用于限制并发工作量。
goroutine 由调用方启动，池负责获取令牌和等待完成；不维护可复用 worker 或任务队列。

## 安装

需要 Go 1.24 或更高版本。

```sh
go get github.com/tr1v3r/pkg/pools
```

## 快速开始

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
        p.Wait() // 先获取令牌，再启动 worker。
        go func() {
            defer p.Done()
            results[i] = value * value
        }()
    }
    p.WaitAll()
    fmt.Println(results) // [1 4 9 16 25]
}
```

在提交任务的 goroutine 中获取令牌，可以同时约束 worker 创建数量和实际并发量。
示例中每个 worker 写入独立的结果槽位，并在全部完成后读取。
共享 map、并发 append 等操作仍需调用方自行同步。

## API

| 方法 | 行为 |
| --- | --- |
| `NewPool(size)` | 返回 `Pool`；限制并发时应使用正容量。 |
| `Wait()` | 阻塞直到获取一个令牌。 |
| `AsyncWait()` | 在新 goroutine 中获取令牌；成功后关闭返回的 channel。 |
| `Done()` | 归还一个令牌，并标记该工作单元完成。 |
| `Num()` | 当前已持有令牌数的快照，不包含阻塞中的获取请求。 |
| `Size()` | 令牌 channel 的容量；配置为零或负数时均返回零。 |
| `WaitAll()` | 等待已登记的工作完成，调用顺序要求见下文。 |
| `AsyncWaitAll()` | 在新 goroutine 中执行 `WaitAll`，返回后关闭通知 channel。 |

## 调用顺序与边界

- 每次成功获取令牌都必须对应一次 `Done()`，通常在 worker 内使用 `defer`。
  多余的 `Done()` 可能阻塞，也可能消耗其他 worker 的令牌。
- `WaitAll` 遵循 `sync.WaitGroup` 的调用顺序要求：先完成本批次任务提交，再等待。
  不要把 `Wait()` 放入刚启动的 worker 后立即调用 `WaitAll()`，否则可能因任务尚未登记而提前返回。
- `AsyncWait()` 返回不代表获取请求已经登记或完成。必须先接收其通知，再执行对应任务，
  或依赖完成等待来覆盖该任务。
- 异步通知不提供取消能力。放弃接收 `AsyncWait()` 的通知，可能遗留等待中的 goroutine，
  或导致已获取的令牌无人归还。给 `AsyncWaitAll()` 的接收操作设置超时，也不会取消内部等待。
- 容量为 **零** 时创建无缓冲令牌 channel，常规的 `Wait → 执行任务 → Done` 流程会阻塞在 `Wait`；
  零不表示无限并发。
- 容量为 **负数** 时，同时禁用并发限制和完成跟踪。`Wait`、`Done` 不做任何操作，
  `WaitAll` 也不会等待调用方的 goroutine。
- 正容量池可在上一批次完成等待返回后复用。没有 `Close` 方法，也没有需要关闭的常驻 worker。
- 池不收集结果或错误、不恢复 panic，也不取消任务。`Num()` 仅供观察，不能用作同步屏障。

## 可运行示例

在本模块目录（`pools/`）中执行：

| 示例 | 命令 | 内容 |
| --- | --- | --- |
| [批量处理](examples/batch/example.go) | `go run ./examples/batch` | 限制 worker 数量，通过 `WaitAll` 等待并按输入顺序收集结果。 |
| [异步获取令牌](examples/async-acquire/example.go) | `go run ./examples/async-acquire` | 等待令牌期间做其他工作，接收通知后使用并归还令牌。 |
| [异步等待完成](examples/async-wait/example.go) | `go run ./examples/async-wait` | 继续做其他工作，再接收整批任务的完成通知。 |

脱离仓库 workspace 独立验证本模块：

```sh
GOWORK=off go test -race ./...
GOWORK=off go vet ./...
```
