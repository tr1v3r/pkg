# fetch — 流程图

更新：2026-10-12（刷新 @ d6b2b9c）· 证据：代码现状 + d3250fb 提交 · 熔断不在本包：见兄弟包 circuitbreaker/

```text
【入口族】fetch.go
  Get/Post/Patch + Ctx 变体 ────→ DoRequestWithOptions（Ctx* 前置注入 WithContext）
  DoRequestWithContext ─────────→ DoRequestWithOptions（d3250fb 起同为前置注入，与 Ctx* 对齐；
                                    此前后置注入整体替换 ctx，静默丢 ctx 携带的 opts——
                                    WithTimeout 不生效/超时 cancel 泄漏/Size+Middleware 被无视）
  DoRequestWithRetry(Context) ──→ 先 io.ReadAll 缓冲 body → WithRetry 循环 → 每次尝试
                                   重建 bodyReader → 走 DoRequestWithOptions（每尝试新请求）

【单次请求管线】DoRequestWithOptions(method, url, opts, body)
  http.NewRequest ──err──→ "build new request fail: %w"
  依序应用 RequestOption（header/WithContext/WithTimeout/WithQueryParams/…）
    ⚠ WithTimeout/WithMaxResponseBodySize/WithMiddleware 把配置存进 ctx.Value
  defer timeoutCancelKey 里的 cancel（防 context.WithTimeout 计时器泄漏）
  读取 maxBodySizeKey（默认 100MB；<0 = 不限）
  doRequest 闭包：
    DefaultClient().Do(req) ──err──→ statusCode=-1 返回 err（网络错误）
    LimitReader(resp.Body, maxBodySize) → io.ReadAll → 返回 (status, content, header)
  ctx 里有 middlewareKey？ → ChainMiddleware(...)(doRequest) ；否则直接 doRequest()

【中间件链】middleware.go  Middleware = func(next func()(int,[]byte,http.Header,error)) (…)
  ChainMiddleware 逆序包装：middlewares[0] 最外层、最后传入的紧贴 HTTP 执行
  WithRequestTimeout：goroutine 跑 next()，select 等 done / time.After(timeout)
    └ 超时 → 返回 ErrRequestTimeout；底层请求继续跑直至完成（drain goroutine 仅助 GC）
  WithLogging/WithMetrics：包不住请求信息（method/url 传空串，LogRequest 被注释）

【重试循环】WithRetry(ctx, config, fn)  retry.go
  MaxAttempts≤0 → fn 恰好执行一次；状态命中 RetryOnStatus → 返回 RetryableError{Attempts:1}
  每次尝试前 select ctx.Done() → 返回 (上次status/content, ctx.Err())
  fn()：
    err==nil 且 status ∉ RetryOnStatus → 成功返回（4xx/2xx 均如此）
    err==nil 且 status ∈ RetryOnStatus(408/429/500/502/503/504) → lastErr=HTTPError，继续
    err≠nil → lastErr=err，继续
  末次尝试 → break
  计算延迟：Retry-After 头（整数秒或 HTTP-date）优先且【不受 MaxDelay 封顶】
            否则退避 = Base*2^attempt*(1±Jitter)，attempt=0 不抖动，cap MaxDelay
  可取消 sleep：select time.After(delay) / ctx.Done() → 返回 ctx.Err()
  耗尽 → RetryableError{Err: lastErr, Attempts: MaxAttempts}（Unwrap 到底层）

【全局状态】fetch.go
  包级 httpClient（RWMutex 保护）：60s 超时、TLS≥1.2、每 host ≤10 连接、ProxyFromEnvironment
  SetDefaultClient 运行期整体替换（进程级共享）
```
