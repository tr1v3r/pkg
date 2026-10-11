# netool 知识库（合并版五件套）

组件：网络诊断工具——指定 DNS 服务器的递补查询（miekg/dns）+ 裸 ICMP 探测（平均时延）。
更新日期：2026-10-12（刷新 @ d6b2b9c）。证据等级：A=代码现状/可回查提交，C=推测。包地图见仓库 ARCHITECTURE.md（A 级来源）。

## 流程

```text
【LookupWithServer】(dns.go；9115d2d 起服务器串支持 host / host:port / [v6] / [v6]:port)
双循环: for server in servers { 归一化端口（裸主机补 :53，SplitHostPort 判别）
  for i < maxRetry { 组 A 记录问题 → dnsClient.Exchange(addr)
  交换错(err) → 记 lastErr, continue 重试
  响应但 Answer==nil → continue 重试（空响应视同可重试，56634b5 刻意加入）
  有 Answer → 收集 CNAME/A（Answer 段）与 SOA 的 NS（Ns 段）→ lastErr=nil, break 双循环 } }
返回累计结果 + lastErr；单查询超时 600ms（dnsClient 包级单例）；rcode 从不检查
LookupIP：直接走系统解析器 net.LookupIP

【ICMP 时延】(icmp.go)
ICMPDelay(ip, count): ctx 预算 1200ms×count → Dial("ip:icmp")（需 raw socket 特权）
  → 循环 count 次 singleICMP（seq 递增, 间隔 200ms）→ 返回平均时延
singleICMP: 组 echo 请求(Type=8, id=0, checksum) → Write → 计时起点 → Read 到任意包即停
  （不校验回包 seq/id/type；时延从写完才开始计）
SingleICMPDelay: 单次探测，预算 1200ms
Dial 失败（无特权/网络不可达）→ 返回错误，不再 panic（a525770 覆盖）
```

## 决策

- D-001 空响应可重试：Answer==nil 与交换错误同样 continue（A：56634b5, 2022-08-12，提交标题明写）。动机 (C)：推测遭遇过 DNS 瞬时空答。当前影响：见 T-001，NXDOMAIN 与瞬时空答不可区分。
- D-002 服务器地址形态开放、传输固定 UDP：9115d2d 起 LookupWithServer 接受 host:port/IPv6 字面量（裸主机仍默认 53）；UDP（dns.Client 零值）+ 600ms 单查超时不变、rcode 仍不检查（A：代码现状 + 提交 "accept host:port DNS servers"）。影响：TCP-only 服务仍不可查。
- D-003 ICMP 用已连接裸 socket + 「读到任何包即算回包」：不匹配 seq/id，也不区分 echo reply 与路过包（A：代码现状；(C) 推测为最小可用实现）。影响：多探测并发或噪声环境下时延可能失真。
- D-004 特权依赖显式化：raw socket 需 root/相应能力；测试在无特权环境 Skip 而非失败（A：icmp_test.go 现状，9115d2d hermetic 化）。

## 历史

- H-001 init code（f8e25d4, 2022-07-30）即含 dns+icmp；唯一功能演进是 56634b5（2022-08-12）空响应重试（A：提交记录）。
- H-002 2026-08~09 两波：9115d2d（#23）——host:port 支持 + ICMP 测试无特权 Skip + dns_server_test/dns_test（本地 mock DNS server，hermetic）；a525770——icmp dial 错误路径测试（A：提交信息）。当前影响：包仍处功能冻结态，但测试网从「裸奔」到可回归。

## 待办

- T-001 DNS 结果语义不可判别（状态：候选未验证，A：代码现状）。影响：域名不存在（NXDOMAIN）会烧完全部重试再落到下一服务器，既慢又误判。完成条件：检查 rcode 区分 NXDOMAIN/空答，或至少文档化。
- T-002 ICMP 回包不校验（状态：候选未验证，A：代码现状）。影响：单 host 多探测或本机多任务时测量失真。完成条件：回读后比对 id/seq。
- ~~T-003 无特权环境测试即红~~ ✅已关闭（9115d2d）：icmp_test.go 对 Dial 错误 t.Skipf("ICMP not permitted in this environment")，CI 普通用户绿。

## 验收案例

- S-001 正常解析：LookupWithServer("example.com", [公共 DNS], 3) 返回非空 A 记录、lastErr=nil。验证：记录数与格式。
- S-002 重试语义（已有测试锁定 dns_server_test.go）：指向返回空 Answer 的本地 mock 服务器 → 恰好发起 maxRetry 次查询后转下一服务器。验证：服务端计数 == maxRetry。
- S-003 全失败传播：servers 全部不可达 → 返回最后一家服务器的 lastErr，结果切片为空。验证：错误内容含最后地址。
- S-004 host:port 形态（9115d2d 新行为）：服务器串 "127.0.0.1:5353" 直连非 53 端口 mock → 解析成功。验证：mock 端口命中计数。
- S-005 ICMP 时延（无特权时 Skip）：SingleICMPDelay("127.0.0.1") 返回亚毫秒级正值；ICMPDelay(ip, 5) ≈ 单次均值且总时长 ≥ (5-1)×200ms。验证：量级断言而非精确值。
- S-006 ICMP dial 错误路径（已有测试锁定 icmp_internal_test.go）：不可达地址/无特权 Dial → 返回包装错误而非 panic/挂死。验证：错误断言 + 测试本身在无特权环境 Skip。
