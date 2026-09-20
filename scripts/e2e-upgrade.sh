#!/bin/bash
# Upgrade E2E test: prove a generator version bump cannot touch user logic.
# Covers the native flavor only — it mutates pkg/templates/files/** tool-owned
# templates; upjet has no equivalent flow.
#
# 1. Scaffold with generator v1, write REAL user logic in all user-owned seams
# 2. Simulate v2 of the generator by changing tool-owned templates
# 3. Run `update` and inspect exactly which files the diff touches
#
# This covers a gap in scripts/e2e-native.sh: that script runs `update` with the
# SAME generator, so tool-owned files come out byte-identical and it can only
# prove user-owned files survive. This proves the other direction too — that tool-owned
# files actually receive a new generator's changes.
#
# Run before shipping a generator bump. Restores the templates it mutates.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO="$(dirname "$SCRIPT_DIR")"
DIR=/tmp/xpg-e2e-upgrade
AUX=/tmp/xpg-e2e-upgrade-aux
B="$REPO/bin/xp-provider-gen"

# shellcheck source=scripts/lib.sh
source "$SCRIPT_DIR/lib.sh"

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
    echo "Usage: $0"
    echo
    echo "Upgrade E2E test: scaffolds a provider, writes real logic into every"
    echo "user-owned seam, simulates a new generator version by mutating tool-owned"
    echo "templates, then runs 'xp-provider-gen update' and asserts user logic"
    echo "survives while tool-owned files receive the simulated change."
    exit 0
fi

