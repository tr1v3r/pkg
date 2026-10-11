# calendar — 自然语言测试案例库

更新：2026-10-12（刷新 @ d6b2b9c）· 基线：d6b2b9c 代码现状。纯 Go 单测即可执行（无外部依赖）。

## S-001：基础事件生成

前置：`NewCalendar("cal", "d")`；`NewEvent("t", "desc", time.Date(2026,10,12,9,0,0,0,time.UTC))`
+ `WithEnd(...10点...)`。
动作：AddEvents 后 `Output()`。
预期：输出含 `BEGIN:VCALENDAR`/`BEGIN:VEVENT`；`DTSTART:20261012T090000Z`（UTC 带 Z）；
`TRANSP:TRANSPARENT`（NewEvent 默认）；组件以 `END:VEVENT`/`END:VCALENDAR` 闭合。
验证：逐行字符串包含断言。

## S-002：全天事件（VALUE=DATE）

前置：UTC 时间构造 start/end；`SetStartFormat(LayoutDate, "VALUE=DATE")` 与
`SetEndFormat(LayoutDate, "VALUE=DATE")`。
动作：Output()。
预期：`DTSTART;VALUE=DATE:20261012`（无时间无 Z）。
验证：字符串断言（参照 parse_test.go 既有 date-only 用例）。

## S-003：解析折行还原

前置：ICS 文本中 DESCRIPTION 值被折成两行，第二行以空格开头（RFC 5545 §3.1）。
动作：Parse(data)。
预期：Event.desc 为拼接后的完整值（续行前导空白被剥去）。
验证：字段相等断言。

## S-004：未知组件整体跳过

前置：VEVENT 内嵌 `BEGIN:X-CUSTOM` … `BEGIN:X-INNER` … `END:X-INNER` … `END:X-CUSTOM`，
其中含 SUMMARY:X 与其他属性行。
动作：Parse 后读取 Event.summary。
预期：SUMMARY:X 被忽略（skipDepth 生效），不污染 Event/Calendar 字段；解析无错误。
验证：字段仍为外层 VEVENT 的值。

## S-005：round-trip 保真（核心回归网）

前置：构造含 RRULE、EXDATE 列表、attendee、alarm、timezone 的复杂 Calendar；
SUMMARY/DESCRIPTION 含逗号、分号、换行；DESCRIPTION 长到触发折行（>75 octet，含多字节）。
动作：Output() → Parse() → 逐字段比较。
预期：组件数量、各字段（字符串/时间/列表）相等；RRULE 原样保留；转义文本双向保真
（EscapeText↔UnescapeText）；折行经 unfoldLines 还原（H-006 起）。
验证：字段级断言（cb007c4 建立的口径，新增字段必须同步加进来——见 D-005；
tz_escape_test.go / output_fold_test.go 已覆盖此两类）。

## S-006：坏日期静默降级

前置：ICS 含 `DTSTART:not-a-date` 与 `PRIORITY:abc`。
动作：Parse(data)。
预期：1. err == nil；2. Event.start.IsZero() 为真；3. priority 为 0。
验证：错误与字段断言（把 D-003 宽松语义钉住；若未来改严格解析，此用例必须同步分裂）。

## S-007：全天事件 + 非 UTC 时区构造不移日（正确性用例，b1d70ff 后转正）

前置：`time.Date(2026,10,12,0,0,0,0, time.FixedZone("CST", 8*3600))` 构造 start，
走 VALUE=DATE 输出。
动作：Output() 并读取 DTSTART 行。
预期：输出 `DTSTART;VALUE=DATE:20261012`（VALUE=DATE 保留构造时日历日期，D-007）。
验证：字符串断言（date_output_utc_test.go 的非 UTC 布局分支同源）。
注记：b1f8695 时代此用例期望 20261011（无条件 UTC 的行为基线）；b1d70ff 修正后
反转为正确语义，旧注记作废。

## S-008：TZID 本地时间解析与重渲染（b1d70ff 后为支持语义）

前置：ICS 含 `DTSTART;TZID=America/New_York:20260101T090000`（无 Z）。
动作：Parse(data) 后再 Output()。
预期：1. 解析成功且 instant 正确（墙钟绑定 New_York 时区，Date.tzid 记录参数）；
2. 重渲染仍为 09:00 墙钟（命名时区，非 13:00Z/UTC 折算）。
验证：字段断言 + Output 行断言（tz_escape_test.go / date_output_utc_test.go 的
TZID 分支）。
注记：b1f8695 时代此用例记录「解析失败静默清零」的行为基线，已作废。

## S-009：孤儿 END 不 panic 不造组件（e0a1fbf 回归）

前置：20 类畸形样本——各组件孤儿 END（含事件内游离 END:VALARM、VTIMEZONE 关闭后的
END:STANDARD）、重复 END、截断事件、嵌套 BEGIN:VEVENT。
动作：Parse(data)。
预期：1. 无 panic、无 error；2. 组件计数精确（stray END 不能伪造组件）；3. 事件不
挂幽灵 alarm；4. Output() 仍可用。
验证：parse_malformed_test.go 表驱动（每例断言计数）+ 良构对照例。

## S-010：NewEvent 在非 UTC 机器上的 UTC 渲染（ad4ce80 回归）

前置：测试内 pin `time.Local = +08:00`（calendar 测试不并行）；以 +08 时间 NewEvent。
动作：Output() 并解析 CREATED/DTSTART/DTEND/DTSTAMP 行。
预期：四个时刻均携带正确的 UTC 值（修前 CREATED 恰差 8 小时——布局带 Z 但值未折算）。
验证：date_output_utc_test.go（TestNewEvent_CreatedAtIsCorrectOnNonUTCMachine）。
