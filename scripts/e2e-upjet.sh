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
         internal/clients/resolve.go cmd/generator/main.go apis/generate.go Makefile \
         hack/xp-provider-gen.mk; do
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

blue "=== 6. update refreshes tool-owned files and seeds no user-owned ones ==="
# update needs a clean tree, so commit what generation produced first. Then make
# two tool-owned files stale (Go source and the make fragment) and delete one
# user-owned file: update must refresh the first two and must not re-seed the
# last, whose template needs init-time Terraform settings PROJECT does not keep.
UPDATE_MARKER="// e2e-upjet: stale tool-owned content"
MK_MARKER="# e2e-upjet: stale tool-owned content"
git add -A && git commit -qm "Generate provider" || fail "could not commit the generated provider"
echo "$UPDATE_MARKER" >>config/provider.go
echo "$MK_MARKER" >>hack/xp-provider-gen.mk
rm examples/providerconfig/providerconfig.yaml
git commit -qam "Make tool-owned files stale and delete the ProviderConfig example" ||
  fail "could not commit the simulated drift"
"$BIN" update >/tmp/e2e-upjet-update.log 2>&1 || {
  tail -30 /tmp/e2e-upjet-update.log
  fail "update failed on the generated upjet provider"
}
if grep -qF "$UPDATE_MARKER" config/provider.go; then
  fail "update did not refresh tool-owned config/provider.go"
fi
if grep -qF "$MK_MARKER" hack/xp-provider-gen.mk; then
  fail "update did not refresh tool-owned hack/xp-provider-gen.mk"
fi
[ ! -e examples/providerconfig/providerconfig.yaml ] ||
  fail "update re-seeded user-owned examples/providerconfig/providerconfig.yaml"
grep -q 'Not seeded.*examples/providerconfig/providerconfig.yaml' /tmp/e2e-upjet-update.log ||
  fail "update did not list the ProviderConfig example as not seeded"
# Stage 10's test/setup.sh applies the ProviderConfig example, so bring it back
# from the commit before the simulated deletion.
git checkout HEAD~1 -- examples/providerconfig/providerconfig.yaml ||
  fail "could not restore the ProviderConfig example"
git add -A && git commit -qm "Update provider" || fail "could not commit the update"
green "  ✓ update refreshed config/provider.go and hack/xp-provider-gen.mk, did not re-seed the ProviderConfig example, and finalized"

blue "=== 7. create api --force, update --adopt, and update's dirty-tree refusal ==="

blue "  --- 7a. create api --force refreshes tool-owned files, preserves user edits ---"
# Mark a tool-owned file (the resource aggregator) and a user-owned one (the
# Secret's own config), then commit: --force must regenerate the first and
# leave the second alone. create api commits its own result (post-5b, exiting
# 0 even when --force reproduces something byte-identical), so no trailing
# commit is needed here.
FORCE_TOOL_MARKER="// e2e-upjet-force: stale tool-owned content"
FORCE_USER_MARKER="// e2e-upjet-force: user customization"
echo "$FORCE_TOOL_MARKER" >>config/zz_resources.go
echo "$FORCE_USER_MARKER" >>config/secret/config.go
git add -A && git commit -qm "simulate: stale tool-owned file and a user customization before --force" ||
  fail "could not commit the simulated --force drift"

# upjet requires --terraform-resource on every create api call, --force
# included, or it fails fast with "missing flag" (createapi.go PreScaffold).
"$BIN" create api --group=core --version=v1alpha1 --kind=Secret \
  --terraform-resource=kubernetes_secret --force >/tmp/e2e-upjet-force.log 2>&1 || {
  tail -30 /tmp/e2e-upjet-force.log
  fail "create api --force failed"
}
if grep -qF "$FORCE_TOOL_MARKER" config/zz_resources.go; then
  fail "--force did not refresh tool-owned config/zz_resources.go"
fi
if ! grep -qF "$FORCE_USER_MARKER" config/secret/config.go; then
  fail "--force clobbered user-owned config/secret/config.go"
