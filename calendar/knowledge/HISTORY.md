# calendar — 修改历史

更新：2026-10-12（刷新 @ d6b2b9c）· 证据等级：A=提交信息/代码可回查，C=推测。
完整流水：`git log --oneline -- calendar/`。

## H-001：初始实现（2024-06-17，8996c02 / #17）

变化：VCALENDAR/VEVENT 生成能力（NewCalendar/NewEvent/Output + 函数式选项）。
背景（A）：docs.go 保留的注释显示参考了 Google Calendar 导出的 ICS 样例与
yulanggong 的 gist——X-WR-CALNAME/X-WR-TIMEZONE 等扩展属性即源于此。
当前影响：字段中文注释、X-WR-* 扩展属性、`MetohdRequest` 拼写（typo 已成导出 API）
均沿用至今。
关联（A）：8996c02。

## H-002：时区单行修复（2024-06-19，91ccccb）

变化：`Date.Output` 由 `d.Format(layout)` 改为 `d.UTC().Format(layout)`。
原因（A）：提交标题 "fix date timezone mistake"——修前本地时区值会被打上 Z 后缀冒充 UTC。
当前影响：这是 D-001 全部边界（TZID 语义错位、全天事件移日）的单一来源；任何时区
行为讨论都应从这一行开始。
关联（A）：91ccccb（单行 diff 可直接回查）。

## H-003：RRULE 透传支持（2026-04-08，31aa117）

变化：Event 增加 RRULE 字段 + Output + WithRRULE；同时把 .claude/ 加进 .gitignore。
当前影响：重复规则仅序列化，无展开（D-002）；该提交也是本包进入 AI 协作期的标志。
关联（A）：31aa117 提交信息。

## H-004：解析能力（2026-04-09，1ed0110）

变化：新增 Parse/ParseReader——"reverse of Output"；含折行还原、date-only 识别、
未知组件跳过。
当前影响：包从单向生成变为双向；宽松解析语义（D-003）随之确立。
关联（A）：1ed0110 提交信息。

## H-005：全量 RFC 5545 铺开（2026-04-09，cb007c4 / #20）

变化：VTIMEZONE/VALARM/VTODO/VJOURNAL 组件 + 17 个新 Event 字段（ATTENDEE/ORGANIZER/
EXDATE/RDATE/GEO/ATTACH…）+ 解析器与选项同步扩展 + round-trip 测试 + 字符串常量化。
当前影响：现行字段面与解析状态机全部来自此提交；H-002 的 UTC 语义在此扩展中被
原样继承到所有新日期字段（未重新审视，C：推测量大从简）。
关联（A）：cb007c4 提交信息（逐条列出新增字段）。

## 演变主线一句话

生成（2024）→ 时区规整为 UTC（2024）→ 透传重复规则（2026）→ 可解析（2026）→
全量字段（2026）→ RFC 正确性四连修 + 协议 schema 补全（2026-08~09）；时区语义从
「一律 UTC」进化为「按值语义渲染 + UTC 指示符例外」（H-006/H-007）。

## H-006：RFC 5545 正确性四连修 + 输出转义折行（2026-08-24，b1d70ff / #24）

变化：(1) TZID 解析——parseDate 按值形状选 UTC/floating/DATE 布局并绑定 TZID 时区，
Output 在命名时区重渲染墙钟（修前 TZID 本地时间完全解析不了）；(2) Date.Output 弃
无条件 UTC()，保留各值时区（DATE 不再跨区移日、floating 保持墙钟）；(3) 文本转义
EscapeText/UnescapeText 双向 + 全 TEXT 字段接线（修前含逗号/分号/换行的 DESCRIPTION
round-trip 损坏或截断）；(4) parseProperty 引号感知（CN="Smith;John" 不再切错）。
附带：Calendar.Output 折行 75 octet（rune-safe）；常量重排 LayoutTimeUTC/LayoutTime。
证据（A）：PR #24 提交信息逐条「verified against RFC 5545」。
当前影响：D-001/D-004 失效、D-007 生效；T-001/T-002/T-003 关闭；S-007/S-008 语义
反转（见 SCENARIOS 注记）。
关联（A）：b1d70ff。

## H-007：UTC 指示符布局先折算（2026-09-14，ad4ce80）

变化：Date.Output 增加 `layout == LayoutTimeUTC → t.In(time.UTC)` 首选分支。
原因（A）：BUG-AUDIT-REPORT H8——b1d70ff 的「保留值时区」对自带 Z 字面的布局是错的：
+08 的 9 点被写成 090000Z（实为 010000Z）；非 UTC 机器上所有 NewEvent（含内置
createdAt）中招；测试全用 time.UTC 构造把病藏住了。
当前影响：D-007 的分支 (a)；date_output_utc_test.go 钉死正/负/半小时/跨日 offset
与 CREATED 路径（测试里 pin time.Local，calendar 测试不并行）。
关联（A）：ad4ce80。

## H-008：孤儿 END 不再 panic（2026-09-14，e0a1fbf）

变化：全部 END 分支 gate 组件打开状态；畸形 END 跳过。修前孤儿 END 解引用 nil
指针 panic——解析不可信 ICS 即崩进程（BUG-AUDIT-REPORT H10）；事件内游离
END:VALARM 是报告未列、提交自查补修的额外 panic 路径。
当前影响：D-008；parse_malformed_test.go 20 例 + 良构对照例锁定。
关联（A）：e0a1fbf。
