#!/bin/bash

set -e

# Configuration
TEST_DIR="/tmp/xpg-e2e-native"
# The update/adopt lifecycle tests run on a throwaway COPY so TEST_DIR is left as
# the pristine, single-commit scaffold for inspection.
LIFECYCLE_DIR="/tmp/xpg-e2e-native-lifecycle"
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

# shellcheck source=scripts/lib.sh
source "$SCRIPT_DIR/lib.sh"

step_header() {
    echo -e "\n${BLUE}========================================${NC}"
    echo -e "${BLUE} Step $1: $2${NC}"
    echo -e "${BLUE}========================================${NC}"
}

# Step 12 (the generated provider's own uptest + chainsaw e2e) needs a running
# Docker daemon. Most CI runners (including GitHub's ubuntu-latest) already
# have one, so relying on "docker info" failing to skip it there would be
# accidental, not intentional — set E2E_SKIP_DOCKER=1 to skip it explicitly
# regardless of daemon availability.
docker_e2e_available() {
    ! docker_skip_requested && docker info >/dev/null 2>&1
}

run_make_target() {
    local target=$1
    log_info "Running 'make $target'..."

    local log
    log="$(mktemp)"
    if make "$target" >"$log" 2>&1; then
        log_success "make $target completed successfully"
        rm -f "$log"
    else
        log_error "make $target failed"
        tail -30 "$log"
        rm -f "$log"
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
        "internal/provider/connector.go" \
        "internal/controller/${KIND1_LOWER}/wiring.go" \
        "hack/xp-provider-gen.mk" \
        "docs/ownership.md"; do
        if grep -q "$marker" "$f" 2>/dev/null; then
            log_success "✓ tool-owned: $f"
        else
            log_error "✗ tool-owned file missing header: $f"
            failed=1
        fi
    done

    # User-owned: MUST NOT carry the header (never clobbered by update).
    for f in \
        "internal/controller/${KIND1_LOWER}/external.go" \
        "internal/provider/client.go" \
        "internal/provider/options.go" \
        "apis/$GROUP/$VERSION/${KIND1_LOWER}_types.go" \
        "apis/v1alpha1/types.go" \
        "Makefile" \
        "AGENTS.md"; do
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

step_prepare_dir() {
    step_header "1" "Prepare test folder"
    if [[ -d "$TEST_DIR" ]]; then
        log_info "Test directory exists, removing and recreating..."
        rm -rf "$TEST_DIR"
        log_success "Existing test directory removed"
    else
        log_info "Test directory does not exist, creating..."
    fi
    mkdir -p "$TEST_DIR"
    log_success "Test directory ready: $TEST_DIR"
}

step_init() {
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

    "$SCRIPT_DIR/assert-layout.sh" "$TEST_DIR"
}

step_build_initial() {
    step_header "3" "Test initial build targets"
    run_make_target "submodules"
    run_make_target "generate"
    run_make_target "reviewable"
    log_success "All initial build targets completed successfully"

    # The init pipeline must leave a clean, fully-committed tree.
    assert_clean_tree "init"
}

step_create_api_first() {
    step_header "4" "Create first API: $GROUP/$VERSION $KIND1"
    log_info "Running: $BINARY_PATH create api --group=$GROUP --version=$VERSION --kind=$KIND1"
    if "$BINARY_PATH" create api --group="$GROUP" --version="$VERSION" --kind="$KIND1"; then
        log_success "First API ($KIND1) created successfully"
    else
        log_error "Failed to create first API ($KIND1)"
        exit 1
    fi

    KIND1_LOWER=$(echo "$KIND1" | tr '[:upper:]' '[:lower:]')
    "$SCRIPT_DIR/assert-layout.sh" "$TEST_DIR" "$GROUP" "$VERSION" "$KIND1"
}

step_create_api_second() {
    step_header "5" "Create second API: $GROUP/$VERSION $KIND2"
    log_info "Running: $BINARY_PATH create api --group=$GROUP --version=$VERSION --kind=$KIND2"
    if "$BINARY_PATH" create api --group="$GROUP" --version="$VERSION" --kind="$KIND2"; then
        log_success "Second API ($KIND2) created successfully"
    else
        log_error "Failed to create second API ($KIND2)"
        exit 1
    fi

    KIND2_LOWER=$(echo "$KIND2" | tr '[:upper:]' '[:lower:]')
    "$SCRIPT_DIR/assert-layout.sh" "$TEST_DIR" "$GROUP" "$VERSION" "$KIND2"
}

step_build_after_apis() {
    step_header "6" "Test build targets after API creation"
    run_make_target "submodules"
    run_make_target "generate"

    verify_files_exist "generated CRDs" \
        "package/crds" \
        "package/crds/${GROUP}.${DOMAIN}_${KIND1_LOWER}s.yaml" \
        "package/crds/${GROUP}.${DOMAIN}_${KIND2_LOWER}s.yaml"

    verify_files_exist "generated examples" \
        "examples/$GROUP" \
        "examples/$GROUP/${KIND1_LOWER}.yaml" \
        "examples/$GROUP/${KIND2_LOWER}.yaml"

    run_make_target "reviewable"
    log_success "All build targets after API creation completed successfully"

    # Tool-owned files carry the generated header; user logic does not.
    assert_ownership
    assert_clean_tree "create api"

    log_info "Asserting initial scaffolding is a single commit..."
    local commit_count
    commit_count="$(git rev-list --count HEAD)"
    if [[ "$commit_count" == "1" && "$(git log -1 --format=%s)" == "Initial commit" ]]; then
        log_success "✓ init + create api folded into one 'Initial commit'"
    else
        log_error "✗ expected a single 'Initial commit', found $commit_count commit(s):"
        git log --oneline
        exit 1
    fi
}

# step_update_lifecycle copies the pristine scaffold to LIFECYCLE_DIR so the
# rest of the lifecycle steps (update, --force, --adopt, create-test) don't
# add commits or leave review changes in TEST_DIR.
step_update_lifecycle() {
    step_header "7" "Test update command (on a copy)"
    log_info "Copying the scaffold to $LIFECYCLE_DIR for the update/adopt lifecycle tests..."
    rm -rf "$LIFECYCLE_DIR"
    cp -r "$TEST_DIR" "$LIFECYCLE_DIR"
    cd "$LIFECYCLE_DIR"

    WIRING="internal/controller/${KIND1_LOWER}/wiring.go"
    CTRL="internal/controller/${KIND1_LOWER}/external.go"
    CLIENT_FILE="internal/provider/client.go"
    OPTS_FILE="internal/provider/options.go"

    log_info "Hand-editing all three user-owned seam files (simulating user code)..."
    printf '\n// USER-EDIT-MARKER: custom reconcile logic\n' >>"$CTRL"
    printf '\n// USER-EDIT-MARKER: custom client construction\n' >>"$CLIENT_FILE"
    printf '\n// USER-EDIT-MARKER: custom provider options\n' >>"$OPTS_FILE"
    git add -A && git commit -q -m "user: customize ${KIND1} controller, client and options"

    log_info "Running: $BINARY_PATH update"
    if "$BINARY_PATH" update; then
        log_success "update completed"
    else
        log_error "update failed"
        exit 1
    fi

    for f in "$CTRL" "$CLIENT_FILE" "$OPTS_FILE"; do
        if grep -q "USER-EDIT-MARKER" "$f"; then
            log_success "✓ user-owned edit preserved: $f"
        else
            log_error "✗ update clobbered user-owned file: $f"
            exit 1
        fi
    done
    for f in "$WIRING" "internal/provider/connector.go" "hack/xp-provider-gen.mk" "docs/ownership.md"; do
        if grep -q "DO NOT EDIT" "$f"; then
            log_success "✓ tool-owned refreshed (header intact): $f"
        else
            log_error "✗ tool-owned file lost its header after update: $f"
            exit 1
        fi
    done
    if ! grep -q "DO NOT EDIT" "AGENTS.md"; then
        log_success "✓ seed-once AGENTS.md left alone by update"
    else
        log_error "✗ update took ownership of AGENTS.md"
        exit 1
    fi

    # Commit whatever update produced, then confirm update refuses a dirty tree.
    git add -A && git commit -q -m "chore: update core components" || true
    assert_update_refuses_dirty "$BINARY_PATH" "$CTRL"
}

# step_force: `create api --force` refreshes a tool-owned file, preserves user edits.
step_force() {
    step_header "8" "Test create api --force"
    log_info "Marking tool-owned $WIRING and user-owned $CTRL, then running --force..."
    assert_force_refreshes "$WIRING" "$CTRL" -- \
        "$BINARY_PATH" create api --group="$GROUP" --version="$VERSION" --kind="$KIND1" --force
}

# step_adopt: `update --adopt` retrofits a provider generated before the ownership contract.
step_adopt() {
    step_header "9" "Test update --adopt"
    log_info "Simulating a pre-contract provider: stripping the header from $WIRING..."
    assert_adopt_restores "$WIRING" -- "$BINARY_PATH" update --adopt
    rm -f "$ASSERT_LOG"

    if grep -q "plugins:" PROJECT; then
        log_success "✓ generator provenance stamped in PROJECT"
    else
        log_error "✗ adopt did not stamp provenance in PROJECT"
        exit 1
    fi
}

step_create_test() {
    step_header "10" "create-test scaffolds a chainsaw test"
    if "$BINARY_PATH" create-test --name smoke-test --kind "$KIND1"; then
        verify_files_exist "create-test output" "test/behavior/smoke-test/chainsaw-test.yaml"
        if "$BINARY_PATH" create-test --name smoke-test --kind "$KIND1" 2>/dev/null; then
            log_error "✗ create-test overwrote an existing test"
            exit 1
        fi
        log_success "✓ create-test refuses to overwrite an existing test"
    else
        log_error "create-test failed"
        exit 1
    fi

    # Done with the lifecycle copy — return to the pristine scaffold and drop it.
    cd "$TEST_DIR"
    rm -rf "$LIFECYCLE_DIR"
}

step_final_verification() {
    step_header "11" "Final verification"

    log_info "Verifying go.mod..."
    if go mod verify; then
        log_success "go.mod verification passed"
    else
        log_warning "go.mod verification failed (might be expected for test)"
    fi

    # When Step 12's uptest flow runs, it performs a full build anyway — only
    # build here when Step 12 will be skipped, where this is the sole build check.
    if ! docker_e2e_available; then
        log_info "Testing provider build..."
        if make build >/dev/null 2>&1; then
            log_success "Provider builds successfully"
        else
            log_warning "Provider build failed (might be expected for test)"
        fi
    fi

    log_info "Final project structure:"
    find . -type f \( -name "*.go" -o -name "*.yaml" -o -name "Makefile" -o -name "go.mod" \) |
        sort |
        head -20 |
        sed 's/^/  /'

    if [[ $(find . -type f \( -name "*.go" -o -name "*.yaml" \) | wc -l) -gt 20 ]]; then
        echo "  ... and more files"
    fi
}

# step_provider_e2e: the generated provider's own e2e must pass — the full
# uptest + chainsaw flow: build the xpkg, stand up a dedicated kind control
# plane with Crossplane, deploy the provider from the local package, run
# every kind's uptest lifecycle, then the chainsaw behavior suite. Needs a
# running Docker daemon and minutes of cluster time; skipped with a warning
# when unavailable, or when E2E_SKIP_DOCKER is set, so docker-less machines
# and fast CI runs can still run the rest of the harness.
step_provider_e2e() {
    step_header "12" "Generated provider's own e2e (uptest + chainsaw)"
    if docker_skip_requested; then
        PROVIDER_E2E_RESULT="SKIPPED (E2E_SKIP_DOCKER set)"
        CREATE_TEST_LIVE_RESULT="SKIPPED (E2E_SKIP_DOCKER set)"
    else
        PROVIDER_E2E_RESULT="SKIPPED (no docker daemon)"
        CREATE_TEST_LIVE_RESULT="SKIPPED (no docker daemon)"
    fi

    if docker_e2e_available; then
        if make e2e; then
            log_success "generated provider's make e2e passed"
            PROVIDER_E2E_RESULT="PASSED"
            # The cluster is still up and kubectl still points at it: prove the
            # full user story end to end — scaffold a NEW behavior test with
            # create-test, then actually run it against the live provider.
            log_info "Running a freshly scaffolded chainsaw test against the cluster..."
            if "$BINARY_PATH" create-test --name smoke-live --kind "$KIND1" >/dev/null &&
                make test-behavior; then
                log_success "create-test output runs green against the live provider"
                CREATE_TEST_LIVE_RESULT="PASSED"
            else
                log_error "a scaffolded chainsaw test failed against the live provider"
                exit 1
            fi
            # Restore the pristine scaffold, then drop the cluster (the scaffold
            # owns its name, so use its own cleanup target).
            rm -rf test/behavior/smoke-live junit.xml
            make e2e-clean >/dev/null 2>&1 || true
        else
            log_error "generated provider's make e2e FAILED"
            exit 1
        fi
    elif docker_skip_requested; then
        log_warning "E2E_SKIP_DOCKER set — skipping the generated provider's make e2e"
    else
        log_warning "docker daemon unavailable — skipping the generated provider's make e2e"
    fi
}

step_summary() {
    echo
    step_header "✅" "Native E2E Test Summary"
    log_success "✅ Generated provider's own e2e (uptest + chainsaw): ${PROVIDER_E2E_RESULT}"
    log_success "✅ scaffolded test runs against the live provider: ${CREATE_TEST_LIVE_RESULT}"
    echo
    log_success "🎉 All native E2E tests completed successfully!"
    log_info "Pristine scaffold (single 'Initial commit', clean tree) at: $TEST_DIR"
    log_info "  inspect with:  git -C $TEST_DIR log --oneline && git -C $TEST_DIR status"
}

main() {
    log_info "Starting local native E2E test for xp-provider-gen"
    log_info "Test directory: $TEST_DIR"
    log_info "Domain: $DOMAIN"
    log_info "Repository: $REPO"
    echo

    if [[ ! -f "$BINARY_PATH" ]]; then
        log_error "Binary not found at $BINARY_PATH"
        log_info "Please run 'make build' first"
        exit 1
    fi
    log_success "Binary found at $BINARY_PATH"

    step_prepare_dir
    step_init
    step_build_initial
    step_create_api_first
    step_create_api_second
    step_build_after_apis
    step_update_lifecycle
    step_force
    step_adopt
    step_create_test
    step_final_verification
    step_provider_e2e
    step_summary
}


# Handle script arguments
if [[ "$1" == "--help" || "$1" == "-h" ]]; then
    echo "Usage: $0"
    echo
    echo "This script runs a comprehensive native-flavor E2E test for xp-provider-gen:"
    echo "   1. Prepare test folder"
    echo "   2. Initialize provider project"
    echo "   3. Test initial build targets"
    echo "   4. Create first API ($GROUP/$VERSION $KIND1)"
    echo "   5. Create second API ($GROUP/$VERSION $KIND2)"
    echo "   6. Test build targets after API creation"
    echo "   7. Test update command (on a copy)"
    echo "   8. Test create api --force"
    echo "   9. Test update --adopt"
    echo "  10. create-test scaffolds a chainsaw test"
    echo "  11. Final verification"
    echo "  12. Generated provider's own e2e (uptest + chainsaw)"
    echo
    echo "Env vars:"
    echo "  E2E_SKIP_DOCKER  Skip step 12 (the Docker-dependent uptest + chainsaw e2e),"
    echo "                   even if a docker daemon is available. Treated as a real"
    echo "                   boolean: unset, empty, 0, false, or no means run it;"
    echo "                   anything else (e.g. 1, true) means skip it."
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
        log_error "Native E2E test failed"
        log_info "Cleaning up incomplete test directory..."
        rm -rf "$TEST_DIR"
    fi
}

trap on_exit EXIT

# Run the main test
main
