# notion — 待办事项

更新：2026-10-12（刷新 @ d6b2b9c）· 待办是背景，不是本次任务的授权清单；执行前需维护者确认。

## T-001：GET 端点分页 cursor 走 JSON body，疑似与 Notion API 规范不符 ✅已关闭

状态：已完成（2026-09-14 e778688）
结果：推断属实——Notion GET 端点只从 query string 读分页参数、忽略 body，修前 cursor
永不前进 → 无限拉取重复首页（BUG-AUDIT-REPORT H2 证实）。修复：GET 走 withListQuery，
POST 维持 body；pagination_test.go 用规范忠实 mock 锁定三端点三页 cursor 链。
残留：带真实 token 的集成复核仍可选（见 S-006 降级注记），但正确性已有单测 pinned。

## T-002：RetrieveProperty 注释与实现不符（未自动分页）

状态：候选未验证
问题：pages.go:35 注释称 "handling pagination automatically"，实现是单次 GET，无分页循环。
证据（A）：代码现状。
影响：rollup 等可分页 property item 只取第一页，调用方若信注释会静默丢数据。
下一步：维护者裁决——改注释或补分页（注意 GET 分页与 T-001 同源）。

## T-003：Search 仅返回单页

状态：候选未验证
问题：`SearchManager.Search` 返回单个 `ListResponse`，无 paginateAll 包裹；doc.go 总述
"automatic pagination" 与此不一致（代码现状，A）。
影响：搜索结果超过一页时静默截断。
下一步：确认是否有意为之（C：推测日常搜索场景一页足够）；否则套用 paginateAll。

## T-004：独立 NewXxxManager 构造器与 NewClient 并存

状态：等待决策
问题：六个 `NewXxxManager(version, token)` 各建独立限流器，同一进程混用两套构造会让
3 rps 限制形同虚设（每实例各 3 rps）。
证据（A）：构造器存在（代码现状）；保留动机（C）：疑似 #22 前旧 API 的兼容残留。
下一步：维护者确认是否废弃；若保留，doc.go 应说明限流不共享。

## T-005：429 响应丢弃 Retry-After 头

状态：候选未验证
问题：`do` 收到 429 立即返回哨兵，响应头未读取（client.go:84-86）。
影响：调用方无法实现精确退避，只能盲等。
下一步（C，方向建议）：可考虑 `ErrRateLimited` 变体携带时长；与 D-004 一起重新评估。

## T-006：CLAUDE.md 描述过时（仓库文件） ✅已关闭

状态：已完成（2026-08-24 9115d2d）
结果：CLAUDE.md 改为指向 AGENTS.md 的 symlink（AGENTS.md 为新权威）。AGENTS.md 的
notion 条目不再称 "notion uses fetch"（跨包依赖一节只列 notion→log），与 #22 后
直连 http.DefaultClient 的现实一致。旧知识库 D-002 里的「见 T-006」引用同步作废。
