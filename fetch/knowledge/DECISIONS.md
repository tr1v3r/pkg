# fetch — 架构与选型说明

更新：2026-10-12（刷新 @ d6b2b9c）· 证据等级：A=代码现状/提交可回查，C=推测（维护者不在场）

## D-001：中间件签名不含请求信息（无 ctx、无 req）

选择：`Middleware = func(next func() (int, []byte, http.Header, error))`（middleware.go）。
代价（A，代码自证）：`WithRequestTimeout` 超时后底层请求继续执行（doc 注释明示）；
`WithLogging` 拿不到 method/url——现传空串、LogRequest 调用被注释掉。
动机（C）：推测为了包装 DoRequestWithOptions 的四元组返回值，而非改造 http.Request。
权衡：签名简单、链式组合容易；牺牲了可观测性与真取消。
重新评估条件：需要中间件读取请求上下文或真正中止传输时，必须做 breaking 的签名变更
（届时优先 `func(ctx, *http.Request, RoundTrip)` 形态）。
状态：生效中。

## D-002：配置经 context.Value 走私到执行层

选择：`WithTimeout`（cancel 函数）、`WithMaxResponseBodySize`、`WithMiddleware` 都把值
存进 req.Context()，由 DoRequestWithOptions 在执行期取回（option.go / fetch.go）。
证据（A）：代码现状；9172b71 修复 WithTimeout cancel 泄漏时建立的机制。
护栏：RequestOption 只能改 *http.Request，想影响"执行"而非"请求"就只能走 ctx——新增
此类选项时沿用私有 contextKey，不要复用已有 key。
重新评估条件：若 option 需求继续增长，考虑显式的执行配置参数（breaking）。
状态：生效中。

## D-003：熔断外迁为兄弟包，fetch 不内置

选择：CircuitBreaker 全部类型移至 `circuitbreaker/`，fetch/errors.go 删除
CircuitBreakerError/ErrCircuitOpen。
证据（A）：ca009f2 提交信息明写动机 "for reuse beyond HTTP (DB, gRPC, Redis, etc)"，
并要求 fetch 测试直接 import circuitbreaker。
权衡：组合自由（cb.Execute 包住任意调用）；使用者要自己拼装重试+熔断的嵌套顺序。
重新评估条件：几乎不需要——如需默认组合，新建高层封装包即可，不必回迁。
状态：生效中。

## D-004：重试判定 = 传输错误 ∪ RetryOnStatus；Retry-After 优先且无上限

选择：`WithRetry` 在 err≠nil 或状态码 ∈ {408,429,500,502,503,504} 时重试；服务端
Retry-After（整数秒 / HTTP-date）优先于本地退避，且不受 MaxDelay 封顶（retry.go:96-100）。
证据（A）：9172b71 提交信息明写 "respect server-specified backoff delays"；代码现状。
权衡：尊重服务端；代价是异常大的 Retry-After 会让调用长时间挂起（仅 ctx 可打断）——
见 TODO T-003。
重新评估条件：出现真实故障案例时，为 Retry-After 增加独立上限（不改变默认语义）。
状态：生效中。

## D-005：重试安全 = 请求体全量缓冲

选择：`DoRequestWithRetryContext` 先把 body 读成 []byte，每次尝试用 bytes.NewReader
重建（retry.go:204-220）。
证据（A）：9172b71 提交信息明写动机——不缓冲则首次消费后重试发空 body。
权衡：大 body 重试场景内存翻倍；流式 body 不支持重试。
重新评估条件：出现大文件上传重试需求时，需要基于 Seeker 的重置协议。
状态：生效中。

## D-006：包级共享默认 client

选择：全局 `httpClient`（RWMutex 保护）+ `SetDefaultClient` 运行期替换；默认 60s 超时、
TLS≥1.2、每 host 10 连接（fetch.go:15-43）。
证据（A）：代码现状；d981439 提到并发测试改全局 client 导致 TestDefaultCoord flaky。
护栏：这是进程级状态——库代码不应在 import 副作用或并发路径里调用 SetDefaultClient。
重新评估条件：需要多租户/多代理隔离时，提供实例化 client 形态的新 API。
状态：生效中。

## D-007：base.go 的 JSONResult/RespCode/Pagination 是历史遗留的 API 响应约定

选择：与 HTTP 客户端无关的"业务响应包装"类型留在包内（base.go，中文注释）。
证据（A）：8e99f4e（2023-12）"add fetch base data struct" 引入后无演进；当前包内无引用。
动机（C）：早期作为内部 API 客户端的统一响应约定。
重新评估条件：确认无外部使用者后可移除（见 TODO T-002 同类清理）。
状态：生效中（未被包内使用）。

## D-008：ctx 注入统一为「最前」——DoRequestWithContext 与 Ctx* 同序

选择：三个 Ctx* 入口与 `DoRequestWithContext` 都以
`append([]RequestOption{WithContext(ctx)}, opts...)` 把 ctx 选项放最前，用户 opts
可随后覆盖（fetch.go）。
证据（A）：d3250fb（2026-09-14，BUG-AUDIT-REPORT H1）——修前 DoRequestWithContext
把 WithContext 追加在 opts 之后，WithContext 全量替换 req 的 context，导致 opts 里
所有 ctx 携带配置被静默清除（WithTimeout 对 2s 端点跑满 2s 且 err=nil、
WithMaxResponseBodySize/WithMiddleware 失效、被替换 ctx 里的 timeout cancel 永不调用
= 计时器泄漏）；`append(opts, ...)` 还会在 opts 有富余容量时覆写调用方底层数组。
权衡：用户显式再传 WithContext 可覆盖注入 ctx（既有语义，四入口一致）；修复同时
消除了入参切片别名副作用。
护栏：新增 ctx 系入口时沿用「ctx 选项最前」的统一顺序；改顺序前先看 d3250fb 的
回归测试（超时/体积上限/中间件/ctx 传播/入参不别名五组）。
重新评估条件：几乎不需要；若未来让 opts 优先于注入 ctx，属 breaking 行为变更。
状态：生效中（2026-09-14 起）。
