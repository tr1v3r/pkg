# websocket 知识库（合并版五件套）

组件：gorilla/websocket 的薄服务端/客户端封装。服务端挂 Gin，每连接串行 echo 式处理；客户端提供泵式读、写、优雅关闭。
更新日期：2026-10-12（刷新 @ d6b2b9c）。证据等级：A=代码现状/可回查提交，C=推测。包地图见仓库 ARCHITECTURE.md（A 级来源）。

## 流程

```text
【服务端连接生命周期】(server.go)
gin 路由 → WSHanlder(handler) / WSHanlderWithUpgrader(自定义 upgrader, handler)
  → upgrader.Upgrade（失败：gorilla 已写 HTTP 错误响应，handler 直接 return）
  → defer ws.Close()
  → 循环: ReadMessage → handler(ws, msg) → WriteMessage(同消息类型 mt)
      任一读/写错误 → log.Warnf + break → defer 关连接
  无 ping/pong 保活、无读写 deadline；每连接一个 gin handler 协程，串行一问一答

【客户端】(cilent.go，文件名拼写刻意保留，见决策 D-005)
ConnectWebsocket: DefaultDialer.DialContext(ctx, url, header)
Read(conn): 起协程把消息泵入 cap=64 的 chan；读错误 → log.Infof 后 close(chan)（错误本身不外传）
Write(conn, msg): 固定 TextMessage 发送
Close(conn): 发 CloseNormalClosure 关闭帧 → 等固定 1s（让服务端先关）→ 返回；不调用 conn.Close()
```

## 决策

- D-001 默认共享零值 upgrader：CheckOrigin 为 nil → 沿用 gorilla 同源策略，浏览器跨域连接被拒（2026-10-12，A：代码现状）。跨域需求的出口是 WSHanlderWithUpgrader 传入自定义 upgrader（该变体存在的原因 (C)：推测即为放开 CheckOrigin）。重新评估：若默认场景是跨域，可考虑默认放行 + 文档警示。
- D-002 服务端串行一问一答：handler 阻塞式循环，不在单连接上并发派发消息（2026-10-12，A：代码现状）。影响：客户端流水线化请求会被队头阻塞。
- D-003 Read() 吞掉错误：消费者只见 chan 关闭，错误只进日志（2026-10-12，A：代码现状）。重新评估：需要错误信号时需改返回 (chan, errChan) 类接口。
- D-004 Close() 只发关闭帧 + 固定 1 秒等待，不关闭底层连接（2026-10-12，A：代码现状；注释明写「等服务端关」）。影响：服务端不回关时连接留在调用方手里，忘记 conn.Close() 会泄漏。
- D-005 拼写错误的导出符号（WSHanlder、cilent.go）长期保留。更正旧记录：9115d2d 重写 .golangci.yml 后 cilent.go 已无整文件豁免（现行配置无该排除项），且 5d88794 反而给它补了 godoc（Close 的关闭语义注释）——拼写保留纯为 import 兼容 (A：代码/配置现状 + AGENTS.md "don't fix it casually" 警示；C：不改原因推测)。改名即破坏兼容。

## 历史

- H-001 init（aa46371, 2022-08-25）时 handler 签名是 func([]byte)[]byte；次日 a998183 加 *websocket.Conn 参数，服务端可在应答时感知连接。当前影响：旧签名回调无法编译，属既成 breaking。
- H-002 模块路径 riverchu→tr1v3r（00887d8, 2023-08-15，A）。当前影响：旧 import 失效。
- H-003 b7685a2（2025-11-17）用 context 化控制修测试超时（A：提交信息）；e1deaff（2026-04-20）log 包重设计的 API 波及（log.Info→log.Infof，A：diff）。当前影响：日志调用以新 log 包 printf 变体为准。
- H-004 9115d2d（2026-08-24，#23）测试套件 hermetic 化重写：client_test/server_test 全部改为进程内 httptest 回环（Upgrade/echo/断开/关闭时序/拒绝非 WS 请求等），不再依赖外部环境（A：diff + 提交 "make suites hermetic"）。当前影响：T 系待办的验证成本降低；行为基线可回归。

## 待办

- T-001 Close() 不关底层连接（状态：候选未验证，2026-10-12，A：代码现状）。影响：对端不回关帧时依赖调用方补 conn.Close()。完成条件：维护者确认语义（补关或文档声明所有权）。
- T-002 无保活与 deadline（状态：候选未验证，2026-10-12，A：代码现状）。影响：空闲连接被中间设备静默掐断后，只能等下一次读写报错。完成条件：决定是否加 SetReadLimit/Ping 周期。
- T-003 Read 泵 chan 容量 64 硬编码（状态：候选未验证，2026-10-12，A：代码现状）。影响：消费慢于生产时泵协程阻塞在 chan 发送。完成条件：评估是否需要可配或满时丢策略。

## 验收案例

- S-001 服务端 echo 回路：起 gin + WSHanlder(回显 handler)，客户端连接后发 text 帧，收到同类型同内容应答。验证：消息内容与帧类型 mt 一致。
- S-002 客户端断开收敛：客户端突然断开 → 服务端 ReadMessage 报错 → handler 退出、连接关闭，进程与其余连接不受影响。验证：其余连接仍可 echo。
- S-003 Read 泵关闭信号：服务端关闭后，Read 返回的 chan 被 close，消费者 range 正常退出。验证：_, ok := <-ch 为 closed。
- S-004 跨域拒绝与放行：默认 WSHanlder 下带异源 Origin 的升级请求失败；换 WSHanlderWithUpgrader（CheckOrigin 恒真）后成功。验证：升级请求的 HTTP 状态对比。
- S-005 优雅关闭时序：Close() 发出关闭帧后约 1s 返回，期间服务端若回关则连接进入关闭流程；返回后底层 conn 仍需调用方关闭（结合 T-001 观察）。验证：抓包看关闭帧 + 返回耗时。
