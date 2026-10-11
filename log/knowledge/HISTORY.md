# log 组件修改历史

基准日期 2026-10-12（刷新 @ d6b2b9c）· 证据分级：A=可回查提交/PR，C=推测（待维护者确认）。旧提交指 e1deaff 重写前的旧设计。

## H-001：整体重写，Handler 体系 → Sink 体系

日期 2026-10-12 · 证据 A：e1deaff（2026-04-20，PR #21，6 个子提交）。

变化：旧设计（ConsoleHandler/FileHandler/sync_console.go/structure.go/logger.go/level.go 及独立 log/go.mod、slog/ 包装包）全部删除，替换为 Sink 设计；notion/、websocket/ 调用方同步迁移。
动机（A）：PR 正文「Sink-oriented architecture」「native structured logging makes it unnecessary (slog/ 包)」。
当前影响：旧 API 无兼容层——任何基于旧 API 的代码必须重写；doc.go 成为唯一的 API 权威说明。

## H-002：重写前夜的大修（旧设计的终版）

日期 2026-10-12 · 证据 A：eb1318f（2026-04-09）。

变化：旧设计下修 Fatal/Panic 语义、FileHandler nextRotationTime 数据竞争（改 atomic.Value）、级别过滤缺失、Write 静默丢弃等 9 项。
动机（A）：提交信息逐项列出；定性为「重写前清障」。
当前影响：其中「级别是初始化期配置」的假设延续进新设计（D-006）；Panic 级别在新设计中被移除（只剩 Trace..Fatal 六级）——CLAUDE.md/README 仍写 7 级含 Panic，已过时。

## H-003：通道满阻塞而非丢弃（跨重写存活的策略）

日期 2026-10-12 · 证据 A：e8e37cf（2025-10-27）。

变化：旧设计移除异步发送的 default 分支，通道满改为阻塞。
动机（A）：提交信息「Prevents log loss under high load conditions」。
当前影响：新设计 Sink.Log 的 `s.ch <- data` 延续同一策略（D-002）——「日志零丢失优先于调用方延迟」是本包长期立场。

## H-004：关闭/死锁问题的多年拉锯（本包最高频事故类别）

日期 2026-10-12 · 证据 A：78570cc（#11，2023-10，Close 永久等待）、3fba6cc（2023-12，惰性 serve goroutine 导致死锁）、20269a0+87b0be1（2024-02，double-close 修复被整体 revert，未留原因）。

当前影响：新设计的哨兵屏障（D-003）正是对这一系列的回应；改动 Sync/Close/drain 任何一处都必须带 `-race` 全量测试，且要覆盖「Close 后再写」「Sync 与 Close 并发」两个历史雷区。

## H-005：输出错误检查的回归

日期 2026-10-12 · 证据 A：旧设计 3418dfe（#15，2024-02）曾专门加输出错误检查；新设计 drain() 对 writer.Write 返回值直接忽略（sink.go 代码现状）。

动机（C）：推测为重写时的简化，未在 PR 中讨论——待确认。
当前影响：磁盘满/权限错误时日志静默丢失；恢复输出错误检查需同时处理轮转失败路径（T-004）。

## H-006：独立模块并回主模块

日期 2026-10-12 · 证据 A：e1deaff 删除 log/go.mod 与根模块 replace，PR 正文「Merge log/ as a regular sub-package (remove separate go.mod, replace directive)」。

动机（C）：推测为降低双模块维护成本——PR 未说明为何放弃独立发布。
当前影响：`github.com/tr1v3r/pkg/log` 不再是独立模块，import 路径不变但版本跟随根模块；CLAUDE.md「separate go.mod entry」描述已失效（T-007）。

## H-007：Fatal 落盘保证 + 覆盖率冲刺（2026-08-24，9115d2d / #23）

变化：Fatal/CtxFatal/Fatalf/CtxFatalf 改走 `exit()`——先 `Sync()` 排空异步 sink 再
`os.Exit(1)`（提交信息 "flush async sinks before Fatal exits"）。同期整体覆盖率
71.7%→95.6%，新增 coverage_test.go/misc_test.go；文件 sink 权限 0644→0600。
当前影响：T-001 关闭、S-003 转为已修复行为的回归；README「guaranteed message
delivery」与现实的差距收窄到「非 Fatal 的硬退出路径」。
关联（A）：9115d2d。

## H-008：With 停止别名父级字段切片（2026-09-14，d6b2b9c）

变化：With 改为全量拷贝进精确尺寸新切片（空结果保持 nil）。
原因（A）：BUG-AUDIT-REPORT H4——append 复用富余容量时，同父的兄弟 Logger 构造
互相覆写字段；覆写发生在构造期，顺序构建时 -race 完全不可见，是「静默日志内容
伪造」而非崩溃。两跳以上 With 链因 append 容量倍增策略几乎必然留富余。
当前影响：D-008；with_test.go 五组回归（审计原场景/多跳表/深链/32 并发兄弟/空参
与残键）+ 负向验证记录（仅回退 With 体时哪些用例挂、-race 何时才报）。
关联（A）：d6b2b9c。
