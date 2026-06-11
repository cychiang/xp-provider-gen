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
# The update/adopt lifecycle tests run on a throwaway COPY so TEST_DIR is left as
# the pristine, single-commit scaffold for inspection.
LIFECYCLE_DIR="/tmp/provider-template-lifecycle"
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

assert_ownership() {
    log_info "Asserting tool-owned vs user-owned file headers..."
    local marker="DO NOT EDIT"
    local failed=0

    # Tool-owned: MUST carry the generated header.
    for f in \
        "apis/register.go" \
        "internal/controller/register.go" \
        "cmd/provider/main.go" \
        "internal/controller/config/config.go" \
        "internal/controller/${KIND1_LOWER}/setup.go"; do
        if grep -q "$marker" "$f" 2>/dev/null; then
            log_success "✓ tool-owned: $f"
        else
            log_error "✗ tool-owned file missing header: $f"
            failed=1
        fi
    done

    # User-owned: MUST NOT carry the header (never clobbered by update).
    for f in \
        "internal/controller/${KIND1_LOWER}/controller.go" \
        "apis/$GROUP/$VERSION/${KIND1_LOWER}_types.go"; do
        if grep -q "$marker" "$f" 2>/dev/null; then
            log_error "✗ user-owned file unexpectedly has header: $f"
            failed=1
        else
            log_success "✓ user-owned: $f"
        fi
    done

    [[ $failed -eq 0 ]] || return 1
    log_success "Ownership headers correct"
}

assert_clean_tree() {
    local context=$1
    log_info "Asserting clean git tree after $context..."
    local dirty
    dirty="$(git status --porcelain)"
    if [[ -n "$dirty" ]]; then
        log_error "Working tree is dirty after $context:"
        echo "$dirty"
        return 1
    fi
    log_success "Working tree is clean after $context"
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

    # The init pipeline must leave a clean, fully-committed tree.
    assert_clean_tree "init"

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

    # Tool-owned files carry the generated header; user logic does not.
    assert_ownership

    # Adding APIs must also leave a clean, fully-committed tree.
    assert_clean_tree "create api"

    # Initial scaffolding (init + create api x2) folds into a single commit.
    log_info "Asserting initial scaffolding is a single commit..."
    commit_count="$(git rev-list --count HEAD)"
    if [[ "$commit_count" == "1" && "$(git log -1 --format=%s)" == "Initial commit" ]]; then
        log_success "✓ init + create api folded into one 'Initial commit'"
    else
        log_error "✗ expected a single 'Initial commit', found $commit_count commit(s):"
        git log --oneline
        exit 1
    fi

    # Run the update/adopt lifecycle tests on a COPY, so they don't add commits or
    # leave review changes in TEST_DIR (which stays the pristine single-commit scaffold).
    log_info "Copying the scaffold to $LIFECYCLE_DIR for the update/adopt lifecycle tests..."
    rm -rf "$LIFECYCLE_DIR"
    cp -r "$TEST_DIR" "$LIFECYCLE_DIR"
    cd "$LIFECYCLE_DIR"

    # Step U: the update command refreshes tool-owned files without touching user logic
    step_header "U" "Test update command (on a copy)"
    local ctrl="internal/controller/${KIND1_LOWER}/controller.go"
    log_info "Hand-editing $ctrl and committing (simulating user business logic)..."
    printf '\n// USER-EDIT-MARKER: custom reconcile logic\n' >> "$ctrl"
    git add -A && git commit -q -m "user: customize ${KIND1} controller"

    log_info "Running: $BINARY_PATH update"
    if "$BINARY_PATH" update; then
        log_success "update completed"
    else
        log_error "update failed"
        exit 1
    fi

    if grep -q "USER-EDIT-MARKER" "$ctrl"; then
        log_success "✓ user-owned controller.go edit preserved"
    else
        log_error "✗ update clobbered user-owned controller.go"
        exit 1
    fi
    if grep -q "DO NOT EDIT" "internal/controller/${KIND1_LOWER}/setup.go"; then
        log_success "✓ tool-owned setup.go refreshed (header intact)"
    else
        log_error "✗ tool-owned setup.go lost its header after update"
        exit 1
    fi

    # Commit whatever update produced, then confirm update refuses a dirty tree.
    git add -A && git commit -q -m "chore: update core components" || true
    log_info "Verifying update refuses a dirty working tree..."
    printf '\n// dirty\n' >> "$ctrl"
    if "$BINARY_PATH" update >/dev/null 2>&1; then
        log_error "✗ update should have refused a dirty working tree"
        exit 1
    fi
    log_success "✓ update refused a dirty working tree"
    git checkout -- "$ctrl" 2>/dev/null || git restore "$ctrl"

    # Step A: `update --adopt` retrofits a provider generated before the ownership contract
    step_header "A" "Test update --adopt"
    local setup="internal/controller/${KIND1_LOWER}/setup.go"
    log_info "Simulating a pre-contract provider: stripping the header from $setup..."
    grep -v "Code generated by xp-provider-gen" "$setup" > "$setup.tmp" && mv "$setup.tmp" "$setup"
    if grep -q "DO NOT EDIT" "$setup"; then
        log_error "failed to strip header for the test"
        exit 1
    fi
    git add -A && git commit -q -m "simulate: provider without ownership headers"

    log_info "Running: $BINARY_PATH update --adopt"
    if "$BINARY_PATH" update --adopt; then
        log_success "adopt completed"
    else
        log_error "update --adopt failed"
        exit 1
    fi
    if grep -q "DO NOT EDIT" "$setup"; then
        log_success "✓ adopt restored the header on setup.go"
    else
        log_error "✗ adopt did not restore the tool-owned header"
        exit 1
    fi
    if grep -q "plugins:" PROJECT; then
        log_success "✓ generator provenance stamped in PROJECT"
    else
        log_error "✗ adopt did not stamp provenance in PROJECT"
        exit 1
    fi

    # Done with the lifecycle copy — return to the pristine scaffold and drop it.
    cd "$TEST_DIR"
    rm -rf "$LIFECYCLE_DIR"

    # Step 7: Final verification (on the pristine single-commit scaffold)
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
    log_success "✅ Single 'Initial commit' scaffold: PASSED"
    log_success "✅ update / update --adopt (on a copy): PASSED"
    echo
    log_success "🎉 All E2E tests completed successfully!"
    log_info "Pristine scaffold (single 'Initial commit', clean tree) at: $TEST_DIR"
    log_info "  inspect with:  git -C $TEST_DIR log --oneline && git -C $TEST_DIR status"
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
    # The lifecycle copy is always throwaway.
    rm -rf "$LIFECYCLE_DIR"
    if [[ $exit_code -ne 0 ]]; then
        log_error "E2E test failed"
        log_info "Cleaning up incomplete test directory..."
        rm -rf "$TEST_DIR"
    fi
}

trap on_exit EXIT

# Run the main test
main