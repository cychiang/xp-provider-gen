#!/bin/bash
# End-to-end test for the upjet flavor.
#
# Scaffolds a provider that wraps the hashicorp/kubernetes Terraform provider,
# configures a resource, then runs the REAL upjet pipeline: download Terraform,
# read the provider schema, scrape its docs, generate API types, controllers and
# CRDs, and build the result. That is what proves the config files this tool
# scaffolds actually satisfy upjet's contract.
#
# Needs network access and takes several minutes. Requires Go and git; Terraform
# and goimports are installed into the project by its own Makefile.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO="$(dirname "$SCRIPT_DIR")"
BIN="$REPO/bin/xp-provider-gen"
DIR=/tmp/provider-upjet-e2e

blue()   { printf '\033[0;34m%s\033[0m\n' "$1"; }
green()  { printf '\033[0;32m%s\033[0m\n' "$1"; }
red()    { printf '\033[0;31m%s\033[0m\n' "$1"; }
yellow() { printf '\033[1;33m%s\033[0m\n' "$1"; }

fail() { red "  ✗ $1"; exit 1; }

[ -x "$BIN" ] || fail "binary not found at $BIN — run 'make build' first"

blue "=== 1. Scaffold an upjet provider for hashicorp/kubernetes ==="
rm -rf "$DIR" && mkdir -p "$DIR" && cd "$DIR"
"$BIN" init --domain=example.com --repo=github.com/example/provider-k8s \
  --upjet --terraform-provider=hashicorp/kubernetes --terraform-provider-version=2.38.0 >/dev/null
green "  ✓ scaffolded"

for f in config/provider.go config/zz_resources.go internal/clients/clients.go \
         internal/clients/resolve.go cmd/generator/main.go apis/generate.go Makefile; do
  [ -f "$f" ] || fail "missing scaffolded file: $f"
done
grep -q 'hashicorp/kubernetes' Makefile || fail "Terraform provider not wired into the Makefile"
green "  ✓ upjet config surface present and wired"

blue "=== 2. Configure a Terraform resource ==="
"$BIN" create api --group=core --version=v1alpha1 --kind=Secret \
  --terraform-resource=kubernetes_secret >/dev/null
[ -f config/secret/config.go ] || fail "per-resource config was not created"
grep -q 'secret.Configure' config/zz_resources.go || fail "resource not wired into the aggregator"
grep -q 'secret.TerraformResource' config/zz_resources.go || fail "resource missing from the include list"
green "  ✓ kubernetes_secret configured and wired"

blue "=== 3. Run the upjet generation pipeline (make generate) ==="
make generate >/tmp/e2e-upjet-generate.log 2>&1 || {
  tail -20 /tmp/e2e-upjet-generate.log
  fail "make generate failed"
}
green "  ✓ generation completed"

