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

# E2E_SKIP_DOCKER is a real boolean, not a "set means yes" flag: unset, empty,
# 0/false/no means "do not skip"; anything else (1, true, yes, ...) means
# "skip". Centralized here so the truthiness test isn't duplicated elsewhere.
docker_skip_requested() {
  case "${E2E_SKIP_DOCKER:-0}" in
    0 | false | False | FALSE | no | No | NO) return 1 ;;
    *) return 0 ;;
  esac
}

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

blue "  --- 6a. Provider binary builds via the scaffold's own build system, and --help works ---"
# Use the scaffold's own "make go.build" (the Docker-free half of "make build")
# rather than a bare "go build -o", which bypasses the Makefile's GO_PROJECT/
# PROJECT_REPO-based import path resolution entirely — that is exactly how a
# hard-coded PROJECT_REPO in the upjet Makefile stayed invisible before.
make go.build >/tmp/e2e-upjet-provider-build.log 2>&1 || {
  tail -20 /tmp/e2e-upjet-provider-build.log
  fail "provider binary failed to build via the scaffold's own build system (make go.build)"
}
PROVIDER_BIN="$(find _output/bin -type f -name provider | head -1)"
[ -n "$PROVIDER_BIN" ] || fail "make go.build reported success but produced no provider binary"
mkdir -p "$DIR/bin"
cp "$PROVIDER_BIN" "$DIR/bin/provider"
"$DIR/bin/provider" --help >/tmp/e2e-upjet-provider-help.log 2>&1 || {
  tail -20 /tmp/e2e-upjet-provider-help.log
  fail "provider --help failed"
}
green "  ✓ provider binary builds via 'make go.build' and --help exits cleanly"

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

blue "=== 7. Full ConfigMap lifecycle against a live cluster (create/update/delete) ==="
if ! docker_skip_requested && docker info >/dev/null 2>&1; then
  blue "  --- 7a. Configure and generate a ConfigMap resource ---"
  "$BIN" create api --group=core --version=v1alpha1 --kind=ConfigMap \
    --terraform-resource=kubernetes_config_map >/dev/null
  make generate >/tmp/e2e-upjet-configmap-generate.log 2>&1 || {
    tail -20 /tmp/e2e-upjet-configmap-generate.log
    fail "make generate failed for the ConfigMap resource"
  }
  green "  ✓ kubernetes_config_map configured and generated"

  LIVE_TEARDOWN_DONE=0
  cleanup_live_cluster() {
    if [ "$LIVE_TEARDOWN_DONE" -eq 0 ]; then
      make controlplane.down >/tmp/e2e-upjet-controlplane-down.log 2>&1 || true
      LIVE_TEARDOWN_DONE=1
    fi
  }
  trap cleanup_live_cluster EXIT

  blue "  --- 7b. Build, stand up a kind cluster with Crossplane, deploy the provider ---"
  make local-deploy >/tmp/e2e-upjet-local-deploy.log 2>&1 || {
    tail -40 /tmp/e2e-upjet-local-deploy.log
    fail "make local-deploy failed (build / kind / Crossplane / provider deploy)"
  }
  green "  ✓ kind cluster up, Crossplane installed, provider deployed and Healthy"

  LIVE_KUBECTL="$(find .cache/tools -type f -name 'kubectl-*' | head -1)"
  [ -n "$LIVE_KUBECTL" ] || fail "could not locate the kubectl binary the build system downloaded"

  blue "  --- 7c. Apply ProviderConfig and credentials ---"
  KUBECTL="$LIVE_KUBECTL" ./cluster/test/setup.sh >/tmp/e2e-upjet-cluster-setup.log 2>&1 || {
    tail -20 /tmp/e2e-upjet-cluster-setup.log
    fail "cluster/test/setup.sh failed"
  }
  green "  ✓ ProviderConfig and credentials applied"

  blue "  --- 7d. CREATE: apply a ConfigMap MR and verify the real object ---"
  LIVE_MR="$(mktemp)"
  cat >"$LIVE_MR" <<'EOF'
apiVersion: core.example.m.com/v1alpha1
kind: ConfigMap
metadata:
  name: e2e-configmap
  namespace: crossplane-system
spec:
  forProvider:
    metadata:
      - name: e2e-configmap
        namespace: default
    data:
      hello: world
  providerConfigRef:
    name: default
    kind: ProviderConfig
