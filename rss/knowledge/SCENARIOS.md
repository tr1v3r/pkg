# rss 组件自然语言测试案例库

基准日期 2026-10-12（刷新 @ d6b2b9c）· 基于代码现状设计。标注「已有测试锁定」的条目存在对应 go 测试可复跑；其余为探索性验收案例，执行结果三态：通过/失败/环境不满足。

## S-001：三种格式经 Parse 自动识别（已有测试锁定：TestParse_AutoDetect）

前置：RSS 2.0 / Atom / JSON Feed 1.1 样本各一。
动作：`Parse(data)`。
预期：分别返回 *RSS / *Feed（Atom）/ *JSONFeed，无错误。
验证：类型断言成功。

## S-002：非 feed 内容的报错形态

前置：`<html><body>hello</body></html>` 与 `{"hello":"world"}`（合法 JSON 但无 version）。
动作：`Parse(data)`。
预期：两者均返回 error；HTML 场景错误信息含原文前 64 字节预览（"unknown feed format: …"）；JSON 场景同为 unknown（嗅探层拒绝，不进 JSONFeed 解析）。
验证：错误串内容；对照 TestParse_UnknownFormat / TestDetectFeedType。

## S-003：字段缺失的零值容错（已有测试锁定：TestParseRSS）

前置：item 无 guid/author/pubDate 的 RSS。
动作：ParseRSS 后读取 Item 字段。
预期：解析成功；缺失字段为空字符串零值，不报错。
验证：TestParseRSS item[1] 断言。

## S-004：RSS 1.0 (RDF) 走全链路（对应 TODO T-001）

前置：`<rdf:RDF …><channel/></rdf:RDF>` 样本。
动作：先 `DetectFeedType` 再 `Parse`。
预期：Detect 返回 FeedTypeRSS；Parse 返回 unmarshal 错误（根元素与 `xml:"rss"` 不匹配）。
验证：两步返回值分别断言——这是「嗅探与解析能力不对齐」的已知边界。

## S-005：去重的键选择与保序（已有测试锁定：TestDeduplicateItems_Mixed）

前置：Channel.Items 混合：重复 GUID、无 GUID 重复 Link、双空项。
动作：`ch.DeduplicateItems()`。
预期：GUID 相同只留首条；GUID 空时按 Link 判重；GUID 与 Link 皆空的项全部保留；输出顺序与首次出现顺序一致；调用方预先留存的旧 Items 引用内容不变（b1d70ff 起新建切片）。
验证：TestDeduplicateItems_Mixed 断言（A/B/C/E/F → 4 条）+ 去重前后旧引用对比。

## S-006：同一 feed 两次解析相互独立（刷新语义基线）

前置：同一 RSS 文本。
动作：两次 `ParseRSS`，随后修改第一次结果的 Items（如清空），再检查第二次结果与原文本再解析结果。
预期：两次结果互不影响——包内无任何缓存/状态（FLOW 刷新语义边界）。
验证：修改一份数据后另一份不变。

## S-007：周期刷新的调用方合并模式（组合验收）

前置：同一源相隔一小时的两次快照 A、B（B 含新条目 + A 中旧条目重复出现）。
动作：调用方按 `GUID（空则 Link）` 对 A∪B 去重（模拟自身逻辑）；对单次快照内部调用 DeduplicateItems。
预期：单快照内重复被本包消除；跨快照合并结果由调用方逻辑决定（本包不提供）。
验证：合并后无重复 GUID；顺序符合调用方策略。

## S-008：互转 round-trip 保核心字段（已有测试锁定：TestRoundTrip_RSS/Atom）

前置：含 enclosure/category/author 的 RSS。
动作：`ToJSONFeed()` → `ToRSS()`。
预期：title/link/guid/enclosure/category value 往返保持；content:encoded→content_html
→Content 链路保持；authors 复数与 icon↔image 双向已打通（H-007）；仍已知丢失：
category.domain、多 author→RSS 单 author 字段（D-005 现行损失面）。
验证：round-trip 测试断言。

## S-009：非法日期的透传（对应 TODO T-005，影响面收窄后）

前置 A：`pubDate = "02 Jan 06 15:04 -0700"`（规范 RFC 822 两位年）；
前置 B：`pubDate = "Mon, 2 Jan 2006 15:04:05 +0800 (CST)"`（注释+单位数日）；
前置 C：`pubDate = "not-a-date"`。
动作：各自 `ToJSONFeed()`。
预期：A/B 均成功转为 RFC3339（b1d70ff 的 12 布局表 + 注释剥离）；C 原样透传
"not-a-date"，不报错。
验证：输出 JSON 检查 date_published；A/B/C 三态对照（dates_typed_test.go 覆盖日期族）。

## S-010：OPML AddOutline 的静默 no-op（已有测试锁定：TestAddOutline_*）

前置：OutlineArray；三种非法输入——o.Text 为空、o.XMLUrl 为空、groupText 为空。
动作：`AddOutline(group, o)`。
预期：均返回原数组不变，不 panic 不报错（D-003 容错哲学在 OPML 侧的体现）。
验证：TestAddOutline_InvalidOutline / TestAddOutline_EmptyGroupText。

## S-011：ParseTyped 的类型不符错误（b1d70ff 新行为）

前置：Atom 样本；`ParseTyped[*RSS](data)`。
动作：调用并检查错误。
预期：error 形如 "ParseTyped: feed is Atom, not RSS"（FeedType.String() 可读名）；
不 panic、不返回部分结果。T 为非法类型（如 *int）时给出参数错误。
验证：类型断言 + 错误串匹配（dates_typed_test.go）。
