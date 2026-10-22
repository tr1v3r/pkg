# fetch — 修改历史

更新：2026-10-12（刷新 @ d6b2b9c）· 证据等级：A=提交信息/代码可回查，C=推测。
完整流水以 `git log --oneline -- fetch/` 为准。

## H-001：函数式极简期（2023-03 ~ 2023-05）

变化：6fa9996 初始实现（Get/Post 直返 body）；7dd7b23 body 接受 io.Reader；
6c71627 删除冗余 API（大扫除，C：推测收紧面）；9292b36 引入 ctx 变体；302784a 补 PATCH。
当前影响：如今的 Get/Post/Ctx* 入口形态即定型于此。
关联（A）：各提交标题。

## H-002：业务响应结构期（2023-12）

变化：8e99f4e 增加 base.go（RespCode/Pagination/JSONResult）。
当前影响：类型仍在但包内零引用（见 D-007）；7340f21 顺手修了 NewRequest 错误检查。
关联（A）：8e99f4e。

## H-003：韧性大重写（2025-10-26，161ea9d）

变化：加入熔断、重试、中间件、错误类型与并发测试——"comprehensive HTTP client"。
当前影响：现行 retry/middleware/errors 骨架来源；errors.go 三个未使用哨兵
（ErrRateLimited 等）即此期遗留（见 TODO T-002）。
关联（A）：161ea9d 提交信息。

## H-004：并发与安全修复（2026-04-09，aeaa5b0）

变化：熔断 RWMutex 锁降级竞态改单一 Mutex；mixed mutex/atomic 混用清除；
WithRequestTimeout goroutine 泄漏修复（缓冲 channel + drain）；响应体 100MB 上限
（LimitReader）+ WithMaxResponseBodySize。
当前影响：DoRequestWithOptions 的 maxBodySize 逻辑与中间件 drain 模式均源于此。
关联（A）：aeaa5b0 提交信息逐条列出。
备注：熔断部分修复随 ca009f2 外迁至 circuitbreaker/，此处仅存历史。

## H-005：上下文感知重试（2026-04-09，9172b71）

变化：WithRetry 加 ctx（可取消退避）；body 缓冲保重试；解析 Retry-After（秒/HTTP-date）；
中间件集成进 DoRequestWithOptions；修 WithTimeout cancel 泄漏；crypto/rand → math/rand。
原因（A）：提交信息明写各项动机（"POST/PUT retries send the full body" 等）；
math/rand 一条的理由是"非安全随机不需要密码学开销"。
当前影响：D-004/D-005 的全部语义。
关联（A）：9172b71。

## H-006：重写当天的补漏（2026-04-09，d981439 + 748bae3）

变化：修复 6 个失败单测（MaxAttempts≤0 语义、IsServerError 排除 ≥600、集成测试期望值、
熔断测试改看状态码、context key 类型不匹配、TestDefaultClient 去 flaky）；随后 748bae3
把 time.NewTimer 换 time.After 简化。
当前影响：`MaxAttempts≤0 = 单次执行` 是被测试固化的显式语义，不是 bug。
关联（A）：d981439 提交信息逐条列出。

## H-007：熔断外迁（2026-04-09，ca009f2）

变化：CircuitBreaker/Config/State 及测试移入新包 circuitbreaker/；fetch/errors.go 删
CircuitBreakerError/ErrCircuitOpen；doc.go 改为指路。
当前影响：fetch 的"韧性"= 重试 + 中间件 + 上限；熔断需使用者自行组合（D-003）。
README/CLAUDE.md 仍把熔断写在 fetch 名下，属文档滞后（A：两文件现文 vs 代码）。
关联（A）：ca009f2。

## H-008：DoRequestWithContext 的 ctx 注入顺序修复（2026-09-14，d3250fb）

变化：`append(opts, WithContext(ctx))` 改为 `append([]RequestOption{WithContext(ctx)}, opts...)`，
与 CtxGet/CtxPost/CtxPatch 对齐。
原因（A）：提交信息引 BUG-AUDIT-REPORT H1——后置 WithContext 整体替换请求 context，
静默丢弃 opts 中的 ctx 携带选项（WithTimeout/WithMaxResponseBodySize/WithMiddleware），
且被替换 ctx 的 timeout cancel 永不调用（计时器泄漏）；旧形式还会覆写有富余容量的
调用方 opts 底层数组。
当前影响：D-008；T-005 关闭；retry_extra_test.go 固化超时/体积/中间件/传播/不别名
五组回归。
关联（A）：d3250fb。
