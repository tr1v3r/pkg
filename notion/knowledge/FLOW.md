# notion — 流程图

更新：2026-10-12（刷新 @ d6b2b9c）· 证据：代码现状 + e778688 提交 · 入口：`NewClient`（manager.go）

```text
【构造】两条路径，限流器归属不同
  NewClient(version, token, opts...)      六个 Manager 共享同一个 notionClient + 默认限流器(3 rps / burst 12)
  NewXxxManager(version, token)           每个 Manager 各建独立 notionClient（限流互不共享）
  Client.Set(version, token)              改写共享 client 的凭据；独立构造的 Manager 不受影响

【请求主路径】Manager 方法 → notionClient.do(ctx, method, path, body, result)   client.go
  limiter.Wait(ctx) ──err──→ return "rate limiter: %w"（ctx 取消在此暴露）
  body≠nil → json.Marshal → http.NewRequestWithContext
  注入固定三头：Notion-Version / Authorization Bearer / Content-Type
  http.Do(req) ──err──→ return "request %s %s: %w"
  io.ReadAll(resp.Body)
  状态分派：
    429             → return ErrRateLimited（哨兵，不自动重试）
    ≥400 且体可解析 → *APIError{Status, Code, Message}（识别条件 Code≠""）
    ≥400 解析失败   → fmt.Errorf("notion api error: [%d] %s", …)（保留原文）
    2xx             → result≠nil 时 json.Unmarshal → nil

【分页查询】databases.go querySetup + client.go paginateEach/paginateAll
  querySetup：cond==nil → 空 Condition；PageSize≤0 → 补 100；filter_properties 走 URL query
  每页请求体 = cond 浅拷贝 + StartCursor=服务端返回的 next_cursor
  paginateEach（惰性）：
    循环 { do → 逐项 yield(item,nil) → HasMore? 否:终止 }
    页内某项 yield 返回 false（消费方 break）→ 立即 return，不再请求下一页
    do 失败 → yield(零值, err) → return（错误即终止，同序列只出现一次）
  paginateAll（全量）：内部消费 paginateEach；任一项出错 → 整体 return nil, err
  QueryIter = paginateEach 直接暴露（iter.Seq2[Page, error]，b1f8695）

【分页传输】client.go paginateEach —— 按 HTTP 方法分途（e778688，2026-09-14）
  GET  → withListQuery 把 start_cursor/page_size 合入 URL query（保留已有参数，
         如 comments 的 block_id），不发送 body
  POST → 维持 JSON body 携带分页（Database.Query 等）
  GET 且 body 非 *ListOptions → yield 显式错误（不再静默退化）
  User.List / Block.Children / Comment.List 三个 GET 端点走此路径
  历史：修前 cursor 放 GET body，服务端只认 query → cursor 永不前进 → 无限
  拉取重复首页（BUG-AUDIT-REPORT H2；回归见 pagination_test.go 规范忠实 mock）

【错误包装约定】每个 Manager 方法再包一层 fmt.Errorf("动作 对象: %w")
  调用方：errors.Is(err, ErrRateLimited) / errors.As(err, &*APIError)
```
