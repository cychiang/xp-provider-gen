#!/bin/bash
# Asserts the generated provider's file layout. This is the single source of
# truth for which files a scaffold must contain — called by both the CI smoke
# test (.github/workflows/test.yml) and scripts/e2e-test.sh, so the two can
# never drift from each other again.
#
# Usage:
#   assert-layout.sh <project-dir>                          base layout (after init)
#   assert-layout.sh <project-dir> <group> <version> <Kind> base + per-kind files
set -e

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

# Base layout, present after init.
require "Makefile"
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
    require "test/e2e/${kind_lower}-lifecycle.yaml"
    require "test/behavior/${kind_lower}-pause/chainsaw-test.yaml"
fi

if [ $fail -ne 0 ]; then
    echo "layout assertion FAILED for $dir"
    exit 1
fi
echo "layout OK: $dir${kind:+ ($group/$version $kind)}"
