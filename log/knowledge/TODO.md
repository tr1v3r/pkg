# log 组件待办事项

基准日期 2026-10-12（刷新 @ d6b2b9c）· 待办是背景信息，不是任何后续任务的授权清单。证据 A=代码现状，C=推测。

## T-001：Fatal 在异步文件 sink 上可能在落盘前 os.Exit ✅已关闭

状态：已完成（2026-08-24 9115d2d）
结果：Fatal 族统一走 exit() = Sync() + os.Exit(1)，异步 sink 先排空再退出
（提交信息 "flush async sinks before Fatal exits"）。README「guaranteed message
delivery」的矛盾面收窄到非 Fatal 硬退出路径（如未 recover 的 panic）。

## T-002：Close 之后再 Log/Sync 会 panic（send on closed channel）

状态：已确认待处理（代码现状 A：sink.go Close 关闭 ch，Log/Sync 无 closed 标记检查）。
影响：优雅关闭期间的迟到日志直接 panic 而非拒绝；与 e8e37cf 的「不丢日志」立场冲突。
下一步：加 closed 标志或 recover，或至少在 doc.go 标注「Close 后不可再用」。
完成条件：补一条 SCENARIOS 回归案例并固化测试。

## T-003：并发 Sync / Sync+Close 未受保护

状态：候选未验证（代码现状 A：Sync 重启 drain、Close 关 ch，均无互斥）。
影响：两个并发 Sync 可能有一个等不到 done 信号；Sync 与 Close 并发可能 panic。
下一步：构造并发测试（-race）复现。
完成条件：确认行为后决定加锁或文档声明单线程收尾约束。

## T-004：drain 忽略写错误；轮转 openFile 失败被吞

状态：已确认待处理（代码现状 A：sink.go drain 丢弃 Write 错误；rotate.go/size.go rotate 中 `_ = w.openFile(now)`）。
影响：轮转失败后 current 已关闭，后续写入持续失败，日志静默全丢且无任何信号（H-005 的回归面）。
下一步：错误至少暴露到 sink 层（计数器或最后一次错误可查询）。
完成条件：磁盘满场景的 SCENARIOS 案例可观察到失败信号。

## T-005：时间轮转边界语义混合（UTC 对齐 vs 本地日历）

状态：候选未验证（代码现状 A：rotate.go calcNextRotation——Daily 用 Truncate(24h) 即 UTC 对齐边界；Weekly/Monthly 用本地日历日期）。
影响：东八区下 Daily 轮转发生在早 8 点而非本地午夜；与 Weekly/Monthly 行为不一致。
动机判断（C）：疑似未意识到 Truncate 语义而非有意设计——待确认。
下一步：用固定时区测试验证边界时刻。
完成条件：确认后统一为本地日历或文档标注 UTC 语义。

## T-006：doc.go「runtime SetLevel」与无锁实现矛盾

状态：已确认待处理（A：doc.go L135 vs sink.go SetLevel 裸写；D-006 设计假设初始化期）。
影响：按文档在运行时并发调级会构成数据竞争（-race 可报）。
下一步：要么 level 改 atomic，要么改文档。
完成条件：文档与实现一致，加并发调级回归测试。

## T-007：README.md 与 AGENTS.md 的级别表述过时

状态：已确认待处理（范围更新：CLAUDE.md 已改为指向 AGENTS.md 的 symlink 并入其内容，
旧清单里「slog/ 包、独立 go.mod」两处过时点随之消灭；残余两处——）
问题（A）：README 仍写 "Trace…Panic" 七级与 "guaranteed message delivery"；AGENTS.md
也写 "7 levels (Trace–Panic)"。实现是六级（Trace..Fatal，无 PanicLevel，
record.go 代码现状）；「guaranteed delivery」与 T-001 关闭后的残余边界（非 Fatal
硬退出）仍有出入。
影响：AI/新人按过时文档理解组件（正是本知识库要防止的场景）。
下一步：由维护者更新两份文档（本任务约束不修改仓库内文件）。
完成条件：级别清单改为六级；delivery 措辞对齐实现。

## T-008：全局 Setup 不关闭旧 sink；并发 Setup 与包级日志存在竞争窗口

状态：候选未验证（代码现状 A：Setup 仅替换 globalLogger；包级函数读 globalLogger 不加锁）。
影响：运行中重配会泄漏旧 sink 的文件句柄；并发 Setup+Info 理论上可触发数据竞争。
动机判断（C）：默认「init 期一次性 Setup」的使用假设——待确认。
下一步：-race 下复现；决定加 RLock 还是文档声明。
完成条件：行为有定论且文档同步。