fi
green "  ✓ --force exited 0, refreshed config/zz_resources.go, preserved config/secret/config.go"

# A second --force in a row, with nothing left to change, must still exit 0
# and must not add or amend a commit (the exact regression Task 5b fixed:
# git.go's stageAndCheck skips the commit when there is nothing staged).
HEAD_BEFORE_FORCE2="$(git rev-parse HEAD)"
"$BIN" create api --group=core --version=v1alpha1 --kind=Secret \
  --terraform-resource=kubernetes_secret --force >/tmp/e2e-upjet-force2.log 2>&1 || {
  tail -30 /tmp/e2e-upjet-force2.log
  fail "second --force (no changes) did not exit 0"
}
grep -q 'No changes to commit' /tmp/e2e-upjet-force2.log ||
  fail "second --force did not report the no-change skip"
[ "$(git rev-parse HEAD)" = "$HEAD_BEFORE_FORCE2" ] ||
  fail "second --force added or amended a commit although nothing changed"
green "  ✓ second --force exited 0 with nothing to commit, added no commit"

blue "  --- 7b. update --adopt retrofits a pre-contract provider ---"
grep -v 'Code generated by xp-provider-gen' config/provider.go >config/provider.go.tmp &&
  mv config/provider.go.tmp config/provider.go
grep -q 'DO NOT EDIT' config/provider.go && fail "failed to strip the header for the --adopt test"
git add -A && git commit -qm "simulate: provider without ownership headers" ||
  fail "could not commit the simulated pre-contract state"

"$BIN" update --adopt >/tmp/e2e-upjet-adopt.log 2>&1 || {
  tail -30 /tmp/e2e-upjet-adopt.log
  fail "update --adopt failed"
}
grep -q 'Adopted 1 tool-owned file(s)' /tmp/e2e-upjet-adopt.log ||
  fail "update --adopt did not report adopting exactly 1 tool-owned file"
grep -q 'DO NOT EDIT' config/provider.go || fail "adopt did not restore the header on config/provider.go"
# adopt also stamps the generator version into PROJECT — but stage 6's update
# already stamped the same version and committed it, so PROJECT may or may
# not show a diff here depending on whether the version changed since. Assert
# what's true either way: config/provider.go is in the diff, nothing besides
# it and (optionally) PROJECT is, and PROJECT carries a version.
ADOPT_DIFF="$(git diff --name-only | sort)"
echo "$ADOPT_DIFF" | grep -qx 'config/provider.go' || fail "adopt did not touch config/provider.go"
echo "$ADOPT_DIFF" | grep -vxE 'config/provider.go|PROJECT' | grep -q . &&
  fail "adopt touched unexpected files: $ADOPT_DIFF"
grep -q '^ *version:' PROJECT || fail "PROJECT carries no generator version after adopt"
git add -A && git commit -qm "chore: adopt tool-owned headers" || fail "could not commit the adopt result"
green "  ✓ adopt restored config/provider.go's header and stamped PROJECT, nothing else"

blue "  --- 7c. update refuses a dirty working tree ---"
printf '\n// e2e-upjet: dirty\n' >>config/provider.go
if "$BIN" update >/tmp/e2e-upjet-dirty.log 2>&1; then
  cat /tmp/e2e-upjet-dirty.log
  fail "update should have refused a dirty working tree"
fi
grep -qi 'working tree' /tmp/e2e-upjet-dirty.log ||
  fail "update's dirty-tree refusal did not mention the working tree"
git checkout -- config/provider.go
green "  ✓ update refused a dirty working tree"

blue "=== 8. The generated provider starts, not just builds ==="

blue "  --- 8a. Provider binary builds via the scaffold's own build system, and --help works ---"
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

