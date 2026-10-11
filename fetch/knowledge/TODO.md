# fetch — 待办事项

更新：2026-10-12（刷新 @ d6b2b9c）· 待办是背景，不是本次任务的授权清单；执行前需维护者确认。

## T-001：WithLogging / WithMetrics 是半成品

状态：已确认待处理
问题：WithLogging 的 LogRequest 调用被注释，method/url 传空串、ctx 用
context.Background()；WithMetrics 的 method/url 也传空串（middleware.go:26-56）。
证据（A）：代码现状。
影响：这两个中间件当前不产出可用日志/指标，仅计时正确。
下一步：随 D-001 的签名重评估一起处理——签名不改则无法取到请求信息。
完成条件：中间件能输出真实 method/url，或导出层明确标注实验性。

## T-002：errors.go 三个哨兵无引用 + base.go 遗留类型

状态：候选未验证
问题：`ErrRateLimited` / `ErrMaxRetries` / `ErrInvalidResponse` 在包内无非测试引用
（重试失败实际返回 `*RetryableError`）；base.go 的 JSONResult 族同样零引用。
证据（A）：grep 代码现状；(C) 推测为 161ea9d 重写期与 2023 年 API 客户端时代的遗留。
影响：误导使用者 errors.Is 判错（尤其 ErrRateLimited——429 走的是 RetryableError 链）。
下一步：维护者确认无外部依赖后删除；或让 WithRetry 在 429 耗尽时挂接 ErrRateLimited。

## T-003：Retry-After 不受 MaxDelay 封顶

状态：候选未验证
问题：服务端返回异常大的 Retry-After（如 3600s）时，重试等待只受 ctx 约束
（retry.go:96-100），MaxDelay 不生效。
证据（A）：代码现状。
影响：失控服务端可拖住调用方；无 ctx 的 DoRequestWithRetry 尤其暴露。
下一步：为 Retry-After 增加独立上限选项（默认不封顶，保持 D-004 语义）。

## T-004：HTTPError.URL 字段从不赋值

状态：候选未验证
问题：`HTTPError{StatusCode, Body, URL}` 的 URL 无任何赋值点（retry.go 构造时只填前两个）。
证据（A）：代码现状。
影响：错误信息缺上下文，字段属死重量。
下一步：在 WithRetry 构造处补 URL（需要签名传参）或删字段。

## T-005：两族 ctx 入口的注入顺序相反 ✅已关闭

状态：已完成（2026-09-14 d3250fb）
结果：DoRequestWithContext 改为前置注入（ctx 选项放最前），与 CtxGet/CtxPost/CtxPatch
统一。修复顺带解决：ctx 携带 opts 被静默清除（WithTimeout/Size/Middleware 失效）、
被替换 ctx 的 timeout cancel 泄漏、`append(opts, ...)` 覆写调用方切片底层数组。
回归：retry_extra_test.go 五组表驱动用例。详见 D-008 / H-008。

## T-006：README 仍把熔断写在 fetch 名下（仓库文件，本任务不改）

状态：等待决策（范围缩小：CLAUDE.md 已并入 AGENTS.md，其 fetch/circuitbreaker 条目准确；
仅 README 残留旧措辞）
问题：README 称 fetch 提供 circuit breaker；ca009f2 后能力在 circuitbreaker/ 包，
fetch 仅在 doc.go 指路。
证据（A）：文档现文 vs 代码。
下一步：由有权者更新 README 措辞（本次交付不修改仓库）。
