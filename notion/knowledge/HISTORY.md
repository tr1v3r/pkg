# notion — 修改历史

更新：2026-10-12（刷新 @ d6b2b9c）· 证据等级：A=提交信息/代码可回查，C=推测。完整流水以
`git log --oneline -- notion/` 为准，此处只记方向性变化。

## H-001：原型期一周内三连修（2023-04-27 ~ 04-28）

变化：853fa20 初始化 → d52aac1 修 API host 为 api.notion.com → 271503e 修 page manager
API typo。
当前影响：无（均被 #22 重写覆盖）；仅说明早期代码未经充分验证。
关联（A）：三个提交均存在。

## H-002：SDK 化与限流引入（2023-05-06）

变化：462bb76（#5）整理为 manager 形态；同日 e719aec 加请求限流、49770a0 加 GetID。
原因（A）：e719aec 标题 "limit request"；对应 notion 官方 3 rps 限制（object.go 注释链接）。
当前影响：`rateLimit = 3` 与 `defaultLimiter()`（burst 4×3）沿用至今。
关联（A）：e719aec。

## H-003：429 语义固化为哨兵错误（2025-01-15）

变化：5ce9d9c 检测到 429 返回 `ErrRateLimited`（errors.go），而非当作普通错误体。
当前影响：调用方依赖 `errors.Is` 判限流；与 D-004 一起构成现行错误契约。
关联（A）：5ce9d9c 提交信息 "return ErrRateLimited when detect return code 429"。

## H-004：filter/property 增量打磨期（2024-10 ~ 2025-01）

变化：5ce881f、85cbe93、b251a45、10d9984、daf55e3 陆续补 filter 条件、select option、
number filter 与 page_size。
当前影响：FilterCondition 的 CompoundConditions 自定义 MarshalJSON（filter.go）是这批
增量的产物，形态较特殊（嵌入 single + 复合 map 二选一序列化）。
关联（A）：上述各提交。

## H-005：整仓重设计 #22（2026-04-20，93f5dc0）

变化：原型 → "production-quality client"：中心化 notionClient、typed structs 取代 god
`Object`、六接口 + BREAKING（context 首参、Create 返回对象、Manager 字段改接口型）。
原因（A）：提交信息逐条列出修复的缺陷——channel 泄漏、limiter 错误被吞、URL 注入、
condition 被原地改写；并记录了被否决方案：WithContext/WithLimiter 方法（复制后悬垂指针）。
当前影响：现行全部架构（见 D-001~D-003）；`Object` 大结构体仍留在 object.go 供旧字段引用。
关联（A）：93f5dc0。

## H-006：QueryIter 惰性迭代（2026-04-20，b1f8695）

变化：新增 `paginateEach[T]` 泛型与 `DatabaseAPI.QueryIter`；`paginateAll` 改为它的收集器，
`querySetup` 抽出共享。同日在 #22 之后数小时落地。
原因（A）：提交信息 "lazy page-by-page iteration"；大结果集内存与早停需求（C：后半句为推测）。
当前影响：Query 与 QueryIter 共用同一分页路径，修分页 bug 时两处同时生效。
关联（A）：b1f8695。

## H-007：GET 分页改走 query string——T-001 判死并修复（2026-09-14，e778688）

变化：`paginateEach` 按 method 分派；GET 分页参数经 `withListQuery` 进 URL query，
POST 维持 body；GET 非 ListOptions body 报显式错误。旧 mock（按 callCount 脚本化）
改为规范忠实（cursor 只从 query 读、拒绝意外 cursor）。
原因（A）：提交信息引 BUG-AUDIT-REPORT H2——GET 端点忽略 body，cursor 永不前进，
迭代无限返回重复项且无界累积。旧知识库 T-001 的「存疑→潜在无限循环」推断被证实。
当前影响：D-007 的分途契约；T-001 关闭；S-006 降级为可选的集成复核。
关联（A）：e778688。

## H-008：godoc 全覆盖与既有 mock 规范化（2026-08-24，5d88794 / e778688 附带）

变化：5d88794 给全部导出符号补注释（object/property/filter 等类型语义、常量组说明）；
随 e778688 的测试修正，UserManager.List、BlockManager.Children 的 mock 也改为按
query cursor 供页并拒绝意外 cursor。
当前影响：文档注释成为字段语义的一级证据来源；mock 规范化后「mock 不校验传输」
类盲区在分页路径上已消除。
关联（A）：5d88794、e778688。
