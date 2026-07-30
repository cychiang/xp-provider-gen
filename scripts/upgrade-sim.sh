#!/bin/bash
# Upgrade-path simulation: prove a generator version bump cannot touch user logic.
#
# 1. Scaffold with generator v1, write REAL user logic in all user-owned seams
# 2. Simulate v2 of the generator by changing tool-owned templates
# 3. Run `update` and inspect exactly which files the diff touches
#
# This covers a gap in scripts/e2e-test.sh: that script runs `update` with the
# SAME generator, so tool-owned files come out byte-identical and it can only
# prove user files survive. This proves the other direction too — that tool-owned
# files actually receive a new generator's changes.
#
# Run before shipping a framework bump. Restores the templates it mutates.
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO="$(dirname "$SCRIPT_DIR")"
SIM=/tmp/upgrade-sim
B="$REPO/bin/xp-provider-gen"

blue()  { printf '\033[0;34m%s\033[0m\n' "$1"; }
green() { printf '\033[0;32m%s\033[0m\n' "$1"; }
red()   { printf '\033[0;31m%s\033[0m\n' "$1"; }

blue "=== 1. Scaffold with the current generator ==="
rm -rf "$SIM" && mkdir -p "$SIM" && cd "$SIM"
$B init --domain=acme.io --repo=github.com/example/provider-acme >/dev/null 2>&1
$B create api --group=compute --version=v1alpha1 --kind=Instance >/dev/null 2>&1
green "scaffolded at $SIM"

blue "=== 2. Write REAL user logic into every user-owned seam ==="

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
cat > internal/provider/client.go <<'EOF'
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

# options.go: real flag + validation
cat > internal/provider/options.go <<'EOF'
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

make generate >/dev/null 2>&1
make reviewable >/dev/null 2>&1 && green "user logic compiles and lints"
git add -A && git commit -q -m "feat: real ACME provider logic"
BEFORE=$(git rev-parse HEAD)
green "committed user logic at $BEFORE"

blue "=== 3. Simulate a NEW generator version (change tool-owned templates) ==="
cd "$REPO"
cp pkg/templates/files/internal/provider/connector.go.tmpl /tmp/connector.bak
cp pkg/templates/files/internal/controller/KIND/wiring.go.tmpl /tmp/wiring.bak

# A framework change lands in tool-owned code only.
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
PY
make build >/dev/null 2>&1
green "generator v2 built"

blue "=== 4. Run update in the provider ==="
cd "$SIM"
$B update >/dev/null 2>&1 && green "update completed" || { red "update FAILED"; exit 1; }

blue "=== 5. What did the update diff touch? ==="
git diff --stat | sed 's/^/  /'

blue "=== 6. Verdict ==="
FAIL=0
for f in internal/provider/client.go internal/provider/options.go \
         internal/controller/instance/external.go apis/v1alpha1/types.go AGENTS.md; do
    if git diff --name-only | grep -qx "$f"; then
        red "  ✗ USER FILE MODIFIED: $f"; FAIL=1
    else
        green "  ✓ user file untouched: $f"
    fi
done

for f in internal/provider/connector.go internal/controller/instance/wiring.go; do
    if grep -q "SIMULATED-V2-CHANGE" "$f"; then
        green "  ✓ tool file received the v2 change: $f"
    else
        red "  ✗ tool file did NOT receive the v2 change: $f"; FAIL=1
    fi
done

if grep -q "USER LOGIC observing" internal/controller/instance/external.go &&
   grep -q "providerConfig.spec.endpoint must be set" internal/provider/client.go &&
   grep -q "ACME_REGION" internal/provider/options.go &&
   grep -q "Endpoint is the ACME API base URL" apis/v1alpha1/types.go; then
    green "  ✓ all user logic still present after upgrade"
else
    red "  ✗ user logic was lost"; FAIL=1
fi

blue "=== 7. Does the upgraded provider still build? ==="
if make reviewable >/dev/null 2>&1 && make build >/dev/null 2>&1; then
    green "  ✓ upgraded provider builds and passes reviewable"
else
    red "  ✗ upgraded provider does not build"; FAIL=1
fi

blue "=== 8. Restore generator templates ==="
cp /tmp/connector.bak "$REPO/pkg/templates/files/internal/provider/connector.go.tmpl"
cp /tmp/wiring.bak "$REPO/pkg/templates/files/internal/controller/KIND/wiring.go.tmpl"
cd "$REPO" && make build >/dev/null 2>&1
green "templates restored"

exit $FAIL
