# AGENTS.md

Guidance for AI coding agents (Claude Code, etc.) working in this repository.
`CLAUDE.md` is a symlink to this file.

## Project Overview

Go utility collection, Go 1.24+. Each of the 17 top-level library directories is an independent module (`github.com/tr1v3r/pkg/<pkg>`). The root module is an empty migration boundary; `go.work` joins all modules for development. See `MODULES.md` for dependencies, standalone checks and first-release migration.

CI test coverage is explicit: Go 1.24.x, 1.25.x, 1.26.x and 1.27.x for every module, module tooling, cache stress and release validation. Keep these matrices aligned.

## Component Knowledge Base

Complex packages carry a `knowledge/` directory beside their code with the
intent, active constraints, failure history, and acceptance scenarios that the
source alone cannot express. Start from the root `knowledge/INDEX.md` map.
Before changing a package's control flow, contracts, or error semantics, read
its FLOW/DECISIONS; after changes, verify against its SCENARIOS and update
only the affected parts. TODO entries are background, not an authorization
list.

## Common Commands

```bash
make test              # Run go test -v ./... in every library module
make test-fast         # Quick tests (-short, no race detection)
make test-race         # Tests with -race -short
make test-cover        # Coverage reports inside each module
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
make check-all         # Full CI check: format + lint + security + tidy check + test-race
make check-quick       # Quick dev check: lint-fast + test-fast
make MODULE=cache test-race # Select one module for most targets
make check-modules     # GOWORK=off tidy check using temporary checkout artifacts
make test-standalone   # GOWORK=off race/integration tests using checkout artifacts
make test-published    # GOWORK=off tests against actual published pins
make deps-published    # Tidy actual releases and record real internal checksums
python3 scripts/test-tooling.py # Module tooling and migration regression checks
python3 scripts/test-release.py # Automatic version selection and release gates
```

Run a single test: `MODULE=cache ./scripts/modules.sh go test -v -run TestName ./...`. Root `go test ./...` does not test nested modules. `scripts/modules.sh --list` drives CI module discovery.

## Package Structure

### Core Packages
- **log/** — Structured logging: `Setup()`-based multi-sink (Console, File, RotateFile with daily/size rotation), async buffered writes, structured key-value + printf styles, Ctx* variants with request-scoped logID, 7 levels (Trace–Panic). Bidirectional `log/slog` compatibility lives here (`log/slog.go`) — there is NO separate `slog/` package.
- **fetch/** — Resilient HTTP client: retry with exponential backoff, middleware chain, functional options, typed errors, TLS 1.2+ enforcement.
- **circuitbreaker/** — Standalone circuit breaker pattern (closed/open/half-open states, `cb.Execute(func() error)`). Used alongside `fetch`, not inside it.
- **notion/** — Notion API client, interface-driven: `NewClient(version, token)` exposes `Database`, `Page`, `Block`, `User`, `Search`, `Comment` APIs. Built-in rate limiting, automatic pagination, `QueryIter` lazy iterator.

### Utility Packages
- **cache/** — Generic, standard-library-only local TTL cache: `New[K, V]` (+ `Option[K, V]` applied to the cache), `Set` (positive TTL only), `Get`, `Delete`, `Close`. Strict expiry on reads; sync.Map snapshots, indexed min-heap and deadline-driven cleanup. Cleanup removes overdue entries in batches bounded by count and wall time (defaults `DefaultCleanupBatchSize`=128 / `DefaultCleanupBatchBudget`=1ms; `WithCleanupBatchSize`/`WithCleanupBatchBudget`, nonpositive disables), releasing the heap lock between passes. Set publishes a new node; only indexes mutate under the heap lock. Always `Close` to stop the worker; zero value is unsupported.
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
- Tests additionally depend on `fetch -> circuitbreaker` and `config -> guard`.
- Everything else is standalone; keep it that way unless there's a strong reason.

## Code Quality

### Linting (.golangci.yml, golangci-lint v2, pinned v2.12.2 in CI)
- Max line length: 120 (`lll`; test files relaxed — fixtures embed long XML/JSON/ICS literals)
- Enabled: bodyclose, dogsled, dupl (threshold 120), errcheck, gocritic, gocyclo (min 20), goconst, gosec, govet (all analyzers except fieldalignment/inline), ineffassign, lll, misspell, nakedret, prealloc, staticcheck, unconvert, unused
- Formatting is enforced by the config itself: `gofmt -s` + `goimports` (local-prefix `github.com/tr1v3r/pkg`) — `golangci-lint fmt` auto-fixes, so `make format` and CI stay consistent
- gosec excludes G115, G204, G104, G304 (SDK flexibility)
- Generated code, vendor/, testdata/, examples/ are excluded
- `_test.go` files are excluded from gocyclo, dupl, goconst, gosec, errcheck, prealloc, ineffassign, dogsled, nakedret, gocritic, bodyclose, lll
- govet printf checker knows the `log.Logger` printf-style methods (`Infof`, `Warnf`, ...)
- Opinionated gocritic checks (hugeParam, rangeValCopy, docStub, ...) are disabled — they fight the value-semantics design; `CANCELLED`/`Cancelled` spellings are intentional (RFC 5545 literal + exported API)

### Patterns
- **Functional options**: config (log, calendar, fetch)
- **Context propagation**: `context.Context` as first argument through package boundaries
- **Interface-driven design**: e.g. notion's `DatabaseAPI`/`PageAPI`/... enable mocking (`mock_test.go`)
- **Manager/client pattern**: one client type exposing domain-specific API groups (notion)
- **Table-driven tests** with `testify`

## Gotchas

- **Formatting is `goimports-reviser`**, not `gofmt`/`goimports`. Install via `make install-tools`. `make format` only touches git-changed files.
- **`log` vs stdlib `log/slog`**: this repo's `log/` is the primary logger and already provides slog compatibility internally; do not create a `slog/` wrapper package.
- **Multiple modules**: each library has its own go.mod; root go.mod contains no library packages or dependencies. Internal pins target planned first v0.1.0 releases; do not claim published compatibility based on checkout-proxy tests. Keep local replacements in go.work only.
- **Automatic releases**: after a same-repository dev -> master PR passes push CI, publish changed Go/manifests with independent patch increments. Four Go toolchains validate real dependency pins before any new tag. Keep the source gate, serialized queue, bootstrap order and retry idempotency intact. Dry-run with `python3 scripts/auto-release.py`.
- **Release tags**: directory-prefixed tags such as cache/v0.1.0. First publish the root split boundary, then dependencies before dependents, including test dependencies. Never create tags/releases without explicit user authorization.
- **Build tags**: tooling references `integration` and `demo` tags (`make test-integration`, lint `build-tags`); most tests are plain unit tests using `httptest`.
- **Worktree checkout**: `.git` here is a file pointing into the parent repo's modules dir; normal `git` commands work fine.
- **Commit style**: Conventional Commits (`feat(scope): ...`, `fix: ...`, `docs: ...`) — see `git log` for examples.
