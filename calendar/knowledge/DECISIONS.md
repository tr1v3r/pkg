# calendar — 架构与选型说明

更新：2026-10-12（刷新 @ d6b2b9c）· 证据等级：A=代码现状/提交可回查，C=推测（维护者不在场）

## D-001：日期输出恒转 UTC（单行修复定下的全局语义）⚠️已失效

状态：已失效（2026-08-24 b1d70ff + 2026-09-14 ad4ce80 起，由 D-007 取代）。
历史内容：`Date.Output` 曾用 `d.UTC().Format(layout)`（91ccccb，2024-06-19 单行修复，
治「本地时区值打 Z 后缀冒充 UTC」）；其副作用（TZID 语义错位、VALUE=DATE 跨时区移日）
曾被本知识库记为 T-001/T-002。b1d70ff 把无条件 UTC() 改为「保留各值自身时区」，
ad4ce80 再补「布局自带 UTC 指示符时先折算到 UTC」分支——91ccccb 想治的病由
LayoutTimeUTC 分支以正确方式复治。条目保留作失效记录，护栏改挂 D-007。

## D-002：RRULE/EXDATE/RDATE 只透传，不展开

选择：RRULE 是字符串字段 + `WithRRULE`；EXDATE/RDATE 是时间列表；无任何重复实例
计算、无语法校验。
证据（A）：31aa117 提交信息（"type definition, output method, and WithRRULE option"）；
代码现状。
权衡：本包保持纯序列化职责；展开计算交给消费方（日历应用）。
重新评估条件：若需要"第 N 次出现"类查询，另建展开模块，勿混入 Output 路径。
状态：生效中。

## D-003：解析宽松——错误不冒泡

选择：坏日期 → 零值 Date；坏数字 → 跳过；未知组件/属性 → 忽略；整个 Parse 仅在
io 错误时返回 error（parse.go）。
证据（A）：代码现状。动机（C）：推测为订阅源良莠不齐时保吞吐、不因单条坏行弃整份日历。
权衡：容错高；静默丢数据难以察觉（消费方必须防御零值）。
重新评估条件：需要严格校验的场景应加 `ParseStrict` 类新 API，不改现有语义。
状态：生效中。

## D-004：输出无折行、无文本转义 ⚠️已失效

状态：已失效（2026-08-24 b1d70ff 起）。现行事实：TEXT 值输出经 EscapeText、解析经
UnescapeText（双向）；Calendar.Output 末尾 foldLines 折到 RFC 5545 §3.1 的 75 octet
（rune-safe 续行，多字节 round-trip 有测）。README「RFC 5545 compliant」措辞与实现的
差距由此大幅收窄（残余差距见 T-005：宽松解析、RRULE 不展开）。原条目（「直接拼接、
无折行无转义」）保留为失效记录。

## D-005：双向 API 以 round-trip 为验收口径

选择：Parse 定位为 "reverse of Output"（1ed0110 标题原文），cb007c4 的测试策略也是
round-trip 覆盖。
证据（A）：两个提交信息；parse_test.go 的用例形态。
护栏：新增字段时应同步补「生成→解析→字段相等」用例，这是本包的主要回归网。
重新评估条件：若开始解析第三方 ICS（非本包 Output 产物），round-trip 口径不足，
需按 RFC 边界单测。
状态：生效中。

## D-006：类型化字符串 + 手写 Output，零反射

选择：Summary/Class/Status 等均为 string 别名，各自带 `Output() []byte` 方法直接拼字节；
组件结构体显式持有 header/tailer 与全字段（models.go / calendar.go）。
证据（A）：代码现状。动机（C）：推测追求零反射直出与字段可见性。
权衡：加字段要动三处（struct/Output/parse 的 setXxxField），漏一处即 round-trip 断链。
重新评估条件：字段继续膨胀（Event 已 25+）时可考虑表驱动注册。
状态：生效中。

## D-007：Date 输出按「值的时区语义」渲染，UTC 折算仅发生在布局自带 UTC 指示符时

选择：`Date.Output` 三分支——(a) `layout == LayoutTimeUTC` → 先 `In(time.UTC)`；
(b) 非本地时区或显式时区（TZID/非零 offset）→ `In(wireLocation)` 按该时区墙钟输出；
(c) 其余按原值。DATE 保持日历日期、floating 保持墙钟、TZID 值在命名时区重渲染。
证据（A）：b1d70ff（2026-08-24，#24「output now preserves each value zone」）+ ad4ce80
（2026-09-14，BUG-AUDIT-REPORT H8：+08 输入曾被 NewEvent 渲染成 090000Z，实为 010000Z；
非 UTC 机器上所有 NewEvent 含内置 createdAt=time.Now() 均中招）。
权衡：语义按值逐类正确，但正确性依赖「构造时 time.Time 落在正确时区」——TZID 参数
只声明、不搬运时间（生成侧 setTimeFormat 不写 tzid 字段，tzid 仅解析侧产生）。
护栏：动 Output 的时区分支前必读 ad4ce80 的四组回归（正/负/半小时/跨日 offset、
DATE、floating、TZID 墙钟）；NewDate/NewDateList 仍产 UTC 形是兼容承诺。
重新评估条件：若生成侧要让 TZID 参数自动把 instant 搬运到目标时区（写 d.tzid），
需同步 DateList 与 round-trip 用例。
状态：生效中（2026-09-14 起）。

## D-008：孤儿 END 行跳过而非报错（解析健壮性优先）

选择：parseCalendar 的每个 END 分支 gate 在「组件确实打开」上（inEvent/inTodo/
inJournal/inTZ/inAlarm；STANDARD/DAYLIGHT 还要求 VTIMEZONE 活着）；畸形 END 静默
跳过，Parse 签名与错误行为不变。
证据（A）：e0a1fbf（2026-09-14，BUG-AUDIT-REPORT H10）——修前孤儿 END 解引用 nil
组件指针直接 panic，攻击者可控/截断/手改 ICS 可崩进程；提交还自报修了报告未列的
一条（事件内游离 END:VALARM 的 nil alarm）。
权衡：与 D-003 宽松哲学一致（跳过而非报错）；完整错误报告被明示为独立的中等级发现，
留待后续。护栏：加新组件类型时 END 分支必须带同样的 gate（parse_malformed_test.go
的 20 例表是回归网）。
重新评估条件：实现结构化错误报告（malformed 行可枚举）时与此不冲突。
状态：生效中（2026-09-14 起）。
