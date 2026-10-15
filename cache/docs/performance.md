# 集中过期与读取阻塞测量（历史双锁方案）

测于 2026-10-15，Apple M3 Pro / darwin arm64，Go 1.24.13，GOMAXPROCS=12。
基线为 PR #53 的 `a41a5cbdb189cfa71878ccd590e554af2a8ef82f`；对比版本使用
独立 heap/map 锁，以及仅最早 deadline 变化时通知 worker。

当前实现已改用sync.Map；本页保留原实验记录，数值不代表当前实现。
后续对照与采纳理由见 [sync.Map选型实验](experiments/syncmap-2026-10-15/README.md)。

## 方法与边界

历史脚本为 [`expiry_contention_test.go`](https://github.com/tr1v3r/pkg/blob/ddb9e7c315c18f908e7ca13c73d43bb26d13ee74/cache/expiry_contention_test.go)。
该脚本依赖当时的mapMu，仅应在提交 `ddb9e7c315c18f908e7ca13c73d43bb26d13ee74`
的独立checkout中运行以下命令，当前源码不再包含它：

```sh
go test -tags=cacheperf ./cache -run '^TestExpiryContentionMeasurement$' -count=3 -v
go test ./cache -run '^$' -bench . -benchmem -count=3
```

- 每种情况运行 20 个清理波次，每次 32 个读取同一存活 key 的请求，共 640 次窗口内读取。
- 提前构建已过期的有效最小堆，直接调用生产实现 `nextDelay`；不启动 timer worker，
  避免将填充和 timer 调度延迟混入结果。基线使用同一脚本，将 `mapMu` 换为原来的 `mu`。
- `identical` 的过期时间完全相同；`staggered` 按 key 顺序相差 1 ns，均已过期，
  模拟一次写入波次的 deadline 次序。相同 deadline 的堆调整可能更少，两种情况应分别看。
- 用一个已有读锁让清理的 map 写锁进入等待，再启动 32 个读取并释放原读锁。
  **这里的 P99 只属于刻意与清理重叠的请求，不是全业务 P99。** 读耗时包含短暂的
  屏障协调和调度；N=0 的基线对照为约十到几十微秒，不能将微秒结果当成纯 Get 成本。
- 清理耗时从释放屏障到 `nextDelay` 返回，包含锁竞争和调度，不是严格的锁持有时间。
- 每次运行另测 100,000 次无清理读取，其 P50 约 42 ns、P99 通常约 84 ns；逐次计时
  自身有开销。下表取三次运行各自 P99 / 清理 P50 的中位数，不是合并样本的分位数。

## 集中过期结果

| deadline 分布 | 到期条数 | 原实现窗口 Get P99 | 拆锁后窗口 Get P99 | 原实现清理 P50 | 拆锁后清理 P50 |
| --- | ---: | ---: | ---: | ---: | ---: |
| identical | 1,000 | 64.5 µs | 21.1 µs | 0.044 ms | 0.067 ms |
| identical | 10,000 | 521.3 µs | 23.1 µs | 0.419 ms | 0.577 ms |
| identical | 100,000 | 5234.8 µs | 29.8 µs | 4.791 ms | 6.156 ms |
| staggered | 1,000 | 141.2 µs | 19.3 µs | 0.122 ms | 0.133 ms |
| staggered | 10,000 | 1553.5 µs | 22.4 µs | 1.476 ms | 1.553 ms |
| staggered | 100,000 | 23699.8 µs | 72.3 µs | 21.227 ms | 18.893 ms |

拆锁让读取能够在 map 删除间隙继续执行；它没有消除整批清理的工作量，
也没有消除写操作对 heap 锁的等待。测试没有提供固定延迟上界或生产负载承诺。

## 普通操作基准

下列为三次运行的 ns/op 中位数，前后保留相同的既有 benchmark 及长 TTL：

| 操作 | 原实现 | 拆锁与通知优化后 |
| --- | ---: | ---: |
| GetHit | 39.92 | 37.84 |
| GetMiss | 6.70 | 6.42 |
| GetParallel | 104.90 | 99.63 |
| Overwrite/Entries=1 | 86.10 | 104.60 |
| Overwrite/Entries=1024 | 273.20 | 279.70 |
| Overwrite/Entries=65536 | 336.60 | 360.40 |
| SetDelete | 190.40 | 114.10 |

覆盖写入有额外加锁与 deadline 比较成本；非堆顶更新不再唤醒 worker，减少部分调度开销。
这些样本用于观察取舍，未做统计显著性检验。真实场景仍需按缓存大小、读写比例、
过期聚集程度以及 CPU/GC 压力测量。
