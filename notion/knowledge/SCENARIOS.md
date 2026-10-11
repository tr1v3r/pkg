# notion — 自然语言测试案例库

更新：2026-10-12（刷新 @ d6b2b9c）· 基线：d6b2b9c 代码现状。mock 手法参照 notion/mock_test.go
（httptest.Server + urlRewriteClient 重写 base URL，不依赖外网）。

## S-001：Query 跨页收集

前置：mock server 对 `POST /v1/databases/db-1/query` 第一次返回
`{results:[p1], has_more:true, next_cursor:"cursor-abc"}`，第二次返回
`{results:[p2], has_more:false}`。
动作：`Query(ctx, "db-1", nil)`。
预期：1. 返回 `[p1, p2]`；2. 第二次请求体的 `start_cursor == "cursor-abc"`；3. 共 2 次请求。
验证：断言请求体 JSON（参照 api_test.go TestDatabaseManager_Query_MultiplePages）。

## S-002：QueryIter 早停不再取下一页

前置：同 S-001 的两页 mock（第二页 has_more 仍可设 true）。
动作：`for page, err := range QueryIter(...)` 收到第 1 项后 break。
预期：1. 只收到 1 项；2. 总请求数为 1（首页），不再发起第二页请求。
验证：mock 侧计数 callCount == 1。

## S-003：迭代中途出错即终止

前置：第一页正常返回 1 项 + has_more:true；第二页返回 HTTP 500 + 合法 Notion 错误体。
动作：完整 range QueryIter。
预期：1. 先 yield(p1, nil)；2. 随后 yield(零值, err) 恰好一次且 err 可 `errors.As` 为
`*APIError`；3. 之后序列结束，无第三次请求。
验证：迭代计数与 callCount；错误类型断言。

## S-004：429 与限流器取消

前置 A：mock 返回 429。前置 B：`WithRateLimiter(rate.NewLimiter(0, 0))`（永不放行）+
即将到期的 ctx。
动作 A：任意 API 调用。动作 B：带该 ctx 调用。
预期 A：`errors.Is(err, notion.ErrRateLimited)` 为真，且错误链不含 APIError。
预期 B：err 字符串含 `rate limiter:` 前缀（limiter.Wait 先于 HTTP 失败）。
验证：错误链断言；确认无 HTTP 请求发出（callCount==0）。

## S-005：非 JSON 错误体兜底

前置：mock 返回 500 + 纯文本 body "boom"。
动作：任意 API 调用。
预期：`errors.As(*APIError)` 为假；err 字符串含 `[500]` 与原文 `boom`。
验证：字符串包含断言（参照 api_test.go TestError_InternalServerError_PlainText）。

## S-006：GET 分页 cursor 传输（已固化单测，集成复核可选）

前置：pagination_test.go 的规范忠实 mock——cursor 只从 query string 读取，请求超过
页数即答 400（非前进 cursor 快速失败）。
动作：三个 GET 端点（users / block children / comments）走三页 cursor 链。
预期：1. cursor 序列正确推进；2. 请求次数恰为页数；3. 聚合完整无重复；4. 每次请求
body 为空；5. 端点自有参数（block_id）保留在 query。
验证：pagination_test.go（e778688 已锁定；POST 侧另有「分页不泄漏进 query」断言）。
执行注记：单元级已 pinned；带真实 token 的集成跑（`-tags=integration`，凭据经
NOTION_VERSION/TOKEN/DATABASE_ID 注入，缺失即 Skip）只剩复核价值。

## S-007：Client.Set 对共享 client 生效

前置：NewClient 构造（六 Manager 共享）；mock 校验 Authorization 头。
动作：`Set(newVersion, newToken)` 后调用 `Page.Retrieve`。
预期：请求头携带新 token/version；证明 Set 走共享 notionClient（对照 T-004 的独立构造
路径——该路径不受 Set 影响）。
验证：mock 侧读取请求头断言。
