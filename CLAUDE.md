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
- Integration tests: `go test -tags=integration ./...`
- Run specific test: `go test -v -run TestSpecificFunction ./packagename`

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
- **Alfred (`alfred/`)**: Alfred workflow JSON output generator for macOS automation
- **Guard (`guard/`)**: Graceful shutdown signal handler with cleanup hooks
- **RSS (`rss/`)**: RSS/Atom feed data structures and parsing utilities
- **Sort (`sort/`)**: Extended sorting utilities beyond standard library

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

### Security Guidelines
- **TLS enforcement**: HTTP clients require TLS 1.2+ minimum
- **Hash deprecation**: MD5/SHA1 include deprecation notices for security contexts
- **Gosec compliance**: Specific exclusions documented for legitimate use cases
- **Input validation**: All external data sources validated before processing

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

### Development Patterns
- **Option pattern**: Use functional options for configuration (seen in log, calendar, fetch)
- **Context awareness**: Propagate context through package boundaries
- **Manager pattern**: Resource-specific managers for complex APIs (notion package)
- **Graceful degradation**: Handle failures gracefully with circuit breakers and retries
- **Interface-driven design**: Define clear interfaces for extensibility

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

### Module Structure
- **Root module**: `github.com/tr1v3r/pkg` contains all packages
- **Demo modules**: `log/demo/` contains practical usage examples
- **Local imports**: Packages reference each other via local module paths
- **Independent usage**: Each package can be imported individually

### Testing Integration
- Use `go test -tags=integration` for integration tests
- Mock external dependencies when testing
- Test packages in isolation and in combination
- **Testdata**: External test data in `testdata/` directories where needed

## Advanced Architecture Patterns

### Resilience and Reliability
- **Circuit breaker**: Automatic failure detection and recovery in fetch package
- **Rate limiting**: Built-in rate limiting using `golang.org/x/time/rate` in notion package
- **Retry mechanisms**: Configurable retry logic with exponential backoff
- **Graceful shutdown**: Signal handling with cleanup hooks in guard package

### Performance Optimizations
- **Async processing**: Non-blocking operations in log package handlers
- **Pool management**: Goroutine and thread pools for concurrent operations
- **Buffered operations**: Configurable buffer sizes for I/O operations
- **Resource tracking**: Monitor active resources and pool utilization