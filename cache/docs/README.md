# cache 架构与生命周期

图表基于 `1d8386417741681b9305ab6b5448c8bc57495069` 的实现；每个关键节点
附有仓库相对路径和源码行号。图中涉及的状态是阅读模型，不是新增的导出 API 或枚举。

| 图表 | 内容 | 可维护源文件 |
| --- | --- | --- |
| [架构图](architecture.html) | 并发 API、map 与索引最小堆、合并唤醒、deadline 驱动的清理 worker | [architecture.json](architecture.json) |
| [实例生命周期](lifecycle.html) | New、开放访问、Close 清空、stop / done 同步与不可逆关闭 | [lifecycle.json](lifecycle.json) |
| [条目生命周期](entry-lifecycle.html) | 存活、逻辑过期、物理移除、到期后重写与主动删除 | [entry-lifecycle.json](entry-lifecycle.json) |

## 阅读

下载 HTML 后直接用浏览器打开即可，不需要启动服务或安装前端依赖。文件自包含，
默认静态展示，可切换明暗主题并查看节点说明。GitHub 文件页只显示 HTML 源码，
不会直接渲染交互图。

源码定位采用 `local-only` 模式：保留仓库 SSH origin、版本、相对路径与行号，
不把开发机器的绝对路径写入图表，也不把 SSH origin 改写成网页链接。

重点区分两个边界：

```text
【条目可见性】
Set → deadline = now + TTL
  ├─ now < deadline  → Get 命中；读取不续期
  └─ now >= deadline → Get 未命中
                       └─ worker 持堆锁，逐项短锁 map 后物理删除

【实例所有权】
New → 一个 worker → nextDelay → 等 wake / timer.C / stop
Close
  ├─ heapMu → mapMu：closed = true → 清空 map / heap → close(stop)
  └─ 释放两把锁 → 等 done → Close 返回时 worker 已退出
```

Set / Delete 仅在最早 deadline 改变时通知 worker，包括空堆转换；Get 不需要堆锁。
实测取舍与复现方法见 [集中过期测量](performance.md)。

## 更新与重新生成

JSON 是图表源文件，HTML 是提交到仓库的可阅读产物。源码行为变化时，先核对
`cache/cache.go`、`cache/heap.go` 及相关测试，再同步修改图中的关系、说明和
`sources` 行号。将各 JSON 的 `meta.repository.revision` 更新到实现所在的完整
commit SHA；证据校验使用该 commit 的内容，而不是未提交的工作区修改。

生成工具为 Archify（本次使用 3.0.1），浏览器检查需要本机 Chrome / Chromium。
不需要向本 Go 模块添加依赖。安装好工具后，在**仓库根目录**执行：

```sh
# 替换为本机 Archify CLI 的实际路径。
ARCHIFY_CLI=/path/to/archify/bin/archify.mjs

node "$ARCHIFY_CLI" finalize architecture cache/docs/architecture.json \
  cache/docs/architecture.html --repo-root . --quality showcase --json
node "$ARCHIFY_CLI" finalize lifecycle cache/docs/lifecycle.json \
  cache/docs/lifecycle.html --repo-root . --quality showcase --json
node "$ARCHIFY_CLI" finalize lifecycle cache/docs/entry-lifecycle.json \
  cache/docs/entry-lifecycle.html --repo-root . --quality showcase --json
```

三个命令应串行运行；Archify 对同一输出目录的交付操作加锁。
`finalize` 依次执行规格与源码证据校验、HTML 交付、严格产物检查、真实浏览器检查。
必须检查退出码，全部通过才算生成成功。已有浏览器证据后重新编排图表时，可为该次
命令增加 `--out-dir cache/docs/<图名>.review-2`，保留旧证据。

校验回执、截图等本机证据由此目录的 `.gitignore` 排除，不作为可移植文档提交；
它们可能含绝对路径。保留 JSON 与 HTML 的配对更新，不手工修改生成的 HTML。
