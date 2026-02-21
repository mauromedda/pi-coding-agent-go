# ABOUTME: Build, test, lint, and release orchestration for pi-go
# ABOUTME: Uses goreleaser for cross-platform builds; golangci-lint for quality

SHELL := bash
.SHELLFLAGS := -eu -o pipefail -c
.DELETE_ON_ERROR:
MAKEFLAGS += --warn-undefined-variables
MAKEFLAGS += --no-builtin-rules

# ──────────────────────────────────────────────
# Variables
# ──────────────────────────────────────────────

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_SHA := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
BINARY := pi-go
MODULE := github.com/mauromedda/pi-coding-agent-go
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(GIT_SHA) -X main.date=$(BUILD_TIME)

# Colors
CYAN := \033[0;36m
GREEN := \033[0;32m
YELLOW := \033[1;33m
RED := \033[0;31m
BOLD := \033[1m
NC := \033[0m

define log_info
	@printf "$(CYAN)[INFO]$(NC) %s\n" "$(1)"
endef

define log_success
	@printf "$(GREEN)[OK]$(NC) %s\n" "$(1)"
endef

define log_step
	@printf "$(BOLD)>>> %s$(NC)\n" "$(1)"
endef

.DEFAULT_GOAL := help

##@ General

.PHONY: help
help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} \
		/^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } \
		/^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: build
build: ## Build the pi-go binary
	$(call log_step,Building $(BINARY))
	@go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/pi-go
	$(call log_success,Built $(BINARY) $(VERSION))

.PHONY: run
run: build ## Build and run pi-go
	./$(BINARY)

.PHONY: install
install: ## Install pi-go to GOPATH/bin
	$(call log_step,Installing $(BINARY))
	@go install -ldflags "$(LDFLAGS)" ./cmd/pi-go
	$(call log_success,Installed $(BINARY))

##@ Quality

.PHONY: test
test: ## Run all tests with race detector
	$(call log_step,Running tests)
	@go test -race -count=1 ./...
	$(call log_success,All tests passed)

.PHONY: test-verbose
test-verbose: ## Run all tests with verbose output
	@go test -race -count=1 -v ./...

.PHONY: test-cover
test-cover: ## Run tests with coverage report
	$(call log_step,Running tests with coverage)
	@go test -race -count=1 -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out
	$(call log_success,Coverage report generated)

.PHONY: bench
bench: ## Run benchmarks
	$(call log_step,Running benchmarks)
	@go test -bench=. -benchmem ./pkg/tui/width/...
	@go test -bench=. -benchmem ./pkg/tui/...

.PHONY: lint
lint: ## Run golangci-lint
	$(call log_step,Running linter)
	@golangci-lint run ./...
	$(call log_success,Lint passed)

.PHONY: fmt
fmt: ## Format all Go files
	@gofmt -s -w .
	@goimports -w .

.PHONY: vet
vet: ## Run go vet
	@go vet ./...

.PHONY: check
check: fmt vet lint test ## Run all quality checks

##@ Release

.PHONY: snapshot
snapshot: ## Build a snapshot release (no publish)
	@goreleaser release --snapshot --clean

.PHONY: clean
clean: ## Remove build artifacts
	@rm -f $(BINARY) coverage.out
	@rm -rf dist/
	$(call log_success,Cleaned)
