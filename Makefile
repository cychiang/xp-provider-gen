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

# Pinned so local runs and CI enforce the same rules. Keep in step with the
# version lint.yml installs and with GOLANGCILINT_VERSION in the generated
# provider's Makefile.tmpl.
GOLANGCILINT_VERSION = 2.12.2

.PHONY: help build clean test coverage fmt vet lint lint-fix lint-install gosec mod-tidy mod-verify check reviewable e2e-test upgrade-sim

help: ## Show this help message
	@echo "Available targets:"
	@echo ""
	@echo "Development:"
	@grep -E '^(build|clean):.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'
	@echo ""
	@echo "Testing:"
	@grep -E '^(test|coverage|e2e-test|upgrade-sim):.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'
	@echo ""
	@echo "Code Quality:"
	@grep -E '^(fmt|vet|lint|lint-fix|gosec|check|reviewable):.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'
	@echo ""
	@echo "Dependencies:"
	@grep -E '^(mod-tidy|mod-verify):.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'
	@echo ""
	@echo "Other:"
	@grep -E '^(help):.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

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

e2e-test: build ## Run local end-to-end test
	@echo "Running local E2E test..."
	@./scripts/e2e-test.sh

upgrade-sim: build ## Simulate a generator version bump against a provider with real user logic
	@echo "Running upgrade-path simulation..."
	@./scripts/upgrade-sim.sh

fmt: ## Format Go code
	$(GOCMD) fmt ./...

vet: ## Run go vet
	$(GOCMD) vet ./...

# Ensure golangci-lint is installed. The v2 module path is required: .golangci.yml
# is a version: "2" config, which a v1 binary cannot parse.
lint-install: # Install golangci-lint if not present (internal)
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "Installing golangci-lint v$(GOLANGCILINT_VERSION)..."; \
		go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v$(GOLANGCILINT_VERSION); \
	fi

lint: lint-install ## Run golangci-lint with configuration
	@echo "Running golangci-lint..."
	golangci-lint run --config .golangci.yml

gosec: ## Run gosec security scanner
	@echo "Running gosec security scanner..."
	gosec -fmt=text -out=gosec-report.txt -stdout -verbose=text -severity=medium -confidence=medium ./...

mod-tidy: ## Run go mod tidy
	$(GOMOD) tidy

mod-verify: ## Verify go mod dependencies
	$(GOMOD) verify

check: fmt vet lint gosec test ## Run all quality checks (format, vet, lint, security, test)
	@echo "All quality checks passed!"

reviewable: mod-tidy check ## Run all checks to make code reviewable
	@echo "Code is ready for review!"

lint-fix: lint-install ## Run golangci-lint with auto-fixing
	@echo "Running golangci-lint with auto-fix..."
	golangci-lint run --config .golangci.yml --fix

