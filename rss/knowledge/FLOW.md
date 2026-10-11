# rss 组件流程图

基准日期 2026-10-12（刷新 @ d6b2b9c）· 证据：代码现状 + b1d70ff 提交；schema 自 #24 起全覆盖。

```text
【阶段1 输入】调用方传入 []byte
  （本包无网络层：抓取、缓存、定时刷新全部由调用方负责）

【阶段2 格式嗅探】DetectFeedType (rss.go)
  ├─ TrimSpace 后为空 ─────────────────────────────► FeedTypeUnknown
  ├─ 首字节 '{' ──JSON 探针──► json.Unmarshal 成功 且 version 前缀
  │                                = "https://jsonfeed.org/version/" ──► FeedTypeJSON
  │            └─ 合法 JSON 但无该 version / 反序列化失败 ──► FeedTypeUnknown
  └─ 其他 ──XML 首元素扫描（跳过声明/注释等非 StartElement token）
                  ├─ local name = "rss" 或 "RDF" ──► FeedTypeRSS
                  ├─ local name = "feed"          ──► FeedTypeAtom
                  └─ 扫描到 EOF 未命中 / 解码出错 ──► FeedTypeUnknown
                        ⚠ RDF 被判为 RSS，但 ParseRSS 的 XMLName 只认 <rss>，
                          Parse() 对 RSS1.0/RDF 文档会返回错误（见 TODO T-001）

【阶段3 分派】Parse (rss.go) ── 返回 any（调用方类型断言）
  ├─ FeedTypeRSS  ──► ParseRSS      ──► *RSS{Channel{…, Items[]}}
  ├─ FeedTypeAtom ──► ParseAtom     ──► *Feed{Entries[]}（Atom 命名空间不敏感）
  ├─ FeedTypeJSON ──► ParseJSONFeed ──► *JSONFeed{Items[]}（不校验 version 字段）
  ├─ Unknown ──► error "unknown feed format: …"（附去空白后前 64 字节预览）
  └─ ParseTyped[T]（b1d70ff）：编译期定形（T ∈ *RSS/*Feed/*JSONFeed）；嗅探与 T
     不符 ──► error "feed is %s, not %s"（FeedType.String() 可读名）

【阶段4 解析容错】三种解析器共同哲学（A 级代码现状）
  ├─ 非法 XML/JSON ──► 包装错误返回（"parse RSS: %w" 等），不做修复重试
  ├─ 缺失字段 ──► 零值填充（无严格校验、无必填检查）；未知元素忽略
  ├─ 日期（Item.PubDate 等）──► 保留原始字符串，不在此阶段解析
  └─ 直接调 ParseRSS/ParseAtom/ParseJSONFeed 可跳过嗅探（各自独立入口）

【schema 面】#24（b1d70ff）起按协议规范全覆盖，schema_complete_test.go 锁定
  RSS 2.0：channel 全部可选元素（language/copyright/managingEditor/webMaster/
    pubDate/lastBuildDate/generator/docs/cloud/ttl/rating/textInput/skipHours/
    skipDays/category）；item 增 *GUID{Value,IsPermaLink}/comments/source/
    category 复数（⚠ Item.GUID 由 string 改 *GUID、Categories 改名 Category，
    两处 source 级 breaking）
  Atom RFC 4287：feed 必填 id/updated（此前无法表达）+ authors/contributors 复数
    + subtitle/icon/logo/rights/generator/xml:lang/base；entry 增 AtomText
    （type 感知 text/html/xhtml + src）、source、rights、多 author
  OPML 2.0：head 增 owner 三件/docs/expansionState/window 四件；outline 增
    description/category/created/language/version/isComment/isBreakpoint

【阶段5 后处理】
  ├─ 去重 Channel.DeduplicateItems（8147c01；b1d70ff 起新建切片，不再复用
  │     Items[:0] 底层数组——调用方持有的旧引用不被覆写，别名安全）
  │     key = GUID.Value（GUID 现为结构），空则 Link，两者皆空 ──► 无键保留
  │     重复 ──► 丢弃后到的；keep-first 且保序
  └─ 互转 convert.go（11df767 起；#24 后按 RFC 合法输出）
        RSS→JSONFeed：内容回退 content:encoded→description；image.url→icon；
          id 取 GUID.Value 回退 Link；comments→external_url
        Atom→JSONFeed：link rel 分派 alternate/self/hub→HomePageURL/FeedURL/Hubs；
          content 空或仅 src 时回退 summary；authors 复数映射
        JSONFeed→RSS：authors[0]→managingEditor；icon→image（双向打通）；
          GUID 带 isPermaLink 推导（ID==URL 时 true）
        →Atom：补必填 id/updated（有回退）；content/summary 按 AtomText 定 type
        日期 RFC822 ⇄ RFC3339：输入侧 12 布局表 + 剥尾部时区注释（"+0800 (CST)"），
          覆盖 RFC822 两位年/无星期/UT·Z 后缀/单位数日等真实形态；全不匹配 ──►
          原样字符串透传（不报错）

【刷新语义边界】（A 级代码现状）
  无状态：每次 Parse 产出独立快照；无 ETag/If-Modified-Since/增量 diff/缓存。
  周期刷新 = 调用方重新抓取+重新解析，新旧快照的合并、排序、去重跨次比较均自理；
  DeduplicateItems 只处理单个 Channel 内部的重复，不做跨快照合并。
```
