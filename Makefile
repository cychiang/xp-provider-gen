# Kubebuilder Crossplane Plugin Makefile

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Plugin metadata
PLUGIN_NAME=crossplane.go.kubebuilder.io
PLUGIN_VERSION=v1

# Build directories
BUILD_DIR=bin
COVERAGE_DIR=coverage

.PHONY: help build clean test test-verbose coverage fmt vet lint mod-tidy mod-verify validate

help: ## Show this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

build: ## Build the plugin library
	@echo "Building Crossplane kubebuilder plugin..."
	$(GOBUILD) ./pkg/plugins/crossplane/v1/

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
.PHONY: dev-setup dev-check plugin-info

dev-setup: mod-tidy ## Set up development environment
	@echo "Development environment setup complete!"
	@echo "Plugin: $(PLUGIN_NAME)/$(PLUGIN_VERSION)"

dev-check: validate ## Quick development check
	@echo "Development check complete!"

plugin-info: ## Show plugin information
	@echo "Plugin Name: $(PLUGIN_NAME)"
	@echo "Plugin Version: $(PLUGIN_VERSION)" 
	@echo "Supported Project Versions: [4]"
	@echo "Supported Commands: init, create api, create webhook, edit"

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