EOF
  "$LIVE_KUBECTL" apply -f "$LIVE_MR" || fail "failed to apply the ConfigMap MR"
  "$LIVE_KUBECTL" wait configmap.core.example.m.com/e2e-configmap -n crossplane-system \
    --for=condition=Ready --timeout=5m >/tmp/e2e-upjet-configmap-wait.log 2>&1 || {
    cat /tmp/e2e-upjet-configmap-wait.log
    "$LIVE_KUBECTL" -n crossplane-system get configmap.core.example.m.com e2e-configmap -o yaml
    fail "ConfigMap MR never became Ready"
  }
  EXTERNAL_NAME="$("$LIVE_KUBECTL" get configmap.core.example.m.com e2e-configmap -n crossplane-system \
    -o jsonpath='{.metadata.annotations.crossplane\.io/external-name}')"
  [ "$EXTERNAL_NAME" = "default/e2e-configmap" ] || fail "unexpected external-name: '$EXTERNAL_NAME' (expected 'default/e2e-configmap')"
  REAL_DATA="$("$LIVE_KUBECTL" -n default get configmap e2e-configmap -o jsonpath='{.data.hello}')"
  [ "$REAL_DATA" = "world" ] || fail "real ConfigMap data mismatch after create: got '$REAL_DATA', want 'world'"
  green "  ✓ CREATE: MR Ready/Synced, external-name=$EXTERNAL_NAME, real ConfigMap data.hello=$REAL_DATA"

  blue "  --- 7e. UPDATE: change spec.forProvider.data and verify the real object changes ---"
  LIVE_MR_UPDATED="$(mktemp)"
  cat >"$LIVE_MR_UPDATED" <<'EOF'
apiVersion: core.example.m.com/v1alpha1
kind: ConfigMap
metadata:
  name: e2e-configmap
  namespace: crossplane-system
spec:
  forProvider:
    metadata:
      - name: e2e-configmap
        namespace: default
    data:
      hello: updated-value
  providerConfigRef:
    name: default
    kind: ProviderConfig
EOF
  "$LIVE_KUBECTL" apply -f "$LIVE_MR_UPDATED" || fail "failed to apply the updated ConfigMap MR"
  UPDATED_DATA=""
  for _ in $(seq 1 30); do
    UPDATED_DATA="$("$LIVE_KUBECTL" -n default get configmap e2e-configmap -o jsonpath='{.data.hello}' 2>/dev/null || true)"
    [ "$UPDATED_DATA" = "updated-value" ] && break
    sleep 2
  done
  [ "$UPDATED_DATA" = "updated-value" ] || fail "real ConfigMap data did not update: got '$UPDATED_DATA', want 'updated-value'"
  green "  ✓ UPDATE: real ConfigMap data.hello changed to '$UPDATED_DATA'"
  rm -f "$LIVE_MR" "$LIVE_MR_UPDATED"

  blue "  --- 7f. DELETE: remove the MR and verify the real object is gone ---"
  "$LIVE_KUBECTL" delete configmap.core.example.m.com e2e-configmap -n crossplane-system --timeout=60s ||
    fail "failed to delete the ConfigMap MR"
  if "$LIVE_KUBECTL" -n default get configmap e2e-configmap >/dev/null 2>&1; then
    fail "real ConfigMap still exists after the MR was deleted"
  fi
  green "  ✓ DELETE: real ConfigMap no longer exists"

  blue "  --- 7g. Provider pod health and reconcile evidence ---"
  LIVE_POD="$("$LIVE_KUBECTL" -n crossplane-system get pods -o name | grep '^pod/provider-k8s-' | head -1)"
  [ -n "$LIVE_POD" ] || fail "could not find the provider pod"
  LIVE_PHASE="$("$LIVE_KUBECTL" -n crossplane-system get "$LIVE_POD" -o jsonpath='{.status.phase}')"
  [ "$LIVE_PHASE" = "Running" ] || fail "provider pod is not Running (phase=$LIVE_PHASE)"
  RECONCILE_LINE="$("$LIVE_KUBECTL" -n crossplane-system logs "$LIVE_POD" | grep -m1 'Reconciling.*kind=configmap' || true)"
  [ -n "$RECONCILE_LINE" ] || fail "no reconcile log line found for the configmap controller"
  echo "$RECONCILE_LINE" >/tmp/e2e-upjet-configmap-reconcile-line.log
  green "  ✓ provider pod ($LIVE_POD) Running, reconciled the ConfigMap controller"

  cleanup_live_cluster
  trap - EXIT
  green "  ✓ kind cluster torn down"
  LIFECYCLE="→ manage (live ConfigMap create/update/delete)"
else
  yellow "  ⚠ docker unavailable (or E2E_SKIP_DOCKER set) — skipping the live ConfigMap lifecycle stage"
  LIFECYCLE="(live ConfigMap lifecycle SKIPPED)"
fi

blue "=== Summary ==="
green "✅ upjet e2e passed: scaffold → configure → generate → build → run ${LIFECYCLE}"
echo "   provider left at $DIR for inspection"
