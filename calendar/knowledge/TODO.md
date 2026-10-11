# calendar — 待办事项

更新：2026-10-12（刷新 @ d6b2b9c）· 待办是背景，不是本次任务的授权清单；执行前需维护者确认。

## T-001：TZID 本地时间双端不支持（输出错位、解析静默清零） ✅已关闭

状态：已完成（2026-08-24 b1d70ff）
结果：解析侧 parseDate 按值形状选布局 + TZID 绑定时区（LoadLocation 失败才降级）；
输出侧按值时区重渲染墙钟（D-007）。旧知识的「输出 UTC 值贴 TZID 声明」与「解析
静默清零」两个症状均消除。残余边界：生成侧 TZID 参数只声明不搬运——time.Time 必须
本身落在目标时区才正确（见 D-007 权衡）。

## T-002：全天事件（VALUE=DATE）跨时区移日 ✅已关闭

状态：已完成（2026-08-24 b1d70ff）
结果：Date.Output 弃无条件 UTC()，VALUE=DATE 保留构造时的日历日期。S-007 由
「行为基线用例」转正为正确性用例（期望 20261012）。

## T-003：折行与文本转义缺失（生成非法 ICS 的风险） ✅已关闭

状态：已完成（2026-08-24 b1d70ff）
结果：EscapeText/UnescapeText 双向 + foldLines 75 octet rune-safe 折行；含特殊字符
与多字节的 round-trip 有测（tz_escape_test.go / output_fold_test.go）。

## T-002b：孤儿 END 解析 panic（新增后即修）

状态：已完成（2026-09-14 e0a1fbf，原为外部审计发现而非旧知识库条目）
结果：END 分支全部 gate 组件打开状态，畸形 ICS 不再崩进程（D-008）。
注：旧知识库未记录此隐患——b1f8695 时代它确实存在，属当时审计盲区。

## T-004：calendar_test.go 无断言 ✅已关闭

状态：已完成（2026-08-24 9115d2d 覆盖率冲刺）
结果：calendar_test.go 重写为断言化（TestNewCalendarWithOptions/TestOutput/
TestNewEventWithOptions/TestModelsOutput 等 10 个测试），生成侧不再只靠 round-trip
间接覆盖。

## T-005：README「RFC 5545 compliant」强于实现（仓库文件，本任务不改）

状态：等待决策（差距已收窄，未归零）
问题：README 称 RFC 5545 compliance；折行/转义/TZID 已补（H-006/H-007），但解析
宽松（D-003）、RRULE 不展开（D-002）、孤儿 END 静默跳过（D-008）仍与「compliant」
的严格读法有距离。
证据（A）：文档现文 vs 代码。
下一步：由有权者决定措辞（补能力或改口径为「生成核心子集 + 宽松解析」）；
本次交付不修改仓库。
