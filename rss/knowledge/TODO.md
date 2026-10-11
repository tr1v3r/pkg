# rss 组件待办事项

基准日期 2026-10-12（刷新 @ d6b2b9c）· 待办是背景信息，不是任何后续任务的授权清单。证据 A=代码现状，C=推测。

## T-001：嗅探把 RDF 判为 RSS，但 ParseRSS 解析不了 RDF

状态：已确认待处理（A：rss.go DetectFeedType 对 "RDF" 返回 FeedTypeRSS；RSS.XMLName 标签为 `xml:"rss"`，RSS 1.0 的 <rdf:RDF> 根元素无法匹配，Parse() 将返回 unmarshal 错误）。
影响：RSS 1.0 (RDF) feed 走 Parse() 必然报错；测试只覆盖了 DetectFeedType 层，未覆盖 Parse 层。
下一步：决定方向——移除 RDF 分支（诚实报 Unknown）或实现 RSS 1.0 结构映射。
完成条件：RDF 样本经 Parse() 的行为有测试锁定。

## T-002：ParseJSONFeed 不校验 version 字段

状态：已确认待处理（A：jsonfeed 解析对任意 JSON 对象成功，version 校验只存在于 DetectFeedType）。
影响：直接调用 ParseJSONFeed 时，非 feed 的 JSON（如普通配置对象）会得到空 Version 的"成功"结果。
下一步：与维护者确认容错边界（D-003 的适用范围）；若收紧，加 version 前缀检查。
完成条件：行为写入 SCENARIOS 并有测试。

## T-003：README 无 rss 章节

状态：已确认待处理（A：README.md 包列表仍缺 rss；CLAUDE.md 已并入 AGENTS.md，其
rss 条目准确）。
影响：包对外不可发现，文档两份不一致。
下一步：由维护者补 README 章节（本任务约束不修改仓库内文件）。
完成条件：README 与 AGENTS.md 均覆盖 rss。

## T-004：非 UTF-8 编码的 XML 未处理

状态：候选未验证（A：DetectFeedType 与 ParseRSS 均直接喂 bytes；encoding/xml 默认按 UTF-8）。
影响：带 `encoding="GBK"` 等声明的中文老站 feed 可能解析失败或乱码。
动机判断（C）：真实抓取源均为 UTF-8，故未触发——待确认消费场景。
下一步：构造 GBK 样本验证失败形态。
完成条件：确认后决定加 charset 探测层或文档标注仅支持 UTF-8。

## T-005：互转中非法日期原样透传，产出不合规范的日期字段

状态：已确认待处理（影响面 2026-08-24 起大幅收窄——rfc822ToRFC3339 从 2 布局扩到
12 布局表 + 注释剥离后，真实世界形态基本全转换成功；残余：A：rfc822ToRFC3339/
rfc3339ToRFC822 全不匹配时仍返回原串，调用点直接赋值）。
影响：下游拿到 `date_published: "invalid"` 之类不合规值（D-003 容错哲学的副作用）。
下一步：与维护者确认是否需要 strict 变体（返回 error 或跳过字段）。
完成条件：D-003 补充该边界的重新评估结论。

## T-006：去重的 in-place 语义可能被误用 ✅已关闭

状态：已完成（2026-08-24 b1d70ff）
结果：DeduplicateItems 改为新建切片（fresh backing array），调用方旧引用不再被
覆写——别名风险根除而非仅文档化。D-004 的 in-place 细节标记失效。
