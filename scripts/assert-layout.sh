#!/bin/bash
# Asserts the generated provider's file layout. This is the single source of
# truth for which files a scaffold must contain — called from scripts/e2e-native.sh
# at each scaffolding stage, and from scripts/e2e-upjet.sh after init.
#
# Usage:
#   assert-layout.sh [--upjet] <project-dir>                          base layout (after init)
#   assert-layout.sh <project-dir> <group> <version> <Kind>           native: base + per-kind files
set -e

flavor=native
if [ "$1" = "--upjet" ]; then
    flavor=upjet
    shift
fi

dir="$1"
group="${2:-}"
version="${3:-}"
kind="${4:-}"

fail=0
require() {
    if [ ! -e "$dir/$1" ]; then
        echo "MISSING: $1"
        fail=1
    fi
}

if [ "$flavor" = upjet ]; then
    # Base layout, present after `init --upjet`.
    require "Makefile"
    require "hack/xp-provider-gen.mk"
    require "config/provider.go"
    require "config/zz_resources.go"
    require "internal/clients/clients.go"
    require "internal/clients/resolve.go"
    require "cmd/generator/main.go"
    require "apis/generate.go"

    if [ $fail -ne 0 ]; then
        echo "layout assertion FAILED for $dir"
        exit 1
    fi
    echo "layout OK: $dir (upjet)"
    exit 0
fi

# Base layout, present after init.
require "Makefile"
require "hack/xp-provider-gen.mk"
require "go.mod"
require ".gitignore"
require "apis"
require "cmd/provider/main.go"
require "internal/controller"
require "internal/provider/connector.go"
require "internal/provider/client.go"
require "internal/provider/options.go"
require "cluster/local/integration_tests.sh"
require "test/setup.sh"
require "test/README.md"
require "docs/ownership.md"
require "AGENTS.md"

# Per-kind files, present after create api.
if [ -n "$kind" ]; then
    kind_lower=$(echo "$kind" | tr '[:upper:]' '[:lower:]')
    require "apis/$group/$version/${kind_lower}_types.go"
    require "internal/controller/$kind_lower/external.go"
    require "internal/controller/$kind_lower/wiring.go"
    # The example manifest is the single source of a valid manifest for this
    # kind: uptest's lifecycle input, create-test's input, and the docs.
    require "examples/$group/${kind_lower}.yaml"
    require "test/behavior/${kind_lower}-pause/chainsaw-test.yaml"
fi

if [ $fail -ne 0 ]; then
    echo "layout assertion FAILED for $dir"
    exit 1
fi
echo "layout OK: $dir${kind:+ ($group/$version $kind)}"
