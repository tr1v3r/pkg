# sync.Map 与当前双锁 TTL 缓存的隔离实验

结论：`sync.Map + Set 时替换节点` 在多核读密集负载下有明显吞吐收益，但覆盖写入更慢且增加分配，保留内存约翻倍；在本次写密集负载及单核单负载goroutine配置中，当前双锁更好。本报告保存测量阶段的选型证据，表中的“当前双锁”均指固定实验基线。

实验日期：2026-10-15。归档时补充了负载goroutine数量与结论边界说明，实验实现和测量数据未改变。

## 后续选型决策（2026-10-15）

用户确认采用 `sync.Map + Set替换节点`：优先考虑多核读密集负载和同步结构简化，
接受低核低并发下的耗时增加，以及覆盖写入的分配与保留内存成本。
正式实现保留一把堆锁协调写入/清理，Get不获取该锁。
以下原始数据、固定基线和复现包不变；采纳此方案不代表所有负载都更快。

## 实验约束与复现

- 基线：PR #53 `ddb9e7c315c18f908e7ca13c73d43bb26d13ee74` 的真实 `cache.go` / `heap.go`，未修改。
- 候选：仅将索引改为 `sync.Map`，移除 mapMu；写入/删除/清理仍由 heapMu 串行。Set 新建节点，覆盖时按旧 index 替换堆节点再 Fix；发布后的 value/deadline 不再修改，index 只由堆锁保护。
- 两者均为每 key 一个堆节点、相同 TTL/通知逻辑，普通基准均启动后台 worker，TTL=24h。int key / int value；未测字符串、大对象或实际业务分布。
- Apple M3 Pro，darwin arm64，12逻辑CPU。Go 1.24.13 测 GOMAXPROCS=1/12，Go 1.25.14 复查 GOMAXPROCS=12。
- 各微基准6轮，逐轮交替两实现运行顺序，单样本150ms；以中位数汇报。统一接口调用两实现，存在共同的接口调用开销。没有同时运行其他实验；-race只用于正确性验证。
- 并行项目的 ns/op 是整个负载耗时除以操作数，即吞吐折算，**不是单次请求延迟**。每 worker 本地随机数，无每操作全局计数器。
- 内存样本3轮，强制GC后记录10万条缓存相对于构建前的HeapAlloc差值，属于进程级近似。

### CPU并行度与负载goroutine数量

