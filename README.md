# pkg

A collection of utility Go packages providing commonly needed functionality for Go applications.

## Packages

### [log](./log) - Structured Logging System
Advanced structured logging with multiple output handlers, async processing, and context-aware logging. Compatible with Go's `slog` package for JSON/text output.

**Features:**
- Multiple log levels (Trace, Debug, Info, Warn, Error, Fatal, Panic)
- Context-aware logging with log IDs
- Async processing with configurable buffer sizes
- ConsoleHandler (async buffered output) and FileHandler (with log rotation)
- Non-blocking log output with guaranteed message delivery

### [fetch](./fetch) - HTTP Client with Resilience
Robust HTTP client with circuit breaker, retry mechanisms, and middleware support.

**Features:**
- Circuit breaker with automatic failure detection and recovery
- Configurable retry logic with exponential backoff
- Extensible middleware architecture
- Thread-safe client operations
- TLS 1.2+ security enforcement
- Context cancellation support

### [hash](./hash) - Cryptographic Hash Functions
Simple cryptographic hash calculation utilities with multiple algorithm support.

**Features:**
- Multiple algorithms: MD5, SHA1, SHA224, SHA256, SHA384, SHA512
- SHA3 variants: SHA3-224, SHA3-256, SHA3-384, SHA3-512
- Simple API with hex-encoded string results

### [netool](./netool) - Network Utilities
DNS and ICMP network diagnostic tools for connectivity testing.

**Features:**
- DNS resolution with multiple server support
- ICMP ping for network connectivity testing
- Configurable retry attempts for DNS queries
- Support for A, CNAME, and NS record types

### [notion](./notion) - Notion API Client
Comprehensive Notion API integration with rate limiting and structured types.

**Features:**
- Full API coverage: Databases, Pages, Blocks, and Search operations
- Built-in rate limiting with `x/time/rate`
- Strongly typed API responses
- Organized manager classes for different resource types
- Context cancellation support

### [pools](./pools) - Goroutine Pool Management
Simple goroutine pool for concurrent task management using token-based patterns.

**Features:**
- Token-based Wait/Done pattern for resource management
- Both synchronous and asynchronous wait patterns
- Configurable pool size limits
- Resource tracking for monitoring active resources

### [thread](./thread) - Advanced Thread Pool
Sophisticated thread pool with timeout support, job management, and graceful shutdown.

**Features:**
- Configurable job execution timeouts
- Buffered job queue with configurable capacity
- Dynamic worker allocation
- Graceful shutdown with job completion
- Job tracking and pool status monitoring

### [websocket](./websocket) - WebSocket Server
WebSocket server integration with Gin framework for bidirectional communication.

**Features:**
- Seamless integration with Gin web framework
- Bidirectional message processing
- Automatic connection upgrade and cleanup
- Robust error handling with logging

### [config](./config) - Configuration Management
Flexible configuration loading from files and URLs with environment support.

**Features:**
- Multiple sources: file system and HTTP/HTTPS URLs
- Extensible parser interface
- Environment variable configuration
- Built-in JSON parsing support

### [calendar](./calendar) - iCalendar Generation
Generate iCalendar (.ics) files programmatically with RFC 5545 compliance.

**Features:**
- Standards-compliant iCalendar generation
- Event creation and management
- Time zone-aware event scheduling
- Flexible options for calendar properties

### [brute](./brute) - Search Algorithm Framework
Generic BFS/DFS search algorithm implementation with type-safe state management.

**Features:**
- Both BFS and DFS search methods
- Type-safe generic interface
- Path tracking with backtracking and cost calculation
- Automatic cycle detection

### [sort](./sort) - Sorting Utilities
Additional sorting algorithms and utilities beyond the standard library.

**Features:**
- Extended sorting methods
- Performance-optimized algorithm implementations

### [shutdown](./shutdown) - Shutdown Signal Handler
Listen for shutdown signals and handle graceful application termination.

**Features:**
- OS signal handling for graceful shutdown
- Cleanup hook registration
- Context-based timeout management

## Installation

```bash
go get github.com/tr1v3r/pkg
```

## Usage

Import individual packages as needed:

```go
import (
    "github.com/tr1v3r/pkg/fetch"
    "github.com/tr1v3r/pkg/log"
    "github.com/tr1v3r/pkg/notion"
)
```

## Recent Updates

The logging package has been completely rewritten with:
- Structured logging compatibility with Go's `slog` package
- Improved async processing with guaranteed message delivery
- Comprehensive test coverage improvements
- Better error handling and resource management
