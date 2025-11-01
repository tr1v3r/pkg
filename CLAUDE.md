# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a collection of utility Go packages (`github.com/tr1v3r/pkg`) providing commonly needed functionality for Go applications. The repository contains independent, modular packages that can be imported individually.

## Development Commands

### Testing
- Run all tests: `make test` or `go test -v ./...`
- Run tests with coverage: `make test-cover` or `go test -coverprofile=coverage.out ./...`
- View coverage report: `go tool cover -html=coverage.out -o coverage.html`

### Linting
- Run golangci-lint: `make lint` or `golangci-lint run`
- Auto-fix lint issues: `make lint-fix` or `golangci-lint run --fix`

### Package-specific Testing
- Test specific package: `go test -v ./fetch`
- Test with coverage for specific package: `go test ./fetch -coverprofile=coverage.out`

## Architecture and Package Structure

### Core Packages

#### Log (`log/`)
- **Purpose**: Structured logging system compatible with Go's `slog` package
- **Key Components**:
  - `Logger`: Main logging interface with async processing
  - `ConsoleHandler`: Async buffered console output
  - `FileHandler`: File output with log rotation
  - Multiple log levels (Trace, Debug, Info, Warn, Error, Fatal, Panic)
- **Architecture**: Handler-based design with async message processing and guaranteed delivery

#### Fetch (`fetch/`)
- **Purpose**: Resilient HTTP client with circuit breaker and retry mechanisms
- **Key Components**:
  - Circuit breaker with automatic failure detection
  - Configurable retry logic with exponential backoff
  - Extensible middleware architecture
  - Thread-safe operations with TLS 1.2+ security
- **Architecture**: Client wrapper with pluggable middleware and resilience patterns

#### Notion (`notion/`)
- **Purpose**: Comprehensive Notion API client with rate limiting
- **Key Components**:
  - `Manager`: Central coordinator with sub-managers for different resource types
  - `DatabaseManager`, `PageManager`, `BlockManager`, `SearchManager`
  - Built-in rate limiting using `golang.org/x/time/rate`
  - Strongly typed API responses
- **Architecture**: Manager pattern with resource-specific sub-managers

### Utility Packages

- **Hash (`hash/`)**: Cryptographic hash functions (MD5, SHA family, SHA3 variants)
- **Netool (`netool/`)**: Network utilities (DNS resolution, ICMP ping)
- **Pools (`pools/`)**: Goroutine pool management with token-based patterns
- **Thread (`thread/`)**: Advanced thread pool with timeout support and graceful shutdown
- **WebSocket (`websocket/`)**: WebSocket server integration with Gin framework
- **Config (`config/`)**: Configuration management from files and URLs
- **Calendar (`calendar/`)**: iCalendar generation with RFC 5545 compliance
- **Brute (`brute/`)**: Generic BFS/DFS search algorithm framework

## Code Quality and Standards

### Linting Configuration
- Uses comprehensive golangci-lint configuration (`.golangci.yml`)
- Line length: 120 characters
- Enabled linters: gocritic, govet, gosec, staticcheck, and others
- Disabled linters: wsl, funlen, nestif (considered too opinionated)
- Test files have relaxed linting rules

### Testing Patterns
- Each package contains comprehensive unit tests
- Integration tests use `integration` build tag
- Test coverage is tracked and reported
- Mocking and test utilities are provided where needed

### Package Dependencies
- Packages are designed to be independent and modular
- Internal package dependencies are managed via Go modules
- External dependencies are minimized and carefully selected

## Development Guidelines

### Adding New Packages
1. Create new directory under root
2. Implement package with clear, focused responsibility
3. Add comprehensive tests
4. Update main README.md with package description
5. Ensure compatibility with existing linting and testing standards

### Package Structure
- Each package should have a clear single responsibility
- Export only necessary types and functions
- Use internal packages for implementation details when needed
- Follow Go naming conventions and idioms

### Error Handling
- Use Go's standard error handling patterns
- Provide meaningful error messages
- Consider context cancellation in long-running operations
- Log errors appropriately using the logging package

## Integration Patterns

### Using Multiple Packages
Packages are designed to work together seamlessly:
- `notion` package uses `fetch` for HTTP requests
- All packages can use `log` for structured logging
- `config` can be used for configuration across packages

### Testing Integration
- Use `go test -tags=integration` for integration tests
- Mock external dependencies when testing
- Test packages in isolation and in combination