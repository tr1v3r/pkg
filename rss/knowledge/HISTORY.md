# rss 组件修改历史

基准日期 2026-10-12（刷新 @ d6b2b9c）· 证据分级：A=可回查提交/PR，C=推测（待维护者确认）。包始于 OPML 结构，半年内长成四格式解析+互转+schema 全覆盖。

## H-001：起点——仅 OPML 结构定义

日期 2026-10-12 · 证据 A：05086c9（#19，2024-10-15）。
变化：Outline/OutlineArray 结构 + AddOutline 分组追加（当时含 Append→AddOutline 前的演化子提交）。
当前影响：OPML 是包内最老部分，API 风格（值接收者数组、静默 no-op）与后来解析函数不同。

## H-002：补齐解析入口与字段，命名清理

日期 2026-10-12 · 证据 A：f6c0906（2026-04-09）。
变化：加 ParseRSS/ParseAtom/ParseOPML/Parse 自动嗅探；Item 补 Content(content:encoded)/GUID/Author，Entry 补 ID/Summary/Content/Author；Atom Link 支持多条 + AlternateLink() 助手；OutlineArray.Append 改名 AddOutline。
动机（A）：提交信息「Rename … for clarity」。
当前影响：AlternateLink 的回退规则（无 alternate 取第一条 link）是消费方取链接的约定入口。

## H-003：JSON Feed 1.1 支持接入嗅探

日期 2026-10-12 · 证据 A：23cb72b（2026-04-14）。
变化：JSONFeed 类型族；DetectFeedType 增加 `{` 开头 + version URL 前缀的 JSON 探针；Parse 分派 FeedTypeJSON。
当前影响：版本判定只看前缀（1.0/1.1 通吃）；ParseJSONFeed 本身不校验 version（TODO T-002）。

## H-004：三向互转与日期桥

日期 2026-10-12 · 证据 A：11df767（2026-04-14）。
变化：RSS/Atom→JSONFeed、JSONFeed→RSS/Atom；为「lossless conversion」补 Enclosure/Category/Published/AtomCategory/Author.URI 字段；RFC822⇄RFC3339 互转（失败原样返回）。
动机（A）：提交信息明写「needed for lossless conversion」「following the official JSON Feed mapping spec」。
当前影响：round-trip 测试（TestRoundTrip_RSS/Atom）是互转行为的回归基线；实际映射存在已知有损点（D-005）。

## H-005：channel.image → JSON Feed icon 的单向补全

日期 2026-10-12 · 证据 A：27ff9b5（2026-04-15，仅一行标题无正文）。
变化：RSS→JSONFeed 时 Channel.Image.URL 映射为 icon。
动机（C）：推测为补全 JSON Feed 官方映射的遗漏项——无提交正文佐证——待确认。
当前影响：icon 是单向映射（JSONFeed→RSS 不回填 image）。

## H-006：频道内去重

日期 2026-10-12 · 证据 A：8147c01（2026-04-17）。
变化：Channel.DeduplicateItems，GUID 优先/Link 回退/keep-first/in-place（D-004）。
动机（C）：推测服务于「同一 feed 刷新合并」的消费场景——提交信息只描述行为——待确认。
当前影响：这是包内唯一与「刷新」相关的辅助；跨快照比较仍属调用方（FLOW 刷新语义边界）。

## H-007：正确性修复 + 协议 schema 全覆盖 + ParseTyped（2026-08-24，b1d70ff / #24）

变化（A，提交信息逐条）：
1. RFC 822 日期——rfc822ToRFC3339 从 2 布局扩到 12 布局表 + 剥 "(CST)" 注释；修前
   RSS 2.0 规范格式（两位年）与真实世界形态（无星期/UT·Z/单位数日）全部静默透传不转换。
2. 去重别名安全——DeduplicateItems 弃 Items[:0] 复用，新建切片（D-004 部分失效）。
3. ParseTyped[T]——any 入口的编译期定形替代（D-002 补充）。
4. schema 全覆盖——RSS 2.0/Atom/OPML 可选元素全量建模 + 转换产 RFC 合法输出；
   含两处 source breaking（Item.GUID→*GUID、Categories→Category，见 D-007）。
当前影响：rss 从「能解析主流字段」升到「可直接投产」的自述口径；dates_typed_test.go
与 schema_complete_test.go 是新增回归网。
关联（A）：b1d70ff。
