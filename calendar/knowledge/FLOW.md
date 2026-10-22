# calendar — 流程图

更新：2026-10-12（刷新 @ d6b2b9c）· 证据：代码现状 + b1d70ff/ad4ce80/e0a1fbf 提交 ·
定位：ICS 生成 + 宽松解析，不计算重复实例

```text
【生成主路径】calendar.go
  NewCalendar(name, desc, opts) → 默认 PRODID/VERSION:2.0/CALSCALE:GREGORIAN/METHOD:PUBLISH
    + CalendarOption（WithTimeZone 只写 X-WR-TIMEZONE 扩展属性，不改事件时间）
  AddEvents/AddTodos/AddJournals/AddTimezones → Output()
  Output 顺序固定：BEGIN:VCALENDAR → 日历级属性 → VTIMEZONE* → VEVENT* → VTODO* →
    VJOURNAL* → END:VCALENDAR；末尾整体过 foldLines（b1d70ff）
  文本值输出转义：SUMMARY/DESCRIPTION/LOCATION/COMMENT/CONTACT/RESOURCES/
    CALNAME/CALDESC 经 EscapeText（RFC 5545 §3.3.11：\\ \; \, \n）
  foldLines：>75 octet 的行折行，续行以单空格开头，UTF-8 不劈 rune（rune-safe）

【Event 构造】NewEvent(sum, desc, start, opts)
  默认：DTSTART=NewDate(LayoutTimeUTC) / CREATED=now / TRANSP=TRANSPARENT
  EventOption 逐个改写：WithEnd/WithRRULE/WithAttendee…；
    SetStartFormat/SetEndFormat 修改 Date.layout 并追加 ;参数（如 VALUE=DATE、TZID=…）

【Date 输出语义】models.go Date.Output —— 三分支（b1d70ff + ad4ce80）
  layout==LayoutTimeUTC（带 Z 字面）→ t.In(UTC) 后格式化（布局自称 UTC，必须先折算；
    修前 +08 值打成 090000Z 冒充 UTC，实际是 010000Z）
  否则 Location≠Local 或 hasExplicitZone → In(wireLocation) 后按原时区墙钟输出
    （floating 时间保持墙钟；VALUE=DATE 保持日历日期不跨日；TZID 时间在命名时区重渲染）
  其余（本地无显式时区）→ 按原值格式化
  常量更名：LayoutTime="…Z" 旧义拆为 LayoutTimeUTC；新 LayoutTime=无 Z 的 floating 布局；
  NewDate/NewDateList 仍产 UTC 形（兼容）

【解析主路径】parse.go
  Parse/ParseReader → 逐行扫描 → unfoldLines（RFC 5545 §3.1：空格/Tab 开头的行并入上一行）
  → parseCalendar 状态机：
    BEGIN:VCALENDAR/VEVENT/VTODO/VJOURNAL/VTIMEZONE/VALARM/STANDARD/DAYLIGHT → 压栈上下文
    BEGIN:其他        → skipDepth++（未知组件整体跳过，其内属性行忽略，END 配对弹出）
    END:对应组件      → 弹栈；但仅当该组件确实处于打开状态（e0a1fbf）——
                        孤儿 END（截断/手改/恶意 ICS）跳过，不再 nil 解引用 panic；
                        VALARM 加 gate inAlarm；STANDARD/DAYLIGHT 还要求 VTIMEZONE 在开
    属性行            → parseProperty 切 (name, params, value)——引号感知：
                        CN="Smith;John" 内的分号不切分（b1d70ff）
                        → 按当前上下文分派 setCalendarField/setEventField/setAlarmField/…
  嵌套优先级：inAlarm > inTZProp > inEvent > inTZ > inTodo > inJournal > 日历级
  TEXT 值解析侧统一 UnescapeText（desc/location/summary/comment/contact/alarm 等）

【日期解析】parseDate / parseDateList（b1d70ff 重写）
  VALUE=DATE 参数 → LayoutDate；否则按值形状选：尾 Z → LayoutTimeUTC；
  长度==8 → LayoutDate；其余 → LayoutTime（floating）
  TZID=… 参数 → 记入 Date.tzid；floating 值 + TZID → LoadLocation 成功则把墙钟
    绑定到该时区（instant 正确），Output 再按该时区重渲染墙钟；LoadLocation 失败
    仍得 floating 零偏移值
  time.Parse 失败 → 返回零值 Date（保留 tzid），不报错（静默降级）

【异常路径】
  Parse 仅在 io 读失败时返回 error；结构错乱/坏日期/坏数字一律静默（Atoi 失败跳过）
  未知属性 → 对应 setXxxField 的 switch 落空 → 忽略；孤儿 END 跳过（同此哲学）
  畸形 GEO（非 lat;lon）→ 零值；CATEGORIES/EXDATE/RDATE 按 , 切分
  round-trip（Output→Parse）：转义/折行双向均已实现（EscapeText↔UnescapeText、
  foldLines↔unfoldLines），多字节内容 round-trip 已验证（output_fold_test.go）
```
