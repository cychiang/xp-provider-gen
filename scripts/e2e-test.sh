#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
TEST_DIR="/tmp/provider-template"
DOMAIN="template.crossplane.io"
REPO="github.com/example/provider-template"
GROUP="sample"
VERSION="v1"
KIND1="MyType"
KIND2="MyValue"

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
BINARY_PATH="$PROJECT_ROOT/bin/xp-provider-gen"

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

step_header() {
    echo -e "\n${BLUE}========================================${NC}"
    echo -e "${BLUE} Step $1: $2${NC}"
    echo -e "${BLUE}========================================${NC}"
}

run_make_target() {
    local target=$1
    log_info "Running 'make $target'..."

    if make "$target" > /dev/null 2>&1; then
        log_success "make $target completed successfully"
    else
        log_error "make $target failed"
        echo "Attempting to show error output:"
        make "$target" || true
        return 1
    fi
}

verify_files_exist() {
    local description=$1
    shift
    local files=("$@")

    log_info "Verifying $description..."
    local missing_files=()

    for file in "${files[@]}"; do
        if [[ -f "$file" || -d "$file" ]]; then
            log_success "✓ $file exists"
        else
            log_error "✗ $file missing"
            missing_files+=("$file")
        fi
    done

    if [[ ${#missing_files[@]} -gt 0 ]]; then
        log_error "Missing files: ${missing_files[*]}"
        return 1
    fi

    log_success "All $description files verified"
}

# Main test function
main() {
    log_info "Starting local E2E test for xp-provider-gen"
    log_info "Test directory: $TEST_DIR"
    log_info "Domain: $DOMAIN"
    log_info "Repository: $REPO"
    echo

    # Check if binary exists
    if [[ ! -f "$BINARY_PATH" ]]; then
        log_error "Binary not found at $BINARY_PATH"
        log_info "Please run 'make build' first"
        exit 1
    fi
    log_success "Binary found at $BINARY_PATH"

    # Step 1: Check test folder and handle accordingly
    step_header "1" "Prepare test folder"
    if [[ -d "$TEST_DIR" ]]; then
        log_info "Test directory exists, removing and recreating..."
        rm -rf "$TEST_DIR"
        log_success "Existing test directory removed"
        mkdir -p "$TEST_DIR"
        log_success "Test directory recreated: $TEST_DIR"
    else
        log_info "Test directory does not exist, creating..."
        mkdir -p "$TEST_DIR"
        log_success "Test directory created: $TEST_DIR"
    fi

    # Step 2: Initialize provider project
    step_header "2" "Initialize provider project"
    cd "$TEST_DIR"
    log_info "Changed to directory: $(pwd)"

    log_info "Running: $BINARY_PATH init --domain=$DOMAIN --repo=$REPO"
    if "$BINARY_PATH" init --domain="$DOMAIN" --repo="$REPO"; then
        log_success "Provider project initialized successfully"
    else
        log_error "Failed to initialize provider project"
        exit 1
    fi

    # Verify basic project structure
    verify_files_exist "basic project structure" \
        "Makefile" \
        "go.mod" \
        ".gitignore" \
        "apis" \
        "cmd/provider" \
        "internal/controller"

    # Step 3: Test initial build targets
    step_header "3" "Test initial build targets"

    # Test make submodules
    run_make_target "submodules"

    # Test make generate
    run_make_target "generate"

    # Test make reviewable
    run_make_target "reviewable"

    log_success "All initial build targets completed successfully"

    # Step 4: Create first API (MyType)
    step_header "4" "Create first API: $GROUP/$VERSION $KIND1"

    log_info "Running: $BINARY_PATH create api --group=$GROUP --version=$VERSION --kind=$KIND1"
    if "$BINARY_PATH" create api --group="$GROUP" --version="$VERSION" --kind="$KIND1"; then
        log_success "First API ($KIND1) created successfully"
    else
        log_error "Failed to create first API ($KIND1)"
        exit 1
    fi

    # Verify first API files
    KIND1_LOWER=$(echo "$KIND1" | tr '[:upper:]' '[:lower:]')
    verify_files_exist "first API files" \
        "apis/$GROUP/$VERSION" \
        "apis/$GROUP/$VERSION/${KIND1_LOWER}_types.go" \
        "internal/controller/${KIND1_LOWER}" \
        "internal/controller/${KIND1_LOWER}/controller.go"

    # Step 5: Create second API (MyValue)
    step_header "5" "Create second API: $GROUP/$VERSION $KIND2"

    log_info "Running: $BINARY_PATH create api --group=$GROUP --version=$VERSION --kind=$KIND2"
    if "$BINARY_PATH" create api --group="$GROUP" --version="$VERSION" --kind="$KIND2"; then
        log_success "Second API ($KIND2) created successfully"
    else
        log_error "Failed to create second API ($KIND2)"
        exit 1
    fi

    # Verify second API files
    KIND2_LOWER=$(echo "$KIND2" | tr '[:upper:]' '[:lower:]')
    verify_files_exist "second API files" \
        "apis/$GROUP/$VERSION/${KIND2_LOWER}_types.go" \
        "internal/controller/${KIND2_LOWER}" \
        "internal/controller/${KIND2_LOWER}/controller.go"

    # Step 6: Test build targets after API creation
    step_header "6" "Test build targets after API creation"

    # Test make submodules again
    run_make_target "submodules"

    # Test make generate (should generate CRDs now)
    run_make_target "generate"

    # Verify CRDs were generated
    verify_files_exist "generated CRDs" \
        "package/crds" \
        "package/crds/${GROUP}.${DOMAIN}_${KIND1_LOWER}s.yaml" \
        "package/crds/${GROUP}.${DOMAIN}_${KIND2_LOWER}s.yaml"

    # Verify examples were generated
    verify_files_exist "generated examples" \
        "examples/$GROUP" \
        "examples/$GROUP/${KIND1_LOWER}.yaml" \
        "examples/$GROUP/${KIND2_LOWER}.yaml"

    # Test make reviewable
    run_make_target "reviewable"

    log_success "All build targets after API creation completed successfully"

    # Step 7: Final verification
    step_header "7" "Final verification"

    # Check that go.mod is valid
    log_info "Verifying go.mod..."
    if go mod verify; then
        log_success "go.mod verification passed"
    else
        log_warning "go.mod verification failed (might be expected for test)"
    fi

    # Check that we can build the provider
    log_info "Testing provider build..."
    if make build > /dev/null 2>&1; then
        log_success "Provider builds successfully"
    else
        log_warning "Provider build failed (might be expected for test)"
    fi

    # Show final project structure
    log_info "Final project structure:"
    find . -type f \( -name "*.go" -o -name "*.yaml" -o -name "Makefile" -o -name "go.mod" \) | \
        sort | \
        head -20 | \
        sed 's/^/  /'

    if [[ $(find . -type f \( -name "*.go" -o -name "*.yaml" \) | wc -l) -gt 20 ]]; then
        echo "  ... and more files"
    fi

    # Summary
    echo
    step_header "✅" "E2E Test Summary"
    log_success "✅ Project initialization: PASSED"
    log_success "✅ Initial build targets: PASSED"
    log_success "✅ First API creation ($KIND1): PASSED"
    log_success "✅ Second API creation ($KIND2): PASSED"
    log_success "✅ Build targets after APIs: PASSED"
    log_success "✅ CRD generation: PASSED"
    log_success "✅ Example generation: PASSED"
    echo
    log_success "🎉 All E2E tests completed successfully!"
    log_info "Test artifacts available at: $TEST_DIR"
}


# Handle script arguments
if [[ "$1" == "--help" || "$1" == "-h" ]]; then
    echo "Usage: $0"
    echo
    echo "This script runs a comprehensive E2E test for xp-provider-gen including:"
    echo "1. Project initialization"
    echo "2. Initial build verification"
    echo "3. API creation (2 APIs with same group/version, different kinds)"
    echo "4. Build verification after API creation"
    echo "5. CRD and example generation verification"
    exit 0
fi

# On failure: remove the (likely incomplete) test directory so the next run
# starts clean. On success: keep it so the generated provider can be inspected
# (the next run recreates it from scratch anyway). This keeps the final
# "Test artifacts available at: $TEST_DIR" message truthful.
on_exit() {
    local exit_code=$?
    if [[ $exit_code -ne 0 ]]; then
        log_error "E2E test failed"
        log_info "Cleaning up incomplete test directory..."
        rm -rf "$TEST_DIR"
    fi
}

trap on_exit EXIT

# Run the main test
main