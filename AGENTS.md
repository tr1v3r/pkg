# AGENTS.md

Guidance for AI coding agents (Claude Code, etc.) working in this repository.
`CLAUDE.md` is a symlink to this file.

## Project Overview

Go utility package collection (`github.com/tr1v3r/pkg`), Go 1.24+. Single module at the repo root; packages are independent and imported individually (`go get github.com/tr1v3r/pkg` + `import "github.com/tr1v3r/pkg/<pkg>"`).

## Common Commands

```bash
make test              # Run all tests (go test -v ./...)
make test-fast         # Quick tests (-short, no race detection)
make test-race         # Tests with -race -short
make test-cover        # Tests with coverage → coverage.html
make test-integration  # Integration tests (-tags=integration)
make test-package PKG=./fetch   # Test one package
make format            # Format changed Go files (goimports-reviser -rm-unused)
make format-all        # Format all Go files
make lint              # golangci-lint run
make lint-fast         # golangci-lint with relaxed issue caps
make lint-fix          # Auto-fix lint issues
make security          # gosec security scan
make check-vet         # go vet
make install-tools     # Install goimports-reviser, golangci-lint, gosec
make check-all         # Full CI check: format + lint + security + deps + test-race
make check-quick       # Quick dev check: lint-fast + test-fast
```

Run a single test: `go test -v -run TestName ./packagename`.

## Package Structure

### Core Packages
- **log/** — Structured logging: `Setup()`-based multi-sink (Console, File, RotateFile with daily/size rotation), async buffered writes, structured key-value + printf styles, Ctx* variants with request-scoped logID, 7 levels (Trace–Panic). Bidirectional `log/slog` compatibility lives here (`log/slog.go`) — there is NO separate `slog/` package.
- **fetch/** — Resilient HTTP client: retry with exponential backoff, middleware chain, functional options, typed errors, TLS 1.2+ enforcement.
- **circuitbreaker/** — Standalone circuit breaker pattern (closed/open/half-open states, `cb.Execute(func() error)`). Used alongside `fetch`, not inside it.
- **notion/** — Notion API client, interface-driven: `NewClient(version, token)` exposes `Database`, `Page`, `Block`, `User`, `Search`, `Comment` APIs. Built-in rate limiting, automatic pagination, `QueryIter` lazy iterator.

### Utility Packages
- **hash/** — Crypto hashes (MD5, SHA family, SHA3) with hex-string results; MD5/SHA1 carry deprecation notices for security contexts.
- **netool/** — DNS resolution (multiple servers, A/CNAME/NS, retries) and ICMP ping.
- **pools/** — Goroutine pool with token-based Wait/Done patterns.
- **thread/** — Thread pool with per-job timeouts, buffered job queue, graceful shutdown.
- **websocket/** — WebSocket server (Gin integration) and client. Note: `cilent.go` is a real (mis-spelled) filename kept for history — don't "fix" it casually, it would break imports.
- **config/** — Config loading from files and URLs (HTTP/HTTPS via `fetch`), extensible parser interface, env-var support, built-in JSON.
- **calendar/** — iCalendar (.ics) generation AND parsing, RFC 5545 compliant.
- **brute/** — Generic BFS/DFS search framework with type-safe state management, path backtracking, cycle detection.
- **alfred/** — Alfred workflow JSON output for macOS automation.
- **guard/** — Shutdown signal handling with cleanup hooks and panic stack traces.
- **rss/** — Parsing for RSS, Atom, JSON Feed, and OPML; format conversion, item deduplication.
- **sort/** — Extended sorting utilities (`sort.By[T]` function-type pattern).

### Cross-Package Dependencies (actual)
- `notion` imports `log`
- `websocket` imports `log`
- `config` imports `fetch`
- Everything else is standalone; keep it that way unless there's a strong reason.

## Code Quality

### Linting (.golangci.yml, golangci-lint v2)
- Max line length: 120 (`lll`)
- Enabled: errcheck, govet (enable-all analyzers), staticcheck, unused, ineffassign, gocyclo (min 20), dupl (threshold 120), goconst, gocritic, gosec, misspell, prealloc, bodyclose, dogsled, nakedret, unconvert
- gosec excludes G115, G204, G104, G304 (SDK flexibility)
- `_test.go` files are excluded from gocyclo, dupl, goconst, gosec, errcheck, prealloc, ineffassign, dogsled, nakedret, gocritic
- govet printf checker knows the `log.Logger` printf-style methods (`Infof`, `Warnf`, ...)
- goimports local-prefixes: `github.com/tr1v3r/pkg`

### Patterns
- **Functional options**: config (log, calendar, fetch)
- **Context propagation**: `context.Context` as first argument through package boundaries
- **Interface-driven design**: e.g. notion's `DatabaseAPI`/`PageAPI`/... enable mocking (`mock_test.go`)
- **Manager/client pattern**: one client type exposing domain-specific API groups (notion)
- **Table-driven tests** with `testify`

## Gotchas

- **Formatting is `goimports-reviser`**, not `gofmt`/`goimports`. Install via `make install-tools`. `make format` only touches git-changed files.
- **`log` vs stdlib `log/slog`**: this repo's `log/` is the primary logger and already provides slog compatibility internally; do not create a `slog/` wrapper package.
- **Single go.mod** at the repo root — `log` is NOT a separate module (it was in the past).
- **Build tags**: tooling references `integration` and `demo` tags (`make test-integration`, lint `build-tags`); most tests are plain unit tests using `httptest`.
- **Worktree checkout**: `.git` here is a file pointing into the parent repo's modules dir; normal `git` commands work fine.
- **Commit style**: Conventional Commits (`feat(scope): ...`, `fix: ...`, `docs: ...`) — see `git log` for examples.
