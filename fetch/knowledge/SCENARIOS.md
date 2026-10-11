# fetch — 自然语言测试案例库

更新：2026-10-12（刷新 @ d6b2b9c）· 基线：d6b2b9c 代码现状。均可 httptest.Server 本地执行。

## S-001：429 + Retry-After 秒数精确等待

前置：mock 首次返回 429 + `Retry-After: 1`，第二次返回 200。
动作：`DoRequestWithRetryContext(ctx, GET, url, nil, nil)`（默认配置）。
预期：1. 最终成功；2. 两次请求间隔 ≥1s（Retry-After 生效，而非 100ms 基础退避）；
3. 请求计数 == 2。
验证：记录首次响应完成与第二次请求到达的时间差。

## S-002：重试耗尽返回 RetryableError

前置：mock 恒返回 503（body "unavailable"）。
动作：`WithMaxAttempts(3)` 执行。
预期：1. err 为 `*RetryableError` 且 `Attempts == 3`；2. `errors.Unwrap` 得到
`*HTTPError{StatusCode:503}`；3. fn 恰好执行 3 次。
验证：类型断言 + 调用计数（参照 retry_test.go TestRetryMaxAttempts）。

## S-003：退避 sleep 期间 ctx 取消

前置：mock 恒返回 500；ctx 设 150ms 超时；BaseDelay 调大（如 5s）。
动作：执行带重试请求。
预期：1. 返回 ctx.Err()（context.DeadlineExceeded）；2. 不会等满 5s 退避；
3. 返回值携带最后一次的 status/content。
验证：耗时断言 < 5s；错误链断言。

## S-004：POST 重试发送完整 body

前置：mock 首次返回 502、二次 200，均回读并记录请求 body。
动作：`DoRequestWithRetryContext(ctx, POST, url, nil, strings.NewReader(payload))`。
预期：两次请求的 body 都等于 payload 全文（缓冲生效，第二次非空）。
验证：服务端存档两份 body 做字节比较。

## S-005：响应体超限被静默截断

前置：mock 返回 150MB body；`WithMaxResponseBodySize(100 << 20)` 或用默认值。
动作：Get(url)。
预期：1. err == nil；2. len(content) == 100MB（LimitReader 截断，不报错）。
验证：长度断言。注意：截断无信号是当前语义，若要报错属行为变更。

## S-006：WithRequestTimeout 超时后底层请求继续

前置：mock 处理故意 sleep 300ms 再返回；`WithMiddleware(WithRequestTimeout(50ms))`。
动作：DoRequestWithOptions + 该 option。
预期：1. 立即（约 50ms）返回 ErrRequestTimeout；2. mock 侧最终仍收到完整请求并完成响应
（后台不取消）；3. 进程不泄漏 goroutine（drain 完成）。
验证：客户端耗时断言 + 服务端计数（延迟检查）。

## S-007：中间件包装顺序

前置：两个记录型中间件 A、B（各自在 next() 前后 append 标记）；
`WithMiddleware(A, B)`。
动作：执行一次请求。
预期：事件序列为 A-enter → B-enter → HTTP → B-exit → A-exit（A 为最外层，
ChainMiddleware 逆序包装语义）。
验证：共享 slice 的顺序断言（参照 middleware_test.go TestMiddlewareOrder）。

## S-008：MaxAttempts=0 单次执行 + 可重试状态报 RetryableError

前置：mock 返回 429；`WithMaxAttempts(0)`。
动作：执行。
预期：1. 恰好 1 次请求；2. 返回 `*RetryableError{Attempts:1}`，Unwrap 到
`*HTTPError{StatusCode:429}`——不是 err==nil。
验证：调用计数 + 错误链断言（d981439 固化的语义）。

## S-009：DoRequestWithContext 的 ctx 携带选项不被清除（d3250fb 回归，已固化）

前置：mock 端点 sleep 2s；`DoRequestWithContext(ctx(150ms 预算), GET, url,
[WithTimeout(150ms), WithMaxResponseBodySize, WithMiddleware(计时)]，nil)`。
动作：执行。
预期：1. 约 150ms 返回超时错误（修前跑满 2s 且 err=nil）；2. 中间件被实际调用；
3. 体积上限生效；4. 调用方 opts 切片底层数组不被覆写（别名检查）。
验证：耗时断言 + 中间件计数；对照 retry_extra_test.go 五组表驱动用例。
