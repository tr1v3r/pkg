# rss 组件架构与选型说明

基准日期 2026-10-12（刷新 @ d6b2b9c）· 证据分级：A=代码现状/可回查提交，C=推测（待维护者确认）。

## D-001：仅用标准库 encoding/xml + encoding/json，不引第三方 feed 库

日期 2026-10-12 · 证据 A：go.mod 无任何 feed 解析依赖（代码现状）。
动机（C）：推测为控制依赖面——rss 从未引入 gofeed 之类库，具体取舍未见记录——待确认。
权衡：包体极小、无传递依赖；代价是没有实体修复、字符集转换、非规范 XML 容忍（TODO T-004）。
重新评估条件：需要解析「野」feed（坏实体、非 UTF-8、HTML 粘连）时，评估引入成熟解析器或扩展容错层。
状态：生效中。

## D-002：Parse() 返回 any，靠嗅探分派（泛型逃生口已补）

日期 2026-10-12 · 更新 2026-10-12（刷新 @ d6b2b9c）· 证据 A：rss.go Parse 签名
（f6c0906 引入、23cb72b 扩展 JSON）；b1d70ff 新增 `ParseTyped[T]`。
动机（C）：推测为避免定义三种类型的总和接口——待确认。
权衡：调用方必须类型断言（`result.(*RSS)`），编译期无保护；换来单入口极简 API。
现况：需要编译期类型时用 ParseTyped（T 不符给带 FeedType.String() 的明确错误），
any 形态保留为兼容主入口；两者共享同一嗅探与解析内核。
重新评估条件：若新增格式或调用方普遍抱怨断言，可引入 `Feed` 接口或泛型访问器。
状态：生效中（ParseTyped 为 2026-08-24 起的补充出口）。

## D-003：容错哲学——「能解析就解析，缺失即零值，失败才报错」

日期 2026-10-12 · 证据 A（代码现状）：字段缺失→零值；日期解析失败→原样字符串透传（convert.go rfc822ToRFC3339）；OPML AddOutline 对非法输入（Text/XMLUrl/groupText 为空）静默返回原数组；未知格式才报错且附前 64 字节预览。
权衡：对混乱的真实世界 feed 友好；代价是错误数据可能静默通过（如非规范日期透传进 JSON Feed，见 TODO T-005）。
重新评估条件：出现「需要严格校验」的消费方（如入库前校验）时，加 Strict 选项。
状态：生效中。

## D-004：去重键 = GUID 优先，空则 Link，双空保留（in-place 细节已废）⚠️部分失效

日期 2026-10-12 · 更新 2026-10-12（刷新 @ d6b2b9c）。
键策略不变（证据 A：8147c01 提交信息 + rss.go 代码），但「in-place 复用 Items[:0]」
细节自 b1d70ff（2026-08-24）失效：去重改为新建切片——旧实现会覆写调用方仍持有的
切片别名。现行为：keep-first 保序 + 零别名副作用，代价是每次去重一次分配。
护栏：不要再改回 in-place「优化」；T-006 随之关闭。
重新评估条件：需要跨 feed/跨快照去重或保留 last 语义时，另立 API。
状态：键策略生效中；in-place 细节已失效（由 b1d70ff 取代）。

## D-005：三向互转遵循 JSON Feed 官方映射规范，接受已知有损（损失面已收窄）

日期 2026-10-12 · 更新 2026-10-12（刷新 @ d6b2b9c）· 证据 A：11df767「following the
official JSON Feed mapping spec」；b1d70ff/#24 的字段补全与 RFC 合法输出。
已修复的旧损失点（A）：authors 复数化（原仅取首个）；icon↔image 双向（原单向）；
Atom content 的 type 不再丢弃（AtomText）；ToAtom 补必填 id/updated（有回退）；
GUID isPermaLink 语义双向推导。
仍接受的有损点（A 级代码现状）：JSONFeed→RSS 的 managingEditor 只取 authors[0]；
category.domain 仍只留 value；附件仍取首个（RSS enclosure 单数结构所限）。
权衡：覆盖主流字段即可往返（round-trip 测试锁定）；结构单复数不对称处显式放弃。
重新评估条件：消费方报告字段丢失影响业务时，按需补映射（每次补充须同步 round-trip 测试）。
状态：生效中（损失面 2026-08-24 起收窄）。

## D-006：包边界 = 纯解析/转换，无抓取与刷新状态

日期 2026-10-12 · 证据 A：包内无任何 HTTP、缓存、ETag、调度代码；仓库中 fetch/ 是独立包且未被 rss 引用。
动机（C）：推测为职责分离（解析与抓取解耦）——待确认。
当前影响：所有「刷新」语义（重抓、条件请求、增量比较、跨次去重）都在调用方；本包每次 Parse 是全新快照。
重新评估条件：若要提供 FeedRefresher 之类高层 API，应建新组件而非塞进本包。
状态：生效中。

## D-007：schema 面向协议规范全覆盖（含两处 source 级 breaking）

日期 2026-10-12（刷新 @ d6b2b9c 记录）· 证据 A：b1d70ff/#24「complete protocol schema
coverage for direct production use」+ schema_complete_test.go。
选择：RSS 2.0/Atom RFC 4287/OPML 2.0 的可选元素全量建模（见 FLOW【schema 面】），
转换携带新字段并产 RFC 合法输出。
代价（A，既成 breaking）：`Item.GUID` string→`*GUID{Value,IsPermaLink}`；
`Item.Categories` 改名 `Category`。外部使用方需同步；本仓内无其他包引用 rss，
波及面在兄弟项目（skr）。
护栏：给结构体加字段是安全的；再改导出字段名/类型属 breaking，须与使用方对表。
重新评估条件：无——这是能力补全的既成事实，记录供后续迁移参考。
状态：生效中（2026-08-24 起）。
