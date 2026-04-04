# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Go utility package collection (`github.com/tr1v3r/pkg`). Go 1.24+. Independent, modular packages imported individually.

## Quick Reference

```bash
make test              # Run all tests
make test-cover        # Tests with coverage → coverage.html
make test-fast         # Quick tests (no race detection)
make test-race         # Tests with race detection
make test-integration  # Integration tests (-tags=integration)
make format            # Format with goimports-reviser
make lint              # golangci-lint
make lint-fix          # Auto-fix lint issues
make install-tools     # Install dev tools (goimports-reviser, golangci-lint, gosec)
make check-all         # Full CI check: format + lint + security + deps + test-race
make check-quick       # Quick dev check: lint-fast + test-fast
```

### Package-specific Testing
```bash
go test -v ./fetch                                     # Test one package
go test -v -run TestSpecificFunction ./packagename     # Run specific test
go test -tags=integration ./...                        # Integration tests
```

## Package Structure

### Core Packages
- **log/** — Structured logging with async processing, handler-based (ConsoleHandler, FileHandler), 7 log levels
- **slog/** — `log/slog` interface compatibility wrapper for the `log` package. Use `slog.AsSlogLogger()` to adapt existing loggers
- **fetch/** — Resilient HTTP client: circuit breaker, retry with exponential backoff, middleware, TLS 1.2+
- **notion/** — Notion API client with rate limiting. Manager pattern: `DatabaseManager`, `PageManager`, `BlockManager`, `SearchManager`

### Utility Packages
- **hash/** — Crypto hashes (MD5, SHA family, SHA3). MD5/SHA1 have deprecation notices for security contexts
- **netool/** — DNS resolution and ICMP ping
- **pools/** — Goroutine pool with token-based Wait/Done patterns
- **thread/** — Thread pool with timeout support, job queue, graceful shutdown
- **websocket/** — WebSocket server with Gin framework
- **config/** — Config loading from files and URLs (JSON built-in)
- **calendar/** — iCalendar (.ics) generation, RFC 5545 compliant
- **brute/** — Generic BFS/DFS search framework with type-safe state management
- **alfred/** — Alfred workflow JSON output for macOS automation
- **guard/** — Graceful shutdown signal handler with cleanup hooks and panic stack traces
- **rss/** — RSS/Atom feed parsing
- **sort/** — Extended sorting utilities

### Cross-Package Dependencies
- `notion` uses `fetch` for HTTP requests
- `slog` wraps `log` package handlers for `log/slog` compatibility
- All packages can use `log` for structured logging

## Code Quality

### Linting (.golangci.yml)
- Line length: 120 chars
- Key linters: gocritic, govet, gosec, staticcheck, errcheck, goconst, bodyclose
- Test files have relaxed rules (excluded from gocyclo, dupl, gosec, etc.)
- Skipped files: `notion/databases.go`, `websocket/cilent.go`

### Patterns
- **Option pattern**: Functional options for config (log, calendar, fetch)
- **Context propagation**: Context through package boundaries
- **Manager pattern**: Resource-specific managers for complex APIs (notion)
- **Interface-driven design**: Clear interfaces for extensibility

## Gotchas

- **`log` vs `slog`**: Two separate packages. `log/` is the primary logger. `slog/` provides `log/slog` stdlib compatibility via wrapping — it is NOT a standalone logger
- **Formatting uses `goimports-reviser`** (not `gofmt` or `goimports`). Install via `make install-tools`
- **Build tags**: Integration tests use `-tags=integration`; demos use `-tags=demo`
- **Module structure**: `log/` is also published as `github.com/tr1v3r/pkg/log` (separate go.mod entry)
