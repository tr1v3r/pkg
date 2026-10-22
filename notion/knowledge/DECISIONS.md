# notion — 架构与选型说明

更新：2026-10-12（刷新 @ d6b2b9c）· 证据等级：A=代码现状/提交可回查，C=推测（维护者不在场）

## D-001：Client 以接口字段暴露六个领域 API

选择：`Client.Database/Page/Block/Search/User/Comment` 是 `DatabaseAPI` 等接口类型字段，
而非具体 Manager（interfaces.go / manager.go）。
证据（A）：93f5dc0（#22）提交信息明写 "Define interfaces … for mock testing"；doc.go 有嵌入
接口的 mock 示例。
权衡：调用方可在测试中替换单个领域；代价是每个新方法要先改接口再改实现。
重新评估条件：接口膨胀到维护成本高于测试收益，或出现需要组合多个领域的跨实体操作。
状态：生效中。

## D-002：中心化 notionClient + httpDoer 抽象

选择：全部 HTTP/限流/错误处理收敛到 `notionClient.do`；HTTP 执行抽象为私有 `httpDoer`
接口（client.go）。#22 后不再依赖 pkg/fetch，默认 `http.DefaultClient`。
证据（A）：93f5dc0 提交信息 "centralized client"；`git log -S 'pkg/fetch' -- notion/` 显示
依赖在 #22 移除。CLAUDE.md「notion uses fetch」已过时（见 T-006）。
权衡：统一错误/限流语义；放弃 fetch 的重试与连接池配置（如需可注入自定义 httpDoer）。
重新评估条件：需要库内重试/熔断时，优先注入带行为的 httpDoer 而非回到 fetch 耦合。
状态：生效中。

## D-003：配置只走构造期 ClientOption，禁止 mutate 型方法

选择：仅 `WithRateLimiter` 一个 ClientOption；曾有的 `WithContext/WithLimiter` 方法被删除。
证据（A）：93f5dc0 提交信息明写原因——"managers held stale notionClient pointers after copy"。
权衡：`Client.Set(version, token)` 仍在（改共享 client），但这是凭据更新而非行为配置。
护栏：不要再给 Client/Manager 加"复制后自改"的方法，#22 已证明该模式产生悬垂指针。
重新评估条件：出现运行期必须动态调整的配置时，应显式加锁或改用原子替换整个 client。
状态：生效中。

## D-004：429 返回哨兵 ErrRateLimited，库内不自动重试

选择：限流器只是主动节流（默认 3 rps、burst 12，notion 官方限制）；收到 429 一律返回
`ErrRateLimited`，不带 Retry-After，不重试（client.go:84-86）。
证据（A）：哨兵由 5ce9d9c（2025-01-15）引入；代码现状无重试路径。
动机（C）：推测调用方希望自己拥有退避策略；也避免了库内重试与 limiter 的交互复杂度。
重新评估条件：若多数调用方都在手写同样的 429 退避，考虑在 client 层提供可选重试。
状态：生效中。

## D-005：分页三种形态并存

选择：`Query`（全量 slice）与 `QueryIter`（惰性 `iter.Seq2[Page, error]`）都建立在泛型
`paginateEach` 上；`paginateAll` 只是它的收集器（client.go:126-161）。
证据（A）：b1f8695（2026-04-20）提交信息明写 "lazy page-by-page iteration"。
权衡：QueryIter 支持早停与流式处理；但错误以迭代值出现，`for range` 必须检查第二个返回值。
重新评估条件：Go 版本降级（iter 需 1.23+）或需要向其他资源类型推广惰性迭代时泛化 querySetup。
状态：生效中。

## D-006：Property 用 json.RawMessage 承载多态值

选择：`Property` 的 title/rich_text/select 等字段是 `json.RawMessage`；写入时
`ForUpdate()` 按固定 switch 顺序只序列化第一个非空字段（property.go）。
证据（A）：代码现状；`RollupProp` 常量注释 "do not used when update"。
权衡：读侧保留原始 JSON 灵活性；写侧一个 Property 只能携带一种值，switch 顺序即优先级。
重新评估条件：若需要同字段多值或更严格类型，应改带判别联合的显式构造器而非放宽 ForUpdate。
状态：生效中。

## D-007：分页参数按 HTTP 方法分途传输（GET→query，POST→body）

选择：`paginateEach` 按 method 分派——GET 用 `withListQuery` 把 start_cursor/page_size
写进 URL query（保留既有参数），POST 维持 JSON body；GET 携带非 ListOptions body 时
报显式错误（client.go）。
证据（A）：e778688（2026-09-14）提交信息明写 Notion API 契约「GET list endpoints read
start_cursor and page_size exclusively from the URL query string and ignore request
bodies」，及修前后果（cursor 卡首页 → 无限重复项）。
权衡：传输层按端点契约分叉，do 不再对全部 method 统一 marshal；新增 GET 分页端点
时必须传 *ListOptions，否则运行期报错（编译期无保护）。
护栏：改分页逻辑时 GET/POST 两条路径都要有传输层断言（pagination_test.go 已建立
规范忠实 mock：cursor 只从 query 读、超页数即 400）。
重新评估条件：Notion 契约变化或新增非 ListOptions 形态的 GET 分页端点时。
状态：生效中（2026-09-14 起）。