本次没有调用 `b.SetParallelism`。Go 1.24 的 `testing.B.RunParallel` 默认创建
`GOMAXPROCS` 个负载goroutine（实现中为 `b.parallelism * runtime.GOMAXPROCS(0)`，
默认 `b.parallelism=1`）。参见 [Go 1.24.13 源码](https://github.com/golang/go/blob/go1.24.13/src/testing/benchmark.go#L951)。

| 测量类别 | GOMAXPROCS | 执行读写负载的goroutine |
| --- | ---: | ---: |
| ReadParallel / Mixed，单核配置 | 1 | **1** |
| ReadParallel / Mixed，多核配置 | 12 | **12** |
| GetHit / GetMiss / Overwrite / SetDelete | 1或12 | 始终1个串行负载goroutine |
| 集中过期持续读取实验 | 12 | 固定12个读取goroutine，另外执行清理 |

普通微基准中，每个缓存还另有1个后台清理worker；上表不是进程全部goroutine数量。
`GOMAXPROCS=1` 限制Go代码的执行并行度，不代表把进程绑定到某一个物理CPU核。

因此，**“95%读时sync.Map耗时约2.1倍”对应1个P、1个负载goroutine，
不是1个P、12个负载goroutine**。本次不同CPU配置还同时改变了并行负载goroutine数，
不能把两组差异全部归因于CPU核数。尚未测量“固定12个负载goroutine、GOMAXPROCS=1”的情况。

### 复现包

- [reproducer.tar.gz](reproducer.tar.gz)：独立、仅依赖标准库的Go模块，包含固定基线、候选、测试、运行及汇总脚本。
- [raw/micro124.txt](raw/micro124.txt)、[raw/micro125.txt](raw/micro125.txt)：六轮普通基准完整输出。
- [raw/expiry-footprint124.txt](raw/expiry-footprint124.txt)、[raw/expiry-footprint125.txt](raw/expiry-footprint125.txt)：集中过期及内存原始输出。
- [raw/alloc-top124.txt](raw/alloc-top124.txt)：分配profile摘要；原始profile同目录保留。
- [SHA256SUMS](SHA256SUMS)：报告、原始数据和复现包的校验值。

先在本目录校验并解包，再执行后面的Go命令：

```sh
shasum -a 256 -c SHA256SUMS
mkdir -p /tmp/cache-syncmap-repro
tar -xzf reproducer.tar.gz -C /tmp/cache-syncmap-repro
cd /tmp/cache-syncmap-repro/syncmap-2026-10-15
```

以下路径和命令均相对于解包后的独立模块，不是当前仓库的生产包。

主要文件：`cache/syncmap_experiment_test.go` 候选实现，`cache/syncmap_comparison_test.go` 公共基准/契约验证，`cache/syncmap_expiry_experiment_test.go` 持续读取期间清理测试，`cache/syncmap_scheduler_experiment_test.go` Go1.25虚拟时间验证。`raw/` 内保留完整输出，`run_micro.py` 交替运行顺序。

```sh
go test -race ./cache -run '^TestExperiment(Model|Concurrent|SyncMapSnapshots|SyncMapScheduler)$' -count=10
go test -c -o cache.test ./cache
python3 run_micro.py ./cache.test 6 micro.txt 1,12
go test ./cache -run '^TestExperiment(Footprint|ExpiryContinuous|Environment)$' -v -count=1
```

分别用对应Go版本运行。旧版本自动排除Go1.25虚拟时间文件。计时不使用-race；完整实验源码和原始输出一并保留。

## 普通操作与并发吞吐

越小越好；每格为6轮中位数，单位ns/op。括号为候选/基线，低于1表示候选更快。

### Go 1.24.13 / GOMAXPROCS=12

| 负载 | 当前双锁 | sync.Map替换节点 | 候选/基线 |
| --- | ---: | ---: | ---: |
| GetHit1024 | 38.38 | 40.02 | 1.04× |
| GetMiss1024 | 7.33 | 6.84 | 0.93× |
| ReadParallel65536 | 103.60 | 17.57 | 0.17× |
| Overwrite1 | 106.15 | 164.20 | 1.55× |
| Overwrite1024 | 277.80 | 351.30 | 1.26× |
| Overwrite65536 | 378.70 | 493.40 | 1.30× |
| SetDelete1024 | 115.95 | 142.85 | 1.23× |
| Mixed95Read65536 | 116.55 | 52.20 | 0.45× |
| Mixed50Read65536 | 181.70 | 317.45 | 1.75× |
| Mixed95ReadHot64 | 91.66 | 34.42 | 0.38× |

### Go 1.25.14 / GOMAXPROCS=12

| 负载 | 当前双锁 | sync.Map替换节点 | 候选/基线 |
| --- | ---: | ---: | ---: |
| GetHit1024 | 39.30 | 41.03 | 1.04× |
| GetMiss1024 | 7.41 | 6.45 | 0.87× |
| ReadParallel65536 | 106.95 | 22.41 | 0.21× |
| Overwrite1 | 96.68 | 155.80 | 1.61× |
| Overwrite1024 | 264.95 | 342.20 | 1.29× |
| Overwrite65536 | 368.95 | 485.30 | 1.32× |
| SetDelete1024 | 116.25 | 144.15 | 1.24× |
| Mixed95Read65536 | 122.10 | 53.10 | 0.43× |
| Mixed50Read65536 | 183.10 | 340.00 | 1.86× |
| Mixed95ReadHot64 | 89.51 | 32.33 | 0.36× |

### Go 1.24.13 / GOMAXPROCS=1，RunParallel负载goroutine=1

| 负载 | 当前双锁 | sync.Map替换节点 | 候选/基线 |
| --- | ---: | ---: | ---: |
| GetHit1024 | 38.14 | 40.15 | 1.05× |
| GetMiss1024 | 7.32 | 7.88 | 1.08× |
| ReadParallel65536 | 51.36 | 86.23 | 1.68× |
| Overwrite1 | 64.66 | 108.95 | 1.68× |
| Overwrite1024 | 161.75 | 268.50 | 1.66× |
| Overwrite65536 | 232.10 | 445.00 | 1.92× |
| SetDelete1024 | 128.25 | 158.30 | 1.23× |
| Mixed95Read65536 | 55.40 | 118.70 | 2.14× |
| Mixed50Read65536 | 91.57 | 291.65 | 3.18× |
| Mixed95ReadHot64 | 41.13 | 45.12 | 1.10× |

ReadParallel / Mixed的键由各worker伪随机选择。Overwrite循环更新已有key，可能频繁改变堆顶；Mixed95Read=95% Get、5%覆盖Set，Mixed50Read=各50%；Hot64只在64个key间操作，其他混合负载65536个key。SetDelete一次操作是一对Set+Delete，其他每次是一个API调用。命中均校验值，覆盖写入保持条目数不变。

## 分配与GC

Go1.24.13 / GOMAXPROCS=12，中位数。GC/Mop为每百万操作期间发生的进程级GC次数估计，不是GC CPU占比。

| 负载 | 双锁B/op | sync.Map B/op | 双锁allocs/op | sync.Map allocs/op | 双锁GC/Mop | sync.Map GC/Mop |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Overwrite1 | 0.0 | 96.0 | 0.0 | 2.0 | 0.00 | 27.00 |
| Overwrite1024 | 0.0 | 102.0 | 0.0 | 2.0 | 0.00 | 29.96 |
| Overwrite65536 | 0.0 | 103.5 | 0.0 | 2.0 | 0.00 | 11.25 |
| SetDelete1024 | 48.0 | 104.0 | 1.0 | 3.0 | 13.32 | 29.27 |
| Mixed95Read65536 | 0.0 | 5.0 | 0.0 | 0.0 | 0.00 | 0.59 |
| Mixed50Read65536 | 0.0 | 51.0 | 0.0 | 1.0 | 0.00 | 5.59 |

混合读写的allocs/op由Go基准输出取整，0不代表写入不分配；请结合B/op看。覆盖写入候选约2次分配/操作，当前实现原地更新为0。分配profile单独保存，未与其他计时实验并行；profile显示候选Set及sync.Map内部newEntryNode是主要分配来源，不能把候选成本理解成只新建一个缓存节点。

### 10万条保留内存

| Go版本 | 双锁中位数 | sync.Map中位数 | 比值 |
| --- | ---: | ---: | ---: |
| 124 | 8.127 MB (81.3 B/key) | 17.551 MB (175.5 B/key) | 2.16× |
| 125 | 8.127 MB (81.3 B/key) | 17.557 MB (175.6 B/key) | 2.16× |

## 持续读取期间集中过期

12个读取goroutine持续读取一个存活key，另一个goroutine执行生产nextDelay，清理1万/10万条到期记录。
同一负载运行5波，两实现交替顺序；没有timer worker、没有Set，直接构建有效堆，隔离清理和Get竞争。
与先前“只在清理开始时排队32个读者”的探针不同，这次覆盖整个清理过程。为了不按请求分配内存，分位数用每个goroutine独立的16子桶/二倍区间直方图近似，上界误差最多约6.25%。
下面P99是**清理期间开始的Get请求**，不是完整业务P99。清理时间含读写竞争和调度，不是纯堆运算时间。
所有样本同时记录最大延迟和慢请求比例，避免只看P99；max取5波最大值，其余指标取各波指标的中位数。

### Go 124

| 分布/数量 | 实现 | 清理耗时 | Get P99上界 | 最大全部观测 | Get >10µs比例 | Get >1ms比例 |
| --- | --- | ---: | ---: | ---: | ---: | ---: |
| identical/10000 | DoubleLock | 55.489 ms | 86.015 µs | 562.208 µs | 5.5262% | 0.0000% |
| identical/10000 | SyncMap | 1.877 ms | 4.863 µs | 1.702 ms | 0.0081% | 0.0000% |
| identical/100000 | DoubleLock | 574.686 ms | 86.015 µs | 2.030 ms | 5.5588% | 0.0001% |
| identical/100000 | SyncMap | 28.218 ms | 3.711 µs | 11.205 ms | 0.0154% | 0.0003% |
| staggered/10000 | DoubleLock | 61.053 ms | 90.111 µs | 603.750 µs | 5.7263% | 0.0000% |
| staggered/10000 | SyncMap | 2.236 ms | 3.839 µs | 2.228 ms | 0.0218% | 0.0014% |
| staggered/100000 | DoubleLock | 644.444 ms | 90.111 µs | 2.779 ms | 5.8100% | 0.0000% |
| staggered/100000 | SyncMap | 46.719 ms | 4.863 µs | 23.454 ms | 0.0354% | 0.0012% |

### Go 125

| 分布/数量 | 实现 | 清理耗时 | Get P99上界 | 最大全部观测 | Get >10µs比例 | Get >1ms比例 |
| --- | --- | ---: | ---: | ---: | ---: | ---: |
| identical/10000 | DoubleLock | 57.839 ms | 73.727 µs | 3.010 ms | 5.0625% | 0.0013% |
| identical/10000 | SyncMap | 1.856 ms | 0.303 µs | 2.251 ms | 0.0199% | 0.0005% |
| identical/100000 | DoubleLock | 587.733 ms | 77.823 µs | 5.094 ms | 4.9150% | 0.0013% |
| identical/100000 | SyncMap | 13.220 ms | 4.095 µs | 9.194 ms | 0.0159% | 0.0015% |
| staggered/10000 | DoubleLock | 60.944 ms | 81.919 µs | 1.717 ms | 5.2343% | 0.0000% |
| staggered/10000 | SyncMap | 2.296 ms | 4.607 µs | 2.321 ms | 0.0947% | 0.0051% |
| staggered/100000 | DoubleLock | 640.963 ms | 77.823 µs | 6.067 ms | 4.9287% | 0.0009% |
| staggered/100000 | SyncMap | 49.899 ms | 0.303 µs | 30.509 ms | 0.0112% | 0.0005% |

sync.Map在此场景的P99和清理速度更好，但最大观测延迟仍有毫秒级尖峰，部分样本高于双锁。测试没有定位这些尖峰到调度、GC或内部实现，因此不能把P99改善解读为更低的最大延迟保证。

## 验证与结论边界

候选通过随机操作模型、一key一节点/堆索引不变量、并发读写删除清理、并发关闭、不可变快照、泛型nil/NaN验证，以及Go1.25虚拟时间下提前到期/续期/删除重插/真实清理回归。Go1.25十轮竞态验证、Go1.24三轮竞态验证均保存输出。

数据支持按负载选择：

- 多核、读占比高：sync.Map值得继续评估，可去掉显式map锁；本次95/5与纯读吞吐明显改善。
- 写入/续期频繁、重视内存，或本次单核单负载goroutine配置：当前双锁更有优势。候选覆盖写入更慢，有每次约百字节分配，保留内存约2倍。
- 当前样本不足以确定真实业务的临界读写比例，没有测实际业务访问轨迹、字符串/大型value、其他架构或GC配置，也没有做统计显著性检验。

本次测量没有替换生产缓存实现。后续应补充固定12个负载goroutine、分别设置GOMAXPROCS=1/12的对照，以区分CPU并行度和请求并发数；新实验应保留独立记录，不覆盖本次数据。
