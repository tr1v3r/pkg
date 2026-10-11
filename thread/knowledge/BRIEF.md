# thread 知识库（合并版五件套）

组件：带停止语义的 worker 池（TimeoutPool）：中央 dispatch 分发任务，worker 归还自身，Terminate 优雅停机；Job.Timeout 逐任务限时（2026-08-24 起）。曾长期冻结（2023-06~2026-08），#23 解冻。
更新日期：2026-10-12（刷新 @ d6b2b9c）。证据等级：A=代码现状/可回查提交，C=推测。包地图与分层见仓库 ARCHITECTURE.md（A 级来源）。

## 流程

```text
【启动与分发】(pool.go)
NewTimeoutPool(workerN, jobQueueN) → start(): 预启 workerN 个 worker + 1 个 dispatch 协程
worker 循环: 自己入 workerQueue（宣告空闲）→ select { jobChannel 收任务 → execute(job) | stop }
dispatch 循环: select { jobQueue 取 Job → 等空闲 worker → 投递其 jobChannel（无缓冲）
                          | stop → 逐个收齐全部空闲 worker（等在途任务做完）→ 停止握手 → terminated 发信号，return }

【Job 执行与限时】worker.execute（9115d2d）
Timeout≤0 → 直接跑 Handler → jobRet 发完成票（旧行为）
Timeout>0 → 起协程跑 Handler（done chan 缓冲=1，晚完成的发送永不阻塞）
  select { done → 记账 | time.After(Timeout) → 放弃该协程，worker 立即归还 }
  jobRet 恰好发一次（完成或超时都发）→ 池的完成计数保持正确
  ⚠ Go 杀不掉 goroutine：被放弃的 Handler 继续在后台跑到自然返回（其完成被丢弃）——
    超时保护的是池吞吐，不是调用方资源

【提交与等待】
Submit(job): job 入 jobQueue（满则阻塞=背压）+ lock 下 jobCount++
StartAndWaitUntilTerminated(): 数 jobRet 完成票到 jobCount → true；中途 terminated → false
StartAndWait(timeout): 同上，内层 select 加 time.After(timeout) 分支 → 超时 false
Terminate(): 向 stop 发信号（dispatch 收到后走优雅停机路径）

【硬约束】StartAndWait* 必须单一 goroutine 调用，且 Submit 应在 start 前完成——
jobCount 的读取无锁，与并发 Submit 存在数据竞态（源码注释明示）
```

## 决策

- D-001 「worker 归还自身 + 中央 dispatch」经典模型：调度点集中，worker 无需自己抢队列（A：代码现状）。
- D-002 优雅停机 = 等所有 worker 空闲：在途任务执行完毕才停；但 jobQueue 里尚未分发的任务被直接丢弃（dispatch return 后无人消费）（A：代码现状）。动机 (C)：推测视为「停止即不再接新活」。重新评估：需要排空语义时须先改 dispatch 停止路径。
- D-003 单一等待者约束：源码注释明写竞态问题，属有意暴露的使用契约而非待修 bug（A：代码现状注释）。
- D-004 StartAndWait 的 timeout 实为「相邻两张完成票之间的最大间隔」而非总时长：time.After 在循环内每次新建，且外层 select 带 default 永不阻塞（A：代码现状）。影响：只要任务以小于 timeout 的间隔持续完成，整体永不超时。注意与 Job.Timeout 区分：后者是真 deadline（D-005）。
- D-005 Job.Timeout = 「释放 worker、放弃协程」而非取消：超时只让池继续派活；Handler 本身无 ctx 可打断，泄漏的协程跑完为止（A：9115d2d 提交信息 "Go cannot kill goroutines… timeouts protect the pool's throughput, not the caller's resources"，字段注释原文）。护栏：不要假设超时后 Handler 副作用停止（写盘/发请求仍在进行）。

## 历史

- H-001 init code 引入（f8e25d4, 2022-07-30）；166a3b4（2023-06-29）仅消除中间变量等风格调整（A：diff）。
- H-002 解冻：9115d2d（2026-08-24，#23）落地 Job.Timeout（提交信息 "enforce per-job timeouts so TimeoutPool lives up to its name"）+ pool_test/terminate_wait_test 大幅扩充；包从「名不副实」变为限时语义真实（A：提交信息）。

## 待办

- T-001 StartAndWait timeout 语义失真（状态：候选未验证，A：代码现状，见 D-004）。Job.Timeout 落地后此待办的紧迫性下降（长任务可逐个限时兜底），但等待侧语义未变。完成条件：确认意图；若要总超时需把 timer 提到循环外。
- T-002 停止时排队任务静默丢弃（状态：候选未验证，A：代码现状）。影响：调用方以为已提交就会被执行。完成条件：文档化或实现排空。
- T-003 NewTimeoutPoolWithDefaults 注释失真：注释写 10 协程/1000 队列，常量实为 20/10000（状态：候选未验证，A：代码现状，d6b2b9c 复核仍未修）。
- T-004 Job 用 ...interface{} 传参（前泛型风格），类型安全靠调用方自觉（状态：候选未验证，A：代码现状）。

## 验收案例

- S-001 全量执行：提交 K 个带计数副作用的 Job → StartAndWaitUntilTerminated 返回 true，K 个副作用全部发生。验证：原子计数器 == K。
- S-002 优雅停机：长任务运行中 Terminate → StartAndWait* 返回 false，在途任务副作用仍完整（worker 等空闲才停）。验证：任务内 progress 标记。
- S-003 停机丢弃排队：灌满 jobQueue 后立即 Terminate → 返回 false，部分已提交 Job 的 handler 从未执行。验证：计数 < 提交数。
- S-004 背压：jobQueue 容量 Q、worker 全忙时，第 Q+1 个 Submit 阻塞直至出现空位。验证：Submit 前后打点观测阻塞。
- S-005 间隔超时语义：任务分别每（timeout−ε）完成一张票，远超 timeout 的总时长后 StartAndWait(timeout) 仍不返回 false（印证 D-004；修复后此案例应反转）。验证：总耗时计时。
- S-006 逐任务超时释放 worker（已有测试锁定 TestJobTimeout_*）：Handler 睡 200ms、Timeout=50ms → worker 立即可接新任务，jobRet 计数不丢；Timeout=0 不限时；长 deadline 不误触发。验证：后续任务完成时间 + 最终计数（pool_test.go）。
