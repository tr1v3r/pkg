# Makefile for Go project

.PHONY: help format lint lint-fix test test-cover clean lint-config lint-verbose lint-fast lint-ci security deps check-vet

# Default target
help:
	@echo "Available targets:"
	@echo "  format        - Format Go code using goimports-reviser"
	@echo "  lint          - Run golangci-lint with standard configuration"
	@echo "  lint-fix      - Run golangci-lint with auto-fix"
	@echo "  lint-fast     - Run golangci-lint fast mode (skips slow linters)"
	@echo "  lint-verbose  - Run golangci-lint with verbose output"
	@echo "  lint-config   - Show current golangci-lint configuration"
	@echo "  lint-ci       - Run golangci-lint in CI mode"
	@echo "  security      - Run gosec security scanner"
	@echo "  check-vet     - Run go vet static analysis"
	@echo "  test          - Run all tests"
	@echo "  test-cover    - Run tests with coverage"
	@echo "  deps          - Check and download dependencies"
	@echo "  clean         - Clean up generated files"

# Formatting
format:
	@echo "Formatting Go code..."
	goimports-reviser -output write .
	@echo "✅ Code formatted successfully"

# Linting
lint:
	@echo "Running golangci-lint..."
	golangci-lint run

lint-fix:
	@echo "Running golangci-lint with auto-fix..."
	golangci-lint run --fix

lint-fast:
	@echo "Running golangci-lint in fast mode..."
	golangci-lint run --max-issues-per-linter=10 --max-same-issues=5

lint-verbose:
	@echo "Running golangci-lint with verbose output..."
	golangci-lint run -v

lint-config:
	@echo "Current golangci-lint configuration:"
	golangci-lint config path

lint-ci:
	@echo "Running golangci-lint in CI mode..."
	golangci-lint run --timeout=10m --out-format=github-actions

security:
	@echo "Running security scan with gosec..."
	gosec ./...

check-vet:
	@echo "Running go vet static analysis..."
	go vet ./...

deps:
	@echo "Checking and downloading dependencies..."
	go mod download
	go mod tidy

# Testing
test:
	@echo "Running all tests..."
	go test -v ./...

test-fast:
	@echo "Running tests without race detection (faster)..."
	go test -short ./...

test-race:
	@echo "Running tests with race detection..."
	go test -race -short ./...

test-cover:
	@echo "Running tests with coverage..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report generated: coverage.html"

test-integration:
	@echo "Running integration tests..."
	go test -v -tags=integration ./...

test-package:
	@if [ -z "$(PKG)" ]; then \
		echo "Usage: make test-package PKG=./pkgname"; \
		exit 1; \
	fi
	@echo "Testing package $(PKG)..."
	go test -v $(PKG)

# Check everything (common CI pipeline)
check-all: format lint security deps test-race
	@echo "✅ All checks completed successfully"

# Quick check for development
check-quick: lint-fast test-fast
	@echo "✅ Quick checks completed"

# Clean up
clean:
	@echo "Cleaning up generated files..."
	go clean
	rm -f coverage.out coverage.html
	@echo "✅ Clean up completed"

# Install development tools
install-tools:
	@echo "Installing development tools..."
	go install github.com/incu6us/goimports-reviser@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	@echo "✅ Development tools installed"