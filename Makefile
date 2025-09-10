# Kubebuilder Crossplane Plugin Makefile

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod


# Build directories
BUILD_DIR=bin
COVERAGE_DIR=coverage

# Binary names
BINARY=crossplane-provider-gen

.PHONY: help build clean test test-verbose coverage fmt vet lint mod-tidy mod-verify validate

help: ## Show this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

build: ## Build the standalone Crossplane provider generator
	@echo "Building Crossplane provider generator..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY) ./cmd/crossplane-provider-gen

clean: ## Clean build artifacts and temporary files
	$(GOCLEAN)
	rm -rf $(BUILD_DIR) $(COVERAGE_DIR)

test: ## Run tests
	$(GOTEST) -v ./pkg/plugins/crossplane/v1/

test-verbose: ## Run tests with verbose output
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

lint: ## Run golangci-lint (requires golangci-lint to be installed)
	golangci-lint run

mod-tidy: ## Run go mod tidy
	$(GOMOD) tidy

mod-verify: ## Verify go mod dependencies
	$(GOMOD) verify

validate: fmt vet lint test ## Run all validation checks (format, vet, lint, test)
	@echo "All validation checks passed!"

# Development helpers
.PHONY: dev-setup dev-check

dev-setup: mod-tidy ## Set up development environment
	@echo "Development environment setup complete!"

dev-check: validate ## Quick development check
	@echo "Development check complete!"

# CI/CD targets
.PHONY: ci-test ci-lint

ci-test: ## Run tests for CI
	$(GOTEST) -race -coverprofile=coverage.out ./...

ci-lint: ## Run linting for CI  
	golangci-lint run --timeout=5m

# Documentation
.PHONY: docs

docs: ## Generate documentation (placeholder)
	@echo "Documentation generation not yet implemented"
	@echo "See README.md and IMPLEMENTATION_PLAN.md for current documentation"

# Integration testing (when implemented)
.PHONY: integration-test

integration-test: ## Run integration tests (placeholder)
	@echo "Integration tests not yet implemented"
	@echo "This will test the plugin with actual kubebuilder CLI integration"