#!/usr/bin/env bash
# Golden test for cliff.toml's git-cliff rules: the next-version bump, note
# grouping (Breaking changes / Features / Bug fixes / Dependency updates),
# the six-character typographic-to-ASCII normalization, and the "nothing to
# release" case. Requires `git-cliff` on PATH (release.yml installs it from
# the pinned official tarball before running this script; see
# docs/testing.md). Runs as a step of every release.yml execution — dry-run
# and real — so a rule change that breaks any of the above turns the release
# run red before GoReleaser ever executes. Can also be run locally.
set -euo pipefail

cd "$(dirname "$0")/.."
CLIFF_TOML="$PWD/cliff.toml"

fail() {
    echo "FAIL: $*" >&2
    exit 1
}

xgit() {
    git -c user.name=xpg-test -c user.email=xpg-test@example.com "$@"
}

BUMP_REPO=$(mktemp -d)
SKIP_REPO=$(mktemp -d)
trap 'rm -rf "$BUMP_REPO" "$SKIP_REPO"' EXIT

# --- fixture 1: version bump, note grouping, ASCII normalization ---
(
    cd "$BUMP_REPO"
    git init -q
    xgit commit -q --allow-empty -m "feat: a"
    git tag v0.1.0
    xgit commit -q --allow-empty -m "fix: b" -m "BREAKING CHANGE: c"
    xgit commit -q --allow-empty -m "refactor: r" -m "BREAKING CHANGE: s" -m "Co-authored-by: x <x@x>"
    xgit commit -q --allow-empty -m "feat: em — dash"
    xgit commit -q --allow-empty -m "wip"
    xgit commit -q --allow-empty -m "deps(go): bump x"
    xgit commit -q --allow-empty -m "chore: e"
    xgit commit -q --allow-empty -m "docs: f"
)

GOT_BUMP=$(cd "$BUMP_REPO" && git-cliff --config "$CLIFF_TOML" --bumped-version 2>/dev/null)
[ "$GOT_BUMP" = "v0.2.0" ] || fail "--bumped-version: want v0.2.0, got $GOT_BUMP"

GOT_NOTES=$(cd "$BUMP_REPO" && git-cliff --config "$CLIFF_TOML" --unreleased --tag v0.2.0 --strip all 2>/dev/null)
WANT_NOTES='
### Breaking changes
- B
- R

### Features
- Em - dash

### Dependency updates
- Bump x'
[ "$GOT_NOTES" = "$WANT_NOTES" ] || fail "release notes did not match golden
--- got ---
$GOT_NOTES
--- want ---
$WANT_NOTES"

if printf '%s' "$GOT_NOTES" | LC_ALL=C grep -n '[^ -~]' >/dev/null; then
    fail "release notes contain non-ASCII text after normalization"
fi

# --- fixture 2: only skip-group commits since the last tag -> nothing to release ---
(
    cd "$SKIP_REPO"
    git init -q
    xgit commit -q --allow-empty -m "feat: a"
    git tag v0.3.0
    xgit commit -q --allow-empty -m "chore: e"
    xgit commit -q --allow-empty -m "docs: f"
)

GOT_SKIP_BUMP=$(cd "$SKIP_REPO" && git-cliff --config "$CLIFF_TOML" --bumped-version 2>/dev/null)
[ "$GOT_SKIP_BUMP" = "v0.3.0" ] || fail "--bumped-version with only skip-group commits: want v0.3.0 (unchanged), got $GOT_SKIP_BUMP"

echo "OK: cliff.toml rules match the golden fixture"
