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

blue()  { printf '\033[0;34m%s\033[0m\n' "$1"; }
green() { printf '\033[0;32m%s\033[0m\n' "$1"; }
red()   { printf '\033[0;31m%s\033[0m\n' "$1"; }

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

blue "=== Summary ==="
green "✅ upjet e2e passed: scaffold → configure → generate → build"
echo "   provider left at $DIR for inspection"