blue "=== 4. Assert upjet produced the provider ==="
[ -f apis/cluster/core/v1alpha1/zz_secret_types.go ] || fail "API types were not generated"
[ -f internal/controller/cluster/core/secret/zz_controller.go ] || fail "controller was not generated"
[ -f apis/cluster/zz_register.go ] || fail "scheme registration was not generated"
[ -f internal/controller/cluster/zz_setup.go ] || fail "controller setup was not generated"
ls package/crds/*secrets.yaml >/dev/null 2>&1 || fail "CRDs were not generated"
grep -q 'kubernetes' apis/cluster/core/v1alpha1/zz_secret_types.go || fail "generated types do not reflect the provider schema"
green "  ✓ types, controllers, registration and CRDs generated from the real schema"

blue "=== 5. The generated provider builds ==="
go build ./... >/tmp/e2e-upjet-build.log 2>&1 || {
  tail -20 /tmp/e2e-upjet-build.log
  fail "generated provider does not build"
}
green "  ✓ builds"

blue "=== 6. The generated provider starts, not just builds ==="

blue "  --- 6a. Provider binary builds and --help works ---"
mkdir -p "$DIR/bin"
go build -o "$DIR/bin/provider" ./cmd/provider/ >/tmp/e2e-upjet-provider-build.log 2>&1 || {
  tail -20 /tmp/e2e-upjet-provider-build.log
  fail "provider binary failed to build"
}
"$DIR/bin/provider" --help >/tmp/e2e-upjet-provider-help.log 2>&1 || {
  tail -20 /tmp/e2e-upjet-provider-help.log
  fail "provider --help failed"
}
green "  ✓ provider binary builds and --help exits cleanly"

blue "  --- 6b. Scheme registration runs against an unreachable API server (no cluster needed) ---"
FAKE_KUBECONFIG="$(mktemp)"
cat >"$FAKE_KUBECONFIG" <<'EOF'
apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://127.0.0.1:1
    insecure-skip-tls-verify: true
  name: fake
contexts:
- context:
    cluster: fake
    user: fake
  name: fake
current-context: fake
users:
- name: fake
  user:
    token: fake-token
EOF
SCHEME_LOG="$(mktemp)"
KUBECONFIG="$FAKE_KUBECONFIG" "$DIR/bin/provider" \
  --terraform-version=1.5.7 --terraform-provider-source=hashicorp/kubernetes \
  --terraform-provider-version=2.38.0 --debug >"$SCHEME_LOG" 2>&1 &
SCHEME_PID=$!
( sleep 15 && kill -KILL "$SCHEME_PID" 2>/dev/null ) &
SCHEME_WATCHDOG=$!
wait "$SCHEME_PID" 2>/dev/null || true
kill "$SCHEME_WATCHDOG" 2>/dev/null || true
wait "$SCHEME_WATCHDOG" 2>/dev/null || true
rm -f "$FAKE_KUBECONFIG"

if grep -qE 'panic:|runtime error' "$SCHEME_LOG"; then
  tail -30 "$SCHEME_LOG"
  rm -f "$SCHEME_LOG"
  fail "provider panicked registering its scheme against an unreachable API server"
fi
if ! grep -q 'SafeStart precheck failed' "$SCHEME_LOG"; then
  tail -30 "$SCHEME_LOG"
  rm -f "$SCHEME_LOG"
  fail "provider did not reach its expected startup precheck (scheme registration may not have run)"
fi
rm -f "$SCHEME_LOG"
green "  ✓ scheme registration runs cleanly against an unreachable API server (no panic)"

blue "  --- 6c. Controller setup runs against a real (ephemeral) API server ---"
ENVTEST_SETUP_LOG="$(mktemp)"
if ENVTEST_ASSETS="$(go run sigs.k8s.io/controller-runtime/tools/setup-envtest@latest use -p path 2>"$ENVTEST_SETUP_LOG")"; then
  HELPER_DIR="$REPO/hack/envtest-provider-check"
  HELPER_BIN="$(mktemp -d)/envtest-provider-check"
  ( cd "$HELPER_DIR" && go build -o "$HELPER_BIN" . ) >/tmp/e2e-upjet-envtest-helper-build.log 2>&1 || {
    tail -20 /tmp/e2e-upjet-envtest-helper-build.log
    fail "hack/envtest-provider-check failed to build"
  }

  HELPER_PID=""
  cleanup_envtest_check() {
    if [ -n "$HELPER_PID" ] && kill -0 "$HELPER_PID" 2>/dev/null; then
      kill -TERM "$HELPER_PID" 2>/dev/null || true
      sleep 1
      kill -KILL "$HELPER_PID" 2>/dev/null || true
    fi
  }
  trap cleanup_envtest_check EXIT

  KUBEBUILDER_ASSETS="$ENVTEST_ASSETS" "$HELPER_BIN" \
    -provider "$DIR/bin/provider" -crd-dir "$DIR/package/crds" \
    >/tmp/e2e-upjet-provider-run.log 2>&1 &
  HELPER_PID=$!
  ( sleep 90 && kill -KILL "$HELPER_PID" 2>/dev/null ) &
  HELPER_WATCHDOG=$!
  HELPER_STATUS=0
  wait "$HELPER_PID" 2>/dev/null || HELPER_STATUS=$?
  kill "$HELPER_WATCHDOG" 2>/dev/null || true
  wait "$HELPER_WATCHDOG" 2>/dev/null || true
  trap - EXIT
  HELPER_PID=""

  if [ "$HELPER_STATUS" -ne 0 ]; then
    tail -40 /tmp/e2e-upjet-provider-run.log
    fail "generated provider failed its real-cluster startup check"
  fi
  green "  ✓ provider registers its scheme and starts controllers against a real API server, no panics"
else
  yellow "  ⚠ envtest assets unavailable (no network, or nothing cached) — skipping stage 6c"
  tail -10 "$ENVTEST_SETUP_LOG"
fi
rm -f "$ENVTEST_SETUP_LOG"

blue "=== Summary ==="
green "✅ upjet e2e passed: scaffold → configure → generate → build → run"
echo "   provider left at $DIR for inspection"
