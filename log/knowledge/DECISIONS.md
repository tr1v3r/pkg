# log 组件架构与选型说明

基准日期 2026-10-12（刷新 @ d6b2b9c）· 证据分级：A=代码现状/可回查提交，C=推测（待维护者确认）。

## D-001：Sink 为自包含输出单元（级别过滤 + 编码器 + 写目标）

日期 2026-10-12 · 证据 A：e1deaff / PR #21「Sink-oriented architecture: Sink = Level Filter + Encoder + Writer」。

背景：旧设计（2023–2025）是 Handler 体系（ConsoleHandler/FileHandler/sync_console 等多套并行），行为分散。
选择：Logger 只做 Record 派发；级别过滤、编码、写出、异步、关闭全部收进 Sink；构造器（Console/File/RotateFile/SizeRotateFile）返回 *Sink。
权衡：sink 之间无法共享编码缓冲；换来行为边界清晰、每 sink 独立级别与异步模式。
重新评估条件：需要多 sink 共享编码结果或动态增删 sink 时。
状态：生效中。

## D-002：异步通道满时阻塞调用方（反压），不丢日志

日期 2026-10-12 · 证据 A：sink.go `s.ch <- data` 无 default 分支；策略自 e8e37cf「ensure no logs are dropped when channel is full」延续。

背景：曾用 default 分支丢弃，高负载下日志丢失。
选择：通道满则 Log 调用阻塞，直到 drain 消费。
权衡：日志风暴时业务延迟被拖慢（日志变成背压传导路径）；换取零丢失。
重新评估条件：出现「日志拖垮业务延迟」的实际场景时，可评估加丢弃计数/降级选项。
状态：生效中。

## D-003：Sync/Close 用 nil 哨兵屏障保证排空

日期 2026-10-12 · 证据 A：e1deaff 子提交「Rewrite Sync/Close to use a sentinel-based barrier instead of the previous quit-channel approach, ensuring all buffered data is flushed before Close returns」。

背景：本包历史上多次在关闭语义上翻车（死锁、双 close、Close 提前返回，见 HISTORY H-004）。
选择：Sync 发送 nil 哨兵并等 drain 确认；Close 先 close(ch) 再等 drain 自然退出。
权衡：Sync 会重启 drain goroutine（每次 Sync 一个新 goroutine）；并发调用 Sync/Close 未受保护（见 TODO T-003）。
重新评估条件：需要并发安全的 Sync，或 Sync 频繁到 goroutine 创建成为开销时。
状态：生效中。

## D-004：文件类 sink 默认异步（buf=1024），Console 默认同步

日期 2026-10-12 · 证据 A：file.go/rotate.go/size.go 均注入 WithAsync(1024)；console.go 不注入；e1deaff 子提交明写「default async writing for file sinks」。
动机（C）：推测为终端交互延迟敏感、文件写入可缓冲——待确认。
选择：WithSync() 可逐 sink 覆盖回同步。
权衡：~~Fatal 后立即 os.Exit 时异步缓冲可能未落盘~~——该边界已于 9115d2d 修复
（Fatal 先 l.Sync() 再退出，见 D-008 前的 H-007）；异步残余风险只剩「未走 Fatal 的
崩溃路径」（panic 未 recover）。
重新评估条件：若要求任意退出路径必落盘，需在进程退出钩子里统一 Sync。
状态：生效中（Fatal 边界 2026-08-24 起已加固）。

## D-005：slog 兼容内建于本包，独立 slog/ 包装包已删除

日期 2026-10-12 · 证据 A：e1deaff / PR #21「Delete slog/ package — native structured logging makes it unnecessary」；slog.go 提供双向适配。

已知边界（均为 A 级代码现状）：
- AsSlogHandler.Enabled 恒 true：slog 侧不过滤，过滤发生在 sink；slog 的 Debug 会按 sink 级别被丢。
- 级别映射有损：slog 无 Fatal，AsSlogHandler 方向最高到 ErrorLevel；TraceLevel 映射为 slog.LevelDebug-4。
- WithGroup 语义偏差：group 前缀作用于全部累积 attrs（含 WithGroup 之前加的），与 slog 规范（先加的 attrs 不进 group）不一致。
- SlogHandler 构造的 sink 无 writer：不可当 io.Writer 用（nil panic）。
重新评估条件：需要严格 slog 语义（分组、级别过滤委托）时重审这四处边界。
状态：生效中。

## D-006：级别按初始化期配置对待（SetLevel 无锁）

日期 2026-10-12 · 证据 A：eb1318f「Remove unnecessary mutex in SetLevel across handlers (level is init-time config)」；现行 sink.go SetLevel 仍为裸字段写。
矛盾：doc.go 宣称「Change level at runtime」——文档与设计假设冲突（TODO T-006）。
重新评估条件：确需运行时动态调级时，必须先给 level 加原子操作。
状态：生效中（按初始化期假设）。

## D-007：Sink 实现 io.Writer 以接入 gin.DefaultWriter

日期 2026-10-12 · 证据 A：e1deaff 子提交「implement io.Writer on Sink for gin.DefaultWriter replacement」（原内部 Write(Record) 为此改名 Log）。
边界：异步 sink 走 Write 时会拷贝字节入队（原始字节不经过编码器，直接落盘）。
重新评估条件：gin 集成退场或需要按 Record 结构化外来日志时。
状态：生效中。

## D-008：With() 为子 Logger 构造独占的字段存储

选择：`Logger.With` 把父级 fields 与新 fields 拷入全新、精确尺寸的切片；空结果保持
nil（log.go）。Fatal 族统一走 `exit()`：先 `Sync()` 排空全部异步 sink 再 `os.Exit(1)`。
证据（A）：d6b2b9c（2026-09-14，BUG-AUDIT-REPORT H4）——修前 `append(l.fields, ...)` 在
父切片有富余容量时原地写入并共享数组，同一父级的后建兄弟覆写先建兄弟的字段（构造期
发生，-race 不可见，属「静默日志内容伪造」）；两跳以上 With 链因容量倍增几乎必中。
Fatal 冲洗为 9115d2d（2026-08-24）「flush async sinks before Fatal exits」。
权衡：每次 With 多一次分配；换来子 Logger 之间的隔离与并发安全（with_test.go 的
32 goroutine 兄弟并发用例 + 多跳/深链用例锁定）。重新分配是正确性必需，不是优化点。
护栏：不要「优化」回 append 复用；给 With 加字段去重/覆盖语义时必须保持独占存储。
重新评估条件：热路径上 With 分配成为实测瓶颈时，考虑写时复制或池化——仍不得共享
可写底层数组。
状态：生效中（2026-09-14 起）。