# The Terraform CLI version is the generator's call, not this script's: it is
# recorded once in pkg/versions/dependencies.yaml and rendered into the
# scaffold's tool-owned make fragment. Read it back out of that fragment rather
# than repeating the literal here, so this e2e always runs the provider with the
# version the tool actually generated — after stage 6's update refreshed it.
TERRAFORM_VERSION="$(sed -n 's/^export TERRAFORM_VERSION[[:space:]]*?*=[[:space:]]*//p' "$DIR/hack/xp-provider-gen.mk" | head -1)"
[ -n "$TERRAFORM_VERSION" ] || fail "could not read TERRAFORM_VERSION out of hack/xp-provider-gen.mk"
green "  ✓ generated make fragment pins Terraform CLI $TERRAFORM_VERSION (from pkg/versions/dependencies.yaml)"

blue "  --- 8b. Scheme registration runs against an unreachable API server (no cluster needed) ---"
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
  --terraform-version="$TERRAFORM_VERSION" --terraform-provider-source=hashicorp/kubernetes \
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

blue "  --- 8c. Controller setup runs against a real (ephemeral) API server ---"
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
    -terraform-version "$TERRAFORM_VERSION" \
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
  yellow "  ⚠ envtest assets unavailable (no network, or nothing cached) — skipping stage 8c"
  tail -10 "$ENVTEST_SETUP_LOG"
fi
rm -f "$ENVTEST_SETUP_LOG"

blue "=== 9. Configure the ConfigMap resource and scaffold its own e2e (no Docker needed) ==="
"$BIN" create api --group=core --version=v1alpha1 --kind=ConfigMap \
  --terraform-resource=kubernetes_config_map >/dev/null
make generate >/tmp/e2e-upjet-configmap-generate.log 2>&1 || {
  tail -20 /tmp/e2e-upjet-configmap-generate.log
  fail "make generate failed for the ConfigMap resource"
}
green "  ✓ kubernetes_config_map configured and generated"

# docs/upjet-provider.md's worked example (§6), unchanged: this is the one
# file both uptest (UPTEST_INPUT_MANIFESTS) and create-test read, so it
# doubles as the live lifecycle input and the chainsaw skeleton's seed data.
mkdir -p examples/core
cat >examples/core/configmap.yaml <<'EOF'
# uptest.upbound.io/* annotations make this the make e2e lifecycle input too.
apiVersion: core.example.m.com/v1alpha1
kind: ConfigMap
metadata:
  name: example
  namespace: crossplane-system
  annotations:
    uptest.upbound.io/timeout: "120"
    uptest.upbound.io/conditions: "Ready,Synced"
spec:
  forProvider:
    metadata:
      - name: example
        namespace: default
    data:
      hello: world
  providerConfigRef:
    name: default
    kind: ProviderConfig
EOF
green "  ✓ wrote examples/core/configmap.yaml (uptest lifecycle input)"

"$BIN" create-test --name configmap-basic --kind ConfigMap >/dev/null
[ -f test/behavior/configmap-basic/chainsaw-test.yaml ] ||
  fail "create-test did not scaffold test/behavior/configmap-basic/chainsaw-test.yaml"
green "  ✓ create-test scaffolded test/behavior/configmap-basic/chainsaw-test.yaml"

