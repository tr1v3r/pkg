# pools 知识库（合并版五件套）

组件：基于 channel 的令牌池（计数信号量）。不是执行任务的 goroutine 池——不接收函数、不代起协程。
更新日期：2026-10-12（刷新 @ d6b2b9c）。证据等级：A=代码现状/可回查提交，C=推测。包地图见仓库 ARCHITECTURE.md（A 级来源）。

## 流程

```text
【令牌生命周期】(pool.go)
NewPool(size) → size≥0: make(chan struct{}, size)；size<0: pool 字段为 nil（不限流模式）
Wait()   : wg.Add(1) 先行 → 向 chan 发令牌 → 满则阻塞（获得 = 持有）
Done()   : <-chan 读走令牌 + wg.Done()（归还；多调于 Wait 会 panic 负计数）
WaitAll(): wg.Wait() —— 等所有「已进入 Wait 的调用」完成配对的 Done
AsyncWait/AsyncWaitAll: 起协程包同步版，返回 close 时机 = 完成/全归还 时刻的信号 chan

【观测】Num()=len(chan)=在途令牌数；Size()=cap(chan)=容量
【nil 安全】*pool 为 nil 或 chan 为 nil 时所有方法安全退化为 no-op（AsyncWait 返回已关闭 chan）
```

## 决策

- D-001 定位是信号量而非任务池：调用方自己起 goroutine，Wait/Done 只管并发上限（2026-10-12，A：代码现状；接口注释却自称 "goroutine pool"，名实不符处以代码为准）。
  重新评估：若要任务式 API，属新增接口而非改造本包。
- D-002 wg.Add(1) 在阻塞获取之前执行：被阻塞等令牌的调用者也被 WaitAll 统计（A：代码现状）。动机 (C)：推测为保证「所有想用池的人做完事」才返回。重新评估：若只需等「已拿到令牌者」，需把 Add 挪到获取之后。⚠ 2026-08-24 起 WaitAll 的 godoc 把这条写成显式契约（"Like sync.WaitGroup, all Wait() calls that begin while the pool is idle must happen before WaitAll is called, or WaitAll may return early"）——Add-before-Wait 从「实现细节」升格为「文档化使用契约」（A：b1d70ff）。
- D-003 负 size = 关闭限流：Init 对 size<0 不建 chan，Wait 直通、Num 恒 0（A：代码现状；是否有意为之 (C)）。
- D-004 无 Close/销毁 API：池生命周期 = 最后一个 Done 归还，无显式终止路径（A：代码现状）。
- D-005 全方法 nil-receiver 防护（A：b7685a2，2025-11-17）。

## 历史

- H-001 init code 引入（f8e25d4, 2022-07-30），此后近十年仅两类改动：格式化（0f0aa55）与 b7685a2 补 nil 防护 + 测试覆盖 0→100%（提交信息明写）。当前影响：包处冻结态，改动风险低但无人带路，先读测试。
- H-002 cb007c4（2026-04-09）仅给 pool_test.go 补行尾换行（A：diff），不是功能演进。
- H-003 b1d70ff/#24（2026-08-24）修测试竞态：TestPoolConcurrent 曾在收尾令牌未归还时就断言 Num()==0（慢 CI 必红）；TestPoolWaitAll 曾犯 wg.Add 与 wg.Wait 并发的经典竞态。两者改用 WaitAll 屏障 + started chan 保证 Add-before-Wait，-count=10 -race 验证（A：提交信息）。当前影响：并发语义测试可信；WaitAll 契约见 D-002。

## 待办

- T-001 接口注释失真：「goroutine pool」「acquires a pool token (blocking or async)」等注释与信号量实义仍有距离（5d88794 godoc 补齐后措辞统一为 token，但接口名 Pool/Wait 的任务池误导仍在）（状态：候选未验证，A：代码现状）。影响：误导新用户找 Submit 类 API。完成条件：维护者确认注释修订。
- T-002 Wait 不支持 context 取消：池满时调用者只能永久阻塞（状态：候选未验证，A：代码现状）。影响：无法优雅放弃排队。完成条件：决定是否加 WaitContext。
- T-003 Size() 对 nil/负 size 返回 0，调用方无法区分「容量 0」与「未初始化」（状态：候选未验证，A：代码现状）。

## 验收案例

- S-001 并发上限：Size=N 的池，M(>N) 个 goroutine 同时 Wait → Num() 稳定为 N，其余 M-N 个阻塞；逐个 Done 后全部放行。验证：Num() 采样 + 各 goroutine 完成计数。
- S-002 WaitAll 语义：N 个 Wait 全部 Done 后 WaitAll 返回；存在阻塞等令牌者时（其 wg.Add 已执行）WaitAll 不返回。验证：配合 timeout 检测 WaitAll 阻塞。
- S-003 不限流模式：NewPool(-1) 下任意并发 Wait 立即返回、Num() 恒 0、WaitAll 随 Done 完成。验证：压力循环无阻塞。
- S-004 nil 池安全：nil *pool 上调用任意方法不 panic，AsyncWait 返回的 chan 已关闭。验证：直接断言无 panic + chan 可立即读出。
- S-005 AsyncWait 时序：返回的 chan 在令牌真正获取后才 close（而非调用即 close）。验证：满池时 select 该 chan 与 Done 触发的先后。
