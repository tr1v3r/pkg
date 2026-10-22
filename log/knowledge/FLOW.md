# log 组件流程图

基准日期 2026-10-12（刷新 @ d6b2b9c）· 证据：代码现状；历史策略延续标注提交号。

```text
【阶段1 记录生成】调用侧
  log.Info(msg, "k", v, log.Err(err)) / Infof / CtxInfo(ctx, …)   (log.go)
    │  Record{Time, Level, Message, Fields, LogID}   仅 Ctx* 系列从 ctx 提取 logID (record.go)
    │  Fields 来源：With() 预置 + 调用参数（交替 k-v 与类型化 Field 可混用）
    │  With() 语义（d6b2b9c 起）：把父级 fields + 新 fields 拷入全新精确尺寸切片——
    │    每个子 Logger 拥有独立存储；修前 append 复用父切片数组，两跳以上链必有富余
    │    容量，后建的兄弟静默覆写先建兄弟的字段（构造期写入，-race 不可见）
    ▼
【阶段2 派发】Logger.dispatch —— RLock 下遍历 sinks，逐个 s.Log(record)
  │
  ▼
【阶段3 Sink 内部】Sink.Log (sink.go)
  ├─ 级别过滤 record.Level < sink.level ──► 静默丢弃（每个 sink 独立级别）
  ├─ logFunc 已设置（SlogHandler 旁路）──► 直接调 logFunc，不走编码/写出
  ├─ enc.Encode(record) → []byte   TextEncoder(彩色/纯文本) | JSONEncoder
  │
  ├── 同步模式 ch==nil（Console 默认）──► writer.Write(data) ──► 本次调用即落盘
  └── 异步模式 ch!=nil（File/RotateFile/SizeRotateFile 默认 buf=1024）
        s.ch <- data
        ├─ 缓冲有空位 ──► 入队即返回
        ├─ 缓冲已满   ──► 阻塞调用方（反压不丢日志，策略自 e8e37cf 延续）
        └─ ch 已 Close ──► panic: send on closed channel（Close 后不可再记日志）

【阶段4 异步排出】drain goroutine（每个异步 sink 一个，newSink 时启动）
    for data := range ch ──► writer.Write(data)
    └─ 写错误被忽略（drain 不检查返回值）──► 写失败=静默丢数据

【阶段5 轮转】写目标内部，惰性触发（只在有写入时检查）
  ├─ 时间轮转 rotateWriter.Write (rotate.go)
  │    now > nextRot ？ ──是──► 关旧文件 → 开 {dir}/{prefix}_{ts}.log → 更新 nextRot
  │    Hourly/Daily 边界 = Truncate(24h) 即 UTC 对齐；Weekly/Monthly = 本地日历（混合语义）
  │    openFile 失败被吞 → current 指向已关闭文件 → 后续写持续失败（静默）
  └─ 大小轮转 sizeWriter.Write (size.go)
       written+len > maxSize 且 written>0 ？ ──是──► seq++ 开 {prefix}_{seq:03}.log
       进程重启：seq = 目录内 max(seq)+1（findLastSeq 扫描）→ 每次重启开新文件
       单条记录 > maxSize：首条（written==0）不触发轮转，独占一个超限文件

【阶段6 冲洗/关闭】生命周期收尾
  ├─ Sync()  异步：ch <- nil 哨兵 → drain 排空队列 → 收到 done → 重建 done + 重启 drain
  │           同步：writer 实现 Sync() error 则透传调用（错误忽略）
  ├─ Fatal/CtxFatal/Fatalf/CtxFatalf → l.exit() = 先 l.Sync() 再 os.Exit(1)
  │           （9115d2d 起：异步 sink 的致命日志先排空再退出；修前直接 Exit 丢最后一条）
  └─ Close() close(ch) → 等 drain 排空（done）→ closer.Close()（关文件句柄）
       安全退出顺序 = 先 Sync/Close 再退出；Close 后再 Log 或 Sync ──► panic

【旁路 slog 兼容】slog.go（双向适配，均绕不过 sink 级别过滤）
  ├─ AsSlogHandler(sink)：slog.Handler 外壳 → Record{Fields} → sink.Log
  │    Enabled 恒 true；slog 级别映射后 Fatal 级不可达（最高映射到 ErrorLevel）
  └─ SlogHandler(handler)：Sink 伪装 → logFunc 构造 slog.Record → h.Handle
       该 sink 的 writer 为 nil ──► 当作 io.Writer 使用会 panic（不可接 gin.DefaultWriter）

【全局单件】globalLogger（默认 Console）
  Setup(sinks…) 原子替换（不关闭旧 sinks）→ 包级 Info/Infof/Ctx*/SetLevel 均转发全局
```