# apache_header prints the Apache 2.0 file header written into every
# user-owned seam file below — inlined four times here as literal Go source,
# so it can't be read from the scaffold's own hack/boilerplate.go.txt (that
# file doesn't exist at this path relative to $DIR).
apache_header() {
    cat <<'EOF'
/*
Copyright 2025 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
EOF
}

step_header 1 "Scaffold with the current generator"
rm -rf "$DIR" && mkdir -p "$DIR"
rm -rf "$AUX" && mkdir -p "$AUX"
cd "$DIR"
$B init --domain=acme.io --repo=github.com/example/provider-acme >/dev/null 2>&1
$B create api --group=compute --version=v1alpha1 --kind=Instance >/dev/null 2>&1
log_success "scaffolded at $DIR"

step_header 2 "Write REAL user logic into every user-owned seam"

# ProviderConfig gains a user field
python3 - <<'PY'
import pathlib
p = pathlib.Path("apis/v1alpha1/types.go")
s = p.read_text()
s = s.replace(
    "type ProviderConfigSpec struct {\n\t// Credentials required to authenticate to this provider.\n\tCredentials ProviderCredentials `json:\"credentials\"`\n}",
    "type ProviderConfigSpec struct {\n\t// Credentials required to authenticate to this provider.\n\tCredentials ProviderCredentials `json:\"credentials\"`\n\n\t// Endpoint is the ACME API base URL.\n\t// +optional\n\tEndpoint string `json:\"endpoint,omitempty\"`\n}")
p.write_text(s)
PY

# client.go: real client built from the user's own spec field + a flag
{
    apache_header
    cat <<'EOF'

package provider

import (
	"context"
	"errors"
	"net/http"
	"time"
)

// Client is USER LOGIC: a real HTTP client for the ACME API.
type Client struct {
	HTTP     *http.Client
	Endpoint string
	Region   string
	Token    string
}

// NewClient is USER LOGIC: reads the user's own ProviderConfig field and flag.
func NewClient(_ context.Context, cfg ClientConfig) (*Client, error) {
	if cfg.Spec.Endpoint == "" {
		return nil, errors.New("providerConfig.spec.endpoint must be set")
	}
	return &Client{
		HTTP:     &http.Client{Timeout: 30 * time.Second},
		Endpoint: cfg.Spec.Endpoint,
		Region:   *region,
		Token:    string(cfg.Credentials),
	}, nil
}
EOF
} > internal/provider/client.go

# options.go: real flag + validation
{
    apache_header
    cat <<'EOF'

package provider

import (
	"errors"

	"github.com/alecthomas/kingpin/v2"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
)

// region is USER LOGIC: a package var shared with client.go, same package.
var region = new(string)

// Flags is USER LOGIC.
func Flags(app *kingpin.Application) {
	region = app.Flag("region", "ACME region.").Envar("ACME_REGION").Default("us-east-1").String()
}

// Configure is USER LOGIC: validates the flag combination before startup.
func Configure(o *controller.Options) error {
	if *region == "" {
		return errors.New("--region must not be empty")
	}
	o.PollInterval = 2 * o.PollInterval
	return nil
}
EOF
} > internal/provider/options.go

# external.go: real reconcile logic + reconciler options
python3 - <<'PY'
import pathlib
p = pathlib.Path("internal/controller/instance/external.go")
s = p.read_text()
s = s.replace(
    'fmt.Printf("Observing: %+v", cr)',
    'fmt.Printf("USER LOGIC observing %s at %s in %s\\n", cr.GetName(), e.client.Endpoint, e.client.Region)')
s = s.replace(
    "func ReconcilerOptions(_ ctrl.Manager, _ controller.Options) ([]managed.ReconcilerOption, error) {\n\treturn nil, nil\n}",
    "// ReconcilerOptions is USER LOGIC.\nfunc ReconcilerOptions(mgr ctrl.Manager, o controller.Options) ([]managed.ReconcilerOption, error) {\n\tif mgr == nil {\n\t\treturn nil, errors.New(\"nil manager\")\n\t}\n\treturn []managed.ReconcilerOption{managed.WithPollInterval(o.PollInterval)}, nil\n}")
p.write_text(s)
PY

# USER tests: pin the behavior of every seam. Run before AND after the upgrade —
# passing both times proves the upgrade changed plumbing, not semantics.
{
    apache_header
    cat <<'EOF'

package provider

import (
	"context"
	"testing"
	"time"

	"github.com/alecthomas/kingpin/v2"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"

	apisv1alpha1 "github.com/example/provider-acme/apis/v1alpha1"
)

func TestNewClientRequiresEndpoint(t *testing.T) {
	if _, err := NewClient(context.Background(), ClientConfig{}); err == nil {
		t.Fatal("NewClient with empty endpoint: want error, got nil")
	}
}

func TestNewClientBuildsFromSpecAndFlag(t *testing.T) {
	*region = "eu-central-1"
	c, err := NewClient(context.Background(), ClientConfig{
		Spec:        apisv1alpha1.ProviderConfigSpec{Endpoint: "https://api.acme.io"},
		Credentials: []byte("secret-token"),
	})
	if err != nil {
		t.Fatalf("NewClient: unexpected error: %v", err)
	}
	if c.Endpoint != "https://api.acme.io" || c.Token != "secret-token" || c.Region != "eu-central-1" {
		t.Fatalf("NewClient: unexpected client %+v", c)
	}
}

func TestFlagsParseRegion(t *testing.T) {
	app := kingpin.New("test", "")
	Flags(app)
	if _, err := app.Parse([]string{"--region=eu-west-1"}); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if *region != "eu-west-1" {
		t.Fatalf("region: want eu-west-1, got %q", *region)
	}
}

func TestConfigureDoublesPollInterval(t *testing.T) {
	*region = "us-east-1"
	o := &controller.Options{PollInterval: time.Minute}
	if err := Configure(o); err != nil {
		t.Fatalf("Configure: unexpected error: %v", err)
	}
	if o.PollInterval != 2*time.Minute {
		t.Fatalf("PollInterval: want 2m0s, got %s", o.PollInterval)
	}
}

func TestConfigureRejectsEmptyRegion(t *testing.T) {
	*region = ""
	if err := Configure(&controller.Options{}); err == nil {
		t.Fatal("Configure with empty region: want error, got nil")
	}
}
EOF
} > internal/provider/client_test.go

{
    apache_header
    cat <<'EOF'

package instance

import (
	"context"
	"testing"
	"time"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource/fake"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/example/provider-acme/apis/compute/v1alpha1"
	"github.com/example/provider-acme/internal/provider"
)

// stubManager is a non-nil ctrl.Manager whose methods are never called:
// the user's ReconcilerOptions only nil-checks it.
type stubManager struct{ ctrl.Manager }

func TestObserveNewInstanceEntersCreateFlow(t *testing.T) {
	e := NewExternal(&provider.Client{Endpoint: "https://api.acme.io", Region: "us-east-1"})
	obs, err := e.Observe(context.Background(), &v1alpha1.Instance{})
	if err != nil {
		t.Fatalf("Observe: unexpected error: %v", err)
	}
	if obs.ResourceExists {
		t.Fatal("Observe on a fresh Instance: want ResourceExists=false (create flow)")
	}
}

func TestObserveRejectsWrongKind(t *testing.T) {
	e := NewExternal(&provider.Client{})
	if _, err := e.Observe(context.Background(), &fake.Managed{}); err == nil || err.Error() != errNotInstance {
		t.Fatalf("Observe(wrong kind): want %q, got %v", errNotInstance, err)
	}
}

func TestReconcilerOptions(t *testing.T) {
	if _, err := ReconcilerOptions(nil, controller.Options{}); err == nil {
		t.Fatal("ReconcilerOptions(nil manager): want error, got nil")
	}
	opts, err := ReconcilerOptions(stubManager{}, controller.Options{PollInterval: time.Minute})
	if err != nil {
		t.Fatalf("ReconcilerOptions: unexpected error: %v", err)
	}
	if len(opts) != 1 {
		t.Fatalf("ReconcilerOptions: want 1 option, got %d", len(opts))
	}
}
EOF
} > internal/controller/instance/external_test.go

# generate+lint here; the baseline behavioral step below is the single test run
make generate >/dev/null 2>&1 && make lint >/dev/null 2>&1 && log_success "user logic compiles and lints"
git add -A && git commit -q -m "feat: real ACME provider logic"
BEFORE=$(git rev-parse HEAD)
log_success "committed user logic at $BEFORE"

step_header 3 "Baseline behavior: seam tests + flag reachability"
if go test ./... >/dev/null 2>&1; then
    log_success "  ✓ behavioral tests pass before upgrade"
else
    log_error "  ✗ behavioral tests FAIL before upgrade — harness broken"
    go test ./...
    exit 1
fi
go build -o "$AUX/provider" ./cmd/provider
if grep -q -- '--region' <<<"$("$AUX/provider" --help 2>&1)"; then
    log_success "  ✓ user flag --region reachable before upgrade"
else
    log_error "  ✗ user flag --region missing before upgrade — harness broken"
    exit 1
fi

step_header 4 "Simulate a NEW generator version (change tool-owned templates)"
cd "$REPO"
git diff --quiet -- pkg/templates ||
  fail "pkg/templates is dirty — git checkout it before running"
cp pkg/templates/files/internal/provider/connector.go.tmpl "$AUX/connector.bak"
cp pkg/templates/files/internal/controller/KIND/wiring.go.tmpl "$AUX/wiring.bak"
cp pkg/templates/files/hack/xp-provider-gen.mk.tmpl "$AUX/xp-provider-gen.mk.bak"
cp pkg/templates/files/cluster/local/integration_tests.sh.tmpl "$AUX/integration_tests.sh.bak"

# From here on the repo's templates are mutated: restore them on ANY exit —
# success, assertion failure, or a set -e abort mid-run — so a failed run can
# never leave the working tree (and bin/) built from simulated-v2 templates.
# $AUX itself is only cleaned at script start (step 1), never here: the trap
# fires on every exit path, and a mid-run cleanup would delete the very
# backups this restore needs.
restore_templates() {
    section_header "Restore generator templates"
    cp "$AUX/connector.bak" "$REPO/pkg/templates/files/internal/provider/connector.go.tmpl"
    cp "$AUX/wiring.bak" "$REPO/pkg/templates/files/internal/controller/KIND/wiring.go.tmpl"
    cp "$AUX/xp-provider-gen.mk.bak" "$REPO/pkg/templates/files/hack/xp-provider-gen.mk.tmpl"
    cp "$AUX/integration_tests.sh.bak" "$REPO/pkg/templates/files/cluster/local/integration_tests.sh.tmpl"
    (cd "$REPO" && make build >/dev/null 2>&1) || true
    log_success "templates restored"
}
trap restore_templates EXIT

# A generator change lands in tool-owned code only.
python3 - <<'PY'
import pathlib
p = pathlib.Path("pkg/templates/files/internal/provider/connector.go.tmpl")
s = p.read_text()
s = s.replace("// Connect implements managed.ExternalConnecter.",
              "// SIMULATED-V2-CHANGE: new framework wiring landed here.\n// Connect implements managed.ExternalConnecter.")
p.write_text(s)

p = pathlib.Path("pkg/templates/files/internal/controller/KIND/wiring.go.tmpl")
s = p.read_text()
s = s.replace("// Setup adds a controller that reconciles",
              "// SIMULATED-V2-CHANGE: new reconciler option added by the framework.\n// Setup adds a controller that reconciles")
p.write_text(s)

# A build pipeline fix lands in the tool-owned make fragment, not the Makefile.
p = pathlib.Path("pkg/templates/files/hack/xp-provider-gen.mk.tmpl")
s = p.read_text()
s = s.replace("-include build/makelib/common.mk",
              "# SIMULATED-V2-CHANGE: a build pipeline fix landed here.\n-include build/makelib/common.mk", 1)
p.write_text(s)
PY
make build >/dev/null 2>&1
log_success "generator v2 built"

step_header 5 "Run update in the provider"
cd "$DIR"
if $B update >/dev/null 2>&1; then
    log_success "update completed"
else
    log_error "update FAILED"
    exit 1
fi

step_header 6 "What did the update diff touch?"
git diff --stat | sed 's/^/  /'

step_header 7 "Verdict"
FAIL=0
DIFF_NAMES="$(git diff --name-only)"
for f in internal/provider/client.go internal/provider/options.go \
         internal/controller/instance/external.go apis/v1alpha1/types.go AGENTS.md Makefile; do
    if grep -qx "$f" <<<"$DIFF_NAMES"; then
        log_error "  ✗ USER-OWNED FILE MODIFIED: $f"; FAIL=1
    else
        log_success "  ✓ user-owned file untouched: $f"
    fi
done

for f in internal/provider/connector.go internal/controller/instance/wiring.go hack/xp-provider-gen.mk; do
    if grep -q "SIMULATED-V2-CHANGE" "$f"; then
        log_success "  ✓ tool-owned file received the v2 change: $f"
    else
        log_error "  ✗ tool-owned file did NOT receive the v2 change: $f"; FAIL=1
    fi
done

# The make fragment is how build pipeline fixes reach an existing provider: it
# must stay tool-owned and show up in the update's diff.
if grep -q "Code generated by xp-provider-gen. DO NOT EDIT." hack/xp-provider-gen.mk &&
   grep -qx "hack/xp-provider-gen.mk" <<<"$DIFF_NAMES"; then
    log_success "  ✓ hack/xp-provider-gen.mk carries the header and was refreshed by update"
else
    log_error "  ✗ hack/xp-provider-gen.mk lost its header or was not refreshed"; FAIL=1
fi

if grep -q "USER LOGIC observing" internal/controller/instance/external.go &&
   grep -q "providerConfig.spec.endpoint must be set" internal/provider/client.go &&
   grep -q "ACME_REGION" internal/provider/options.go &&
   grep -q "Endpoint is the ACME API base URL" apis/v1alpha1/types.go; then
    log_success "  ✓ all user logic still present after upgrade"
else
    log_error "  ✗ user logic was lost"; FAIL=1
fi

step_header 8 "Does the upgraded provider still build?"
# generate+lint+build; step 9 is the single post-upgrade test run
if make generate >/dev/null 2>&1 && make lint >/dev/null 2>&1 && make build >/dev/null 2>&1; then
    log_success "  ✓ upgraded provider generates, lints and builds"
else
    log_error "  ✗ upgraded provider does not build"; FAIL=1
fi

step_header 9 "Behavior unchanged after upgrade?"
if go test ./... >/dev/null 2>&1; then
    log_success "  ✓ behavioral tests pass after upgrade"
else
    log_error "  ✗ behavioral tests FAIL after upgrade"
    go test ./... | tail -20 || true
    FAIL=1
fi
if go build -o "$AUX/provider" ./cmd/provider &&
   grep -q -- '--region' <<<"$("$AUX/provider" --help 2>&1)"; then
    log_success "  ✓ user flag --region still reachable after upgrade"
else
    log_error "  ✗ user flag --region lost after upgrade"; FAIL=1
fi

step_header 10 "update removes a tool-owned file the generator no longer produces"
git add -A && git commit -qm "chore: update to simulated v2"
mkdir -p _output && printf '%s\n\npackage x\n' '// Code generated by xp-provider-gen. DO NOT EDIT.' > _output/copied.go
cd "$REPO"
rm pkg/templates/files/cluster/local/integration_tests.sh.tmpl   # backup already made in step 4; the trap restores it
make build >/dev/null 2>&1
cd "$DIR"
$B update >"$AUX/update.log" 2>&1 || { tail -30 "$AUX/update.log"; fail "update failed after a template was removed"; }
[ ! -e cluster/local/integration_tests.sh ]                            || fail "orphaned tool-owned file was not removed"
grep -q 'removed 1' "$AUX/update.log"                                  || fail "summary does not report removed 1"
grep -q 'removed cluster/local/integration_tests.sh' "$AUX/update.log" || fail "removal was not named"
[ -f _output/copied.go ]                                               || fail "a gitignored headered file was deleted"
[ "$(git status --porcelain | sort)" = "$(printf ' D cluster/local/integration_tests.sh\n M docs/ownership.md' | sort)" ] \
    || { git status --porcelain; fail "update touched more than the orphan and docs/ownership.md"; }
log_success "  ✓ orphaned tool-owned file removed, reported, and nothing else touched"

exit $FAIL
