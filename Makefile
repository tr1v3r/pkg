# Commands run in every library module; set MODULE=cache to select one.
MODULES := ./scripts/modules.sh
LINT ?= golangci-lint
LINT_CONFIG := $(CURDIR)/.golangci.yml

.PHONY: help modules format format-all lint lint-fix lint-fast lint-verbose lint-config lint-ci \
        security check-vet deps deps-published check-modules test test-fast test-race test-cover test-integration \
        test-package test-standalone test-published check-all check-quick clean install-tools

help:
	@echo "Use MODULE=cache to select one module. Available targets:"
	@echo "  modules          List library modules"
	@echo "  format[-all]     Format changed/all Go files with goimports-reviser"
	@echo "  lint[-fix]       Lint each module with the shared configuration"
	@echo "  security         Run gosec in each module"
	@echo "  check-vet        Run go vet in each module"
	@echo "  deps             Tidy modules using isolated checkout dependencies"
	@echo "  deps-published   Tidy against actual releases and record their checksums"
	@echo "  check-modules    Check tidy manifests without workspace replacements"
	@echo "  test[-fast|-race|-integration|-cover]  Test each module"
	@echo "  test-package PKG=./fetch              Test one module"
	@echo "  test-standalone  Test checkout artifacts with GOWORK=off"
	@echo "  test-published   Test actual pinned releases with GOWORK=off"
	@echo "  check-all        Format, lint, security, tidy check and race tests"
	@echo "  check-quick      Lint and short tests"
	@echo "  clean            Remove module coverage reports and test binaries"
	@echo "  install-tools    Install formatting, lint and security tools"

modules:
	@$(MODULES) --list

format:
	@python3 scripts/format.py

format-all:
	$(MODULES) goimports-reviser -project-name github.com/tr1v3r/pkg -rm-unused ./...

lint:
	$(MODULES) $(LINT) run --config="$(LINT_CONFIG)"

lint-fix:
	$(MODULES) $(LINT) run --fix --config="$(LINT_CONFIG)"

lint-fast:
	$(MODULES) $(LINT) run --config="$(LINT_CONFIG)" --max-issues-per-linter=10 --max-same-issues=5

lint-verbose:
	$(MODULES) $(LINT) run -v --config="$(LINT_CONFIG)"

lint-config:
	$(LINT) config path --config="$(LINT_CONFIG)"

lint-ci:
	$(MODULES) $(LINT) run --config="$(LINT_CONFIG)" --timeout=10m

security:
	$(MODULES) gosec ./...

check-vet:
	$(MODULES) go vet ./...

deps:
	./scripts/standalone.sh --tidy

deps-published:
	./scripts/standalone.sh --published --tidy

check-modules:
	./scripts/standalone.sh --check-tidy

test:
	$(MODULES) go test -v ./...

test-fast:
	$(MODULES) go test -short ./...

test-race:
	$(MODULES) go test -race -short ./...

test-cover:
	$(MODULES) go test -coverprofile=coverage.out ./...
	$(MODULES) go tool cover -html=coverage.out -o coverage.html

# Preserve the existing opt-in integration build mode.
test-integration:
	$(MODULES) go test -v -tags=integration ./...

test-package:
	@test -n "$(PKG)" || { echo "Usage: make test-package PKG=./fetch"; exit 1; }
	MODULE="$(PKG)" $(MODULES) go test -v ./...

test-standalone:
	./scripts/standalone.sh go test -race -count=1 -tags=integration ./...

test-published:
	./scripts/standalone.sh --published go test -race -count=1 -tags=integration ./...

check-all: format lint security check-modules test-race

check-quick: lint-fast test-fast

clean:
	$(MODULES) go clean
	$(MODULES) sh -c 'rm -f coverage.out coverage.html'

install-tools:
	go install github.com/incu6us/goimports-reviser@latest
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2
	go install github.com/securego/gosec/v2/cmd/gosec@latest
