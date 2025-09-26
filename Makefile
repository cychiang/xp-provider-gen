# Crossplane Provider Generator Makefile

# Build variables
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')

# Go build flags
LDFLAGS := -X github.com/cychiang/xp-provider-gen/pkg/version.Version=$(VERSION) \
           -X github.com/cychiang/xp-provider-gen/pkg/version.GitCommit=$(GIT_COMMIT) \
           -X github.com/cychiang/xp-provider-gen/pkg/version.BuildDate=$(BUILD_DATE)

# Build flags
BUILD_FLAGS := -ldflags "$(LDFLAGS)" -trimpath

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build $(BUILD_FLAGS)
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod


# Build directories
BUILD_DIR=bin
COVERAGE_DIR=coverage

# Binary names
BINARY=xp-provider-gen

.PHONY: help build clean test coverage fmt vet lint lint-fix lint-install mod-tidy mod-verify check reviewable integration-test ci-test ci-lint docs

help: ## Show this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

build: ## Build the standalone Crossplane provider generator
	@echo "Building Crossplane provider generator..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY) ./cmd/xp-provider-gen

clean: ## Clean build artifacts and temporary files
	$(GOCLEAN)
	rm -rf $(BUILD_DIR) $(COVERAGE_DIR)

test: ## Run tests with race detection
	$(GOTEST) -v -race ./...

coverage: ## Generate test coverage report
	@mkdir -p $(COVERAGE_DIR)
	$(GOTEST) -coverprofile=$(COVERAGE_DIR)/coverage.out ./...
	$(GOCMD) tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo "Coverage report generated: $(COVERAGE_DIR)/coverage.html"

fmt: ## Format Go code
	$(GOCMD) fmt ./...

vet: ## Run go vet
	$(GOCMD) vet ./...

# Ensure golangci-lint is installed
lint-install: ## Install golangci-lint if not present
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi

lint: lint-install ## Run golangci-lint with configuration
	@echo "Running golangci-lint..."
	golangci-lint run --config .golangci.yml

mod-tidy: ## Run go mod tidy
	$(GOMOD) tidy

mod-verify: ## Verify go mod dependencies
	$(GOMOD) verify

check: fmt vet lint test ## Run all quality checks (format, vet, lint, test)
	@echo "All quality checks passed!"

reviewable: mod-tidy check ## Run all checks to make code reviewable
	@echo "Code is ready for review!"

# CI/CD targets

ci-test: ## Run tests for CI with coverage
	$(GOTEST) -race -coverprofile=coverage.out ./...

ci-lint: lint-install ## Run linting for CI with extended timeout
	@echo "Running CI linting..."
	golangci-lint run --config .golangci.yml --timeout=5m --out-format=github-actions

lint-fix: lint-install ## Run golangci-lint with auto-fixing
	@echo "Running golangci-lint with auto-fix..."
	golangci-lint run --config .golangci.yml --fix

# Documentation

docs: ## Generate documentation (placeholder)
	@echo "Documentation generation not yet implemented"
	@echo "See README.md and IMPLEMENTATION_PLAN.md for current documentation"

# Integration testing

integration-test: build ## Run comprehensive integration tests
	@echo "Running integration tests..."
	./scripts/integration-test.sh
	@echo "Integration tests completed!"
