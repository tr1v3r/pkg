# circuitbreaker 知识库（合并版五件套）

组件：三态熔断器。从 fetch 抽出的独立包，fetch 不再内嵌使用，由调用方自行组合（见决策）。
更新日期：2026-10-12（刷新 @ d6b2b9c：功能零变化，仅 5d88794 godoc 补齐 + 9115d2d hermetic 测试 error_test.go）。证据等级：A=代码现状/可回查提交，C=推测。包地图见仓库 ARCHITECTURE.md（A 级来源，含 §4.4 本包深写）。

## 流程

```text
【状态机】(circuitbreaker.go, 单一 sync.Mutex 保护全部字段)
Closed ──failureCount≥FailureThreshold──▶ Open (记录 lastFailure=now)
Open ──time.Since(lastFailure)>OpenTimeout 且有调用──▶ HalfOpen (仅重置 successCount=0)
HalfOpen ──successCount≥SuccessThreshold──▶ Closed (failureCount/successCount 双清零)

【Execute 主路径】
Execute(fn) → Allow()? 否 → return &Error{Err: ErrOpen}（errors.Is(err, ErrOpen) 可判）
           → 是 → fn() → RecordResult(err==nil) → 返回 fn 的原始 err
Allow/RecordResult 各自独立加锁，可脱离 Execute 单独使用（fetch 风格的组合式用法依赖这点）

【各状态下记账】
Closed   : 成功→failureCount=0（连续失败语义）；失败→failureCount++
HalfOpen : 成功→successCount++，到阈值关闸；失败→failureCount++（未清零，见决策 D2）
Open     : 迟到的成功/失败结果一律丢弃（onSuccess/onFailure 无 Open 分支）
默认参数：5 次失败 / 3 次成功 / Open 30s（DefaultConfig）
```

## 决策

- D-001 单 Mutex 取代 RWMutex+atomic（2026-10-12 记录，A：aeaa5b0）。修复锁降级窗口的竞态。
  重新评估：仅当锁竞争 profiling 显示瓶颈时再考虑细分。
- D-002 HalfOpen 不限并发探针：Allow() 对 HalfOpen 一律放行，多个调用者可同时探测（2026-10-12，A：代码现状）。
  与 gobreaker 系「单探针」实现不同。动机 (C)：推测为保持实现简单。重新评估：若下游无法承受并发试探，需加在途计数。
- D-003 进入 HalfOpen 时不清零 failureCount：任何一次失败立即重新 Open（2026-10-12，A：代码现状）。
  副作用：HalfOpen 下成功也不重置 failureCount，只有凑满 SuccessThreshold 关闸才清。
- D-004 独立包而非 fetch 内嵌；fetch 主代码不再引用，doc.go 示例教用户 cb.Execute 包住 fetch.Get（2026-10-12，A：ca009f2 提交信息）。

## 历史

- H-001 诞生于 fetch（161ea9d），2026-04-09 同日两连击：aeaa5b0 修竞态（RWMutex 降级窗口 + 混用 atomic 导致 UB）→ ca009f2 抽包并改 API 名（CircuitBreakerConfig→Config、CircuitBreakerError/ErrCircuitOpen→Error/ErrOpen），动机明写「for reuse beyond HTTP (DB, gRPC, Redis)」（A：提交信息）。当前影响：fetch 侧错误类型判断需用新包符号。
- H-002 模块路径 github.com/riverchu/pkg → github.com/tr1v3r/pkg（2026-10-12 记录，A：00887d8, 2023-08-15）。当前影响：旧 import 路径已失效。

## 待办

- T-001 HalfOpen 并发探针无上限（状态：候选未验证，2026-10-12，A：代码现状）。影响：弱下游恢复期可能被并发试探压垮。完成条件：维护者确认是否需要单探针语义。
- T-002 Config 零值语义未防护（状态：候选未验证，2026-10-12，A：代码现状）。阈值/超时为 0 时行为退化为「一次失败即开、瞬间半开」。完成条件：New() 拒绝零值或落默认值。

## 验收案例

- S-001 闭环熔断：Closed 下连续 FailureThreshold 次失败 → State()==StateOpen；Execute 返回的 err 满足 errors.Is(err, ErrOpen)。验证：State()/FailureCount() 读数。
- S-002 恢复闭环：Open 后等 > OpenTimeout，下一次 Execute 被放行且 State 变 HalfOpen；连续 SuccessThreshold 次成功后回 Closed，两计数清零。验证：SuccessCount() 中间读数。
- S-003 半开即断：HalfOpen 下 1 次失败 → 立即回 Open（failureCount 未清零直接过阈）。验证：State() 与 lastFailure 更新（经 Allow 再次放行时间推后）。
- S-004 迟到结果丢弃：探针 A 失败使状态回 Open 后，并发探针 B 的成功结果不改变 Open 状态。验证：并发执行后 State() 仍为 Open。
- S-005 连续性语义：Closed 下「失败×(阈值-1) → 成功 1 次 → 失败×(阈值-1)」不触发开闸。验证：State() 全程 Closed。
