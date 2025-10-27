# Makefile for Go project

.PHONY: help lint lint-fix test test-cover clean

# Default target
help:
	@echo "Available targets:"
	@echo "  lint       - Run golangci-lint"
	@echo "  lint-fix   - Run golangci-lint with auto-fix"
	@echo "  test       - Run all tests"
	@echo "  test-cover - Run tests with coverage"
	@echo "  clean      - Clean up generated files"

# Linting
lint:
	golangci-lint run

lint-fix:
	golangci-lint run --fix

# Testing
test:
	go test -v ./...

test-cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Clean up
clean:
	go clean
	rm -f coverage.out coverage.html