blue "=== 10. The generated provider's own e2e (uptest + chainsaw) against a live cluster ==="
if ! docker_skip_requested && docker info >/dev/null 2>&1; then
  LIVE_TEARDOWN_DONE=0
  cleanup_live_cluster() {
    if [ "$LIVE_TEARDOWN_DONE" -eq 0 ]; then
      make e2e-clean >/tmp/e2e-upjet-e2e-clean.log 2>&1 || true
      LIVE_TEARDOWN_DONE=1
    fi
  }
  trap cleanup_live_cluster EXIT

  # make e2e: build -> controlplane.down -> controlplane.up -> deploy the
  # provider -> uptest (create -> Ready/Synced -> import -> delete, reading
  # UPTEST_INPUT_MANIFESTS=examples/*/*.yaml) -> e2e.run's test-behavior hook,
  # which runs the chainsaw suite scaffolded above. The generated chainsaw
  # skeleton sets its own cleanup timeout (pkg/templates/generators/
  # chainsaw_test.yaml.tmpl), so no CHAINSAW_ARGS override is needed here —
  # this proves the provider passes with the defaults an author actually gets.
  make e2e >/tmp/e2e-upjet-project-e2e.log 2>&1 || {
    tail -40 /tmp/e2e-upjet-project-e2e.log
    fail "generated provider's make e2e failed"
  }
  grep -q 'configmap-basic' junit.xml || fail "junit.xml does not show configmap-basic having run"
  green "  ✓ generated provider's own e2e passed (uptest lifecycle + chainsaw configmap-basic)"

  # uptest's own lifecycle never exercises UPDATE: the worked example carries
  # no uptest.upbound.io/update-parameter annotation, so uptest only runs
  # create -> Ready -> import -> delete (confirmed in the manual reproduction
  # in the task report). make e2e does not tear the cluster down, so drive an
  # update by hand here — the one upjet reconcile path otherwise left
  # completely uncovered by this e2e.
  blue "  --- Exercising UPDATE on the live cluster (uptest never triggers it) ---"
  LIVE_KUBECTL="$(find .cache/tools -type f -name 'kubectl-*' | head -1)"
  [ -n "$LIVE_KUBECTL" ] || fail "could not locate the kubectl binary the build system downloaded"

  "$LIVE_KUBECTL" apply -f examples/core/configmap.yaml || fail "failed to apply the ConfigMap example"
  "$LIVE_KUBECTL" wait configmap.core.example.m.com/example -n crossplane-system \
    --for=condition=Ready --timeout=5m >/tmp/e2e-upjet-update-wait.log 2>&1 || {
    cat /tmp/e2e-upjet-update-wait.log
    fail "ConfigMap example never became Ready"
  }
  REAL_DATA="$("$LIVE_KUBECTL" -n default get configmap example -o jsonpath='{.data.hello}')"
  [ "$REAL_DATA" = "world" ] || fail "real ConfigMap data mismatch after create: got '$REAL_DATA', want 'world'"

  "$LIVE_KUBECTL" patch configmap.core.example.m.com/example -n crossplane-system \
    --type=merge -p '{"spec":{"forProvider":{"data":{"hello":"updated-value"}}}}' ||
    fail "failed to patch the ConfigMap example"
  UPDATED_DATA=""
  for _ in $(seq 1 30); do
    UPDATED_DATA="$("$LIVE_KUBECTL" -n default get configmap example -o jsonpath='{.data.hello}' 2>/dev/null || true)"
    [ "$UPDATED_DATA" = "updated-value" ] && break
    sleep 2
  done
  [ "$UPDATED_DATA" = "updated-value" ] || fail "real ConfigMap data did not update: got '$UPDATED_DATA', want 'updated-value'"
  green "  ✓ UPDATE: real ConfigMap data.hello changed to '$UPDATED_DATA'"

  "$LIVE_KUBECTL" delete configmap.core.example.m.com/example -n crossplane-system --timeout=2m ||
    fail "failed to delete the ConfigMap example"
  if "$LIVE_KUBECTL" -n default get configmap example >/dev/null 2>&1; then
    fail "real ConfigMap still exists after the MR was deleted"
  fi
  green "  ✓ DELETE: real ConfigMap no longer exists"

  rm -rf test/behavior/configmap-basic junit.xml
  cleanup_live_cluster
  trap - EXIT
  green "  ✓ kind cluster torn down"
  LIFECYCLE="→ generated provider's own e2e (uptest + chainsaw + UPDATE)"
else
  yellow "  ⚠ docker unavailable (or E2E_SKIP_DOCKER set) — skipping the generated provider's own e2e"
  LIFECYCLE="(generated provider's own e2e SKIPPED)"
fi

blue "=== Summary ==="
green "✅ upjet e2e passed: scaffold → configure → generate → build → run ${LIFECYCLE}"
echo "   provider left at $DIR for inspection"
