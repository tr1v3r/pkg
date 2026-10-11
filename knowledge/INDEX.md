# Component Knowledge Index

Updated 2026-10-12, baseline master @ d6b2b9c. Packages in this collection are
fully independent; each complex one carries its own `knowledge/` beside the
code. Full sets = FLOW / DECISIONS / HISTORY / TODO / SCENARIOS; compact sets
= a single BRIEF.md. Motive-level entries are graded (C) inside each set and
await maintainer confirmation. Repo-level evidence sources: ARCHITECTURE.md
(4cb7e6a) and the 100% godoc coverage (5d88794).

## Full knowledge sets

| Package | Scope | Read first |
| --- | --- | --- |
| `notion` | Notion API client: centralized rate-limited client + six domain interfaces + Query/QueryIter dual pagination forms (b1f8695); 429 maps to the ErrRateLimited sentinel without auto-retry; GET pagination params travel as query, POST as body (e778688) | DECISIONS |
| `fetch` | Resilient HTTP wrapper: shared package-level client (TLS≥1.2), ctx-aware retry (Retry-After honored first without a cap; bodies fully buffered to survive retries), middleware chain without request visibility (the underlying request keeps running after a timeout); context options apply first (d3250fb); circuit breaking moved to the sibling `circuitbreaker` package (ca009f2) | DECISIONS |
| `calendar` | iCalendar generation + lenient parsing (round-trip is the acceptance yardstick): Date renders in its own zone with UTC-designator layouts pre-converted (b1d70ff/ad4ce80); TEXT escaping + 75-octet folding both ways; TZID supported in both directions; orphan END lines skipped; RRULE/EXDATE passed through, never expanded | FLOW |
| `log` | Structured logging: Logger dispatches Records to multiple sinks (level filter + encoder + target); Console synchronous, file sinks async (buf=1024) + nil-sentinel barrier for Sync/Close; lazy time/size rotation; backpressure without dropping on a full channel; two-way slog compatibility; child Loggers from `With` own their field storage (d6b2b9c); Fatal syncs before exit (9115d2d) | DECISIONS |
| `rss` | RSS/Atom/JSON Feed/OPML parsing and conversion: DetectFeedType sniffing → Parse returns any; ParseTyped generic entry (b1d70ff, full protocol schema coverage incl. the `Item.GUID→*GUID` source-breaking change); lenient philosophy — zero values for missing fields, bad dates passed through, invalid OPML is a silent no-op; DeduplicateItems dedupes within a single channel only; no fetch/cache/refresh state | FLOW |

## Compact knowledge sets (BRIEF.md)

| Package | One-liner |
| --- | --- |
| `circuitbreaker` | Three-state breaker: single-Mutex concurrency semantics, HalfOpen does not cap probes, late results dropped; extracted from fetch (ca009f2) |
| `pools` | Channel token pool (a semaphore, not a task pool): wg.Add before the blocking acquire, negative size = unlimited, nil-safe |
| `websocket` | gorilla/websocket thin wrapper: serial server echo loop, default upgrader is same-origin, Close sends the frame without closing the connection |
| `hash` | 11 one-shot hash functions (hex output): MD5/SHA1 kept for legacy, SHA3 still via x/crypto pending stdlib migration |
| `thread` | TimeoutPool worker pool: per-job Job.Timeout deadline (abandons the goroutine, frees the worker, #23); graceful shutdown drops queued jobs; StartAndWait timeout is still gap semantics (D-04 unchanged) |
| `sort` | Generic single/multi-key sorting for []*T: unstable, lexicographic short-circuit; the only breaking change was []T→[]*T (f59633f) |
| `netool` | DNS against an explicit server (host:port / IPv6, #23) + raw ICMP latency: empty responses retried, rcode never checked, ICMP replies unverified; tests skip in unprivileged environments |

## No separate knowledge base (by design)

`brute` (self-explanatory BFS/DFS framework), `config` (thin file/URL loader
with pluggable parsers), `alfred` (Alfred Script Filter JSON struct mirror;
Reset fix in 9115d2d), `guard` (51 lines — but note: importing it takes over
SIGINT/SIGTERM; `InspectShutSignal` derives its replacement watcher from a
fresh Background since 2f00c53, with a regression test). Upgrade a package to
its own set only when its logic or history grows — a rotten knowledge base is
worse than none.
