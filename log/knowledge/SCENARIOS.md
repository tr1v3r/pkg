# log 组件自然语言测试案例库

基准日期 2026-10-12（刷新 @ d6b2b9c）· 案例基于代码现状设计；标注「已有测试锁定」的条目存在对应 go 测试可复跑，执行结果三态：通过/失败/环境不满足。

## S-001：异步 File sink 经 Sync 后全部落盘

前置：临时目录；`File(path)` 默认异步 buf=1024。
动作：连续 Info 100 条，调用 `Sync()`，读文件。
预期：1) 100 条全部出现；2) Sync 返回后文件内容稳定（哨兵屏障语义，D-003）。
验证：文件行数统计；测试范本 TestFileSink。

## S-002：缓冲满时的反压（不丢不崩）

前置：`File(path)` 改用 `WithAsync(1)`（小缓冲）；writer 用可控延迟的自定义 io.Writer。
动作：发起第 2、3 条日志，观察 Log 调用是否阻塞；释放 writer 后再观察。
预期：1) 超出缓冲的 Log 阻塞而非丢弃或 panic（D-002）；2) 解除阻塞后无消息丢失、顺序保持。
验证：goroutine 计数 + 最终落盘条数与顺序。

## S-003：Fatal 在异步文件 sink 上的可见性（9115d2d 已修复，转为回归）

前置：子进程运行 `New(File(path)).Fatal("boom")`（异步默认）。
动作：子进程退出后读文件。
预期：1) Console 与 File sink 的 "boom" 均落盘（Fatal 先 Sync 再 Exit，H-007）；
2) 若人为去掉 exit 里的 Sync（回归验证），File 场景消息缺失——该负向形态即旧 T-001。
验证：文件 grep "boom"；两种 sink 对比。

## S-004：Close 的排空语义与关后行为（对应 T-002）

前置：异步 File sink，写入 3 条。
动作：调用 `Close()` 后读文件；随后再调用一次 `Info`。
预期：1) Close 返回时 3 条全部落盘；2) Close 后的 Info 触发 panic: send on closed channel。
验证：文件行数；recover 捕获 panic 类型。

## S-005：时间轮转的惰性触发与命名

前置：`RotateFile(dir, "app", Hourly)`。
动作：写入后把系统时间/注入时间推过边界（或用 Daily+短周期模拟），再写入；随后静默跨一个整周期不写，再写。
预期：1) 跨界后的首次写入落到新文件 `{prefix}_{ts}.log`；2) 静默期不产生空文件（轮转只在写入时检查）。
验证：目录文件列表与文件名时间戳。

## S-006：大小轮转的 seq 续接与超限单条

前置：`SizeRotateFile(dir, "app", 100)`（100 字节上限）。
动作：1) 写多条小记录至超限，观察 seq 递增；2) 重启进程（新建 sink 同参数）再写；3) 单次写入 200 字节。
预期：1) 依次产生 app_001/app_002…；2) 重启后从目录内 max seq+1 续写（不覆盖旧文件）；3) 200 字节单条独占一个文件（written==0 不触发轮转）。
验证：目录文件清单与各文件大小。

## S-007：slog → Sink 方向的级别与分组边界（D-005）

前置：`slog.SetDefault(slog.New(AsSlogHandler(fileSink)))`，sink 默认 Info 级。
动作：slog.Debug / slog.Info / slog 记录带 group 的 attrs / slog.Error+4（模拟 fatal 级）。
预期：1) Debug 被 sink 丢弃；2) Info 正常落盘；3) group key 输出为 `group.key=`（点连接）；4) Error+4 落盘为 ERROR 级（Fatal 不可达）。
验证：文件内容 grep；对照 TestSink_AsSlogHandler_WithGroup。

## S-008：Sink → slog 方向的字段透传

前置：`SlogHandler(slog.NewTextHandler(&buf, Debug))` + `Setup`。
动作：`log.Info("from our log", "key", "value")`。
预期：buf 中出现 `msg=... key=value`；slog 级别为 INFO。
验证：TestSlogHandler_AsSink 范本。

## S-009：Sink 作为 io.Writer 接管 gin 输出

前置：异步 File sink。
动作：`gin.DefaultWriter = sink`，gin 打一行请求日志，随后 Sync。
预期：gin 的原始输出（未结构化）经 Write 拷贝入队并落盘；与结构化日志同文件混排。
验证：文件含 gin 格式行。

## S-010：同 Logger 多 sink 独立级别

前置：`New(consoleSink(Debug 级), fileSink(Warn 级))`。
动作：依次 Debug、Warn。
预期：Debug 只进 console；Warn 两处都有（过滤在各自 sink，FLOW 阶段 3）。
验证：对照两个目标的输出（TestMultiSink 范本）。

## S-011：With 兄弟字段隔离（d6b2b9c 回归，已有测试锁定）

前置：父 Logger 以显式富余容量构造（确定性复现审计场景）。
动作：`base.With("k","X")` 与 `base.With("k","Y")` 各记一条日志；再按一/二/三/四跳
链与四层深链（每层带兄弟）重复；32 个 goroutine 并发建兄弟各记预设+调用字段。
预期：1) 先建兄弟记录 k=X 不被后建者覆写（修前为 Y）；2) 祖先字段全部存活；
3) 并发下 32 个 worker id 全部可区分（修前 -race 报 write/write 且丢一个 id）。
验证：with_test.go 五组用例；`With()` 空参、悬键、非 string 键的退化行为另有一组。
