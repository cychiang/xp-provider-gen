# AGENTS.md

Guidance for working on this repository — the `xp-provider-gen` generator itself, not a
provider scaffolded by it. Keep this file short — depth lives in [`docs/`](docs/).

## Project overview

`xp-provider-gen` is a Go CLI that scaffolds [Crossplane](https://crossplane.io) providers
using Kubebuilder v4 and crossplane-runtime v2. From `init`, `create api`, `create-test` and
`update` it generates and maintains a complete, buildable provider project.

It scaffolds two **flavors** — native and upjet — each a self-contained template
root: `pkg/templates/files/` and `pkg/templates/upjet/`. What the flavors are and
when to pick one is in the skill (see "Further reading").

## Setup commands

```bash
make build       # builds bin/xp-provider-gen
```

Prerequisites (Go version, gosec, Docker for e2e) are in
[docs/development.md](docs/development.md#requirements).

## Core concepts

- **Ownership contract** — a `// Code generated … DO NOT EDIT.` header (first 1024 bytes) marks a file tool-owned (`update` overwrites/removes it); never quote that line in your own file, even in a comment — it's a substring match and flips ownership. See [architecture.md §6](docs/architecture.md#6-ownership-contract-the-upgrade-foundation).
- **Golden ownership test** (`templates/engine/ownership_test.go`) fails on any new template until you add it to `wantOwnership` — deliberate, not a bug.
- **Prove template changes with a scaffold diff** (an old-vs-new binary's `init` output; a git worktree gets the old binary cheaply) — never trust a template read alone.
- **`PROJECT`** holds flavor, upjet settings and the provenance `version:` stamp of whichever generator last ran `update`/`init`; a corrupt or unrecognized one is refused, not guessed at.
- **`pkg/versions/dependencies.yaml`** pins a generated provider's dependencies (Renovate-tracked via a custom manager); `pkg/version/` is this CLI's own version, stamped into PROJECT — don't confuse the two.

## Testing instructions

| Command | Purpose |
|---------|---------|
| `make reviewable` | mod-tidy + fmt/vet/lint/gosec/test + check-consistency — the same checks CI enforces |
| `make e2e-native` / `make e2e-upjet` / `make e2e-upgrade` | The three e2e surfaces |
| `make help` | List all targets |

`e2e-native.sh` and `e2e-upjet.sh` skip their Docker-dependent step — not fail — when
Docker is unavailable or `E2E_SKIP_DOCKER=1` is set. See [docs/testing.md](docs/testing.md)
for what each script covers, and [docs/manual-testing.md](docs/manual-testing.md) for a
step-by-step guide to reproducing the same surfaces by hand.

## Code style

These two override cleverness. If a change makes the code harder to understand, stop and
reconsider.

- **KISS — Keep It Simple.** Prefer the smallest, most obvious solution. No speculative
  abstraction, no configuration knobs nobody asked for, no dead code kept "for later."
  A new reader should understand a file without a guided tour.
- **DRY — Don't Repeat Yourself.** A fact, string, or rule lives in exactly one place.
  Repeated string literals become named constants; repeated logic becomes a shared
  function. The linter enforces part of this (`goconst`, `dupl`).
- **Match the surrounding code.** Idiomatic Go: small files, explicit error wrapping
  (`fmt.Errorf("...: %w", err)`), table-driven tests, formatted by `gofumpt`/`gci`
  (`make lint-fix` fixes both).
- **Templates are data.** Provider scaffolding lives in `pkg/templates/files/**` (native) and `pkg/templates/upjet/**` (upjet), both auto-discovered (see [docs/templates.md](docs/templates.md)) — add a template file, don't wire it in by hand. Two "fix one place, forget another" traps: golangci-lint's version is pinned by hand in four places (Renovate bumps only `lint.yml`), and five templates are byte-identical across the two flavors.

## Security

- **Pin GitHub Actions to commit SHAs**, never floating tags. Renovate keeps the digests
  updated.
- `make reviewable` runs `gosec` — do not skip it to get a check to pass.
- Vulnerability reporting policy and scope: [SECURITY.md](SECURITY.md).

## PR instructions

- **Conventional commits**, small and focused: `feat:`, `fix:`, `refactor:`, `chore:`, `ci:`, `docs:`, `test:`. `pr-title.yml` enforces the type and an ASCII-plus-punctuation subject — squashed, a PR's title becomes the commit subject, which `cliff.toml` turns directly into a release-note line.
- Run `make reviewable` before pushing — CI runs the same checks, so green locally means green in CI.
- **Cutting a release** is manual (`release.yml`'s `workflow_dispatch`): git-cliff computes the version and notes from commit messages, GoReleaser builds and publishes; `dry_run` (default) rehearses everything without tagging or pushing. See [.github/WORKFLOWS.md](.github/WORKFLOWS.md).

## Repository layout

- `cmd/xp-provider-gen/` — CLI entry point (Kubebuilder CLI wiring)
- `pkg/plugins/crossplane/v2/` — the plugin: commands, template engine, automation, validation
- `pkg/templates/files/`, `pkg/templates/upjet/` — embedded `.tmpl` scaffolding, one root per flavor; `pkg/templates/generators/` — Go-rendered files neither tree can express (register.go, go.mod, `docs/ownership.md`)
- `pkg/versions/`, `pkg/version/` — see Core concepts above
- `scripts/` — the three e2e scripts plus their shared helpers (`lib.sh`, `assert-layout.sh`,
  `check-go-version`); see [docs/testing.md](docs/testing.md)
- `hack/envtest-provider-check/` — its own Go module; proves an upjet-generated provider's
  controllers actually start, used by `e2e-upjet.sh`
- `hack/check-consistency.sh` — the gate behind `make check-consistency`, nine checks including
  `hack/check-workflow-paths.py` and `hack/check-links.py` (markdown links and anchors)
- `hack/cliff-golden.sh` — golden test for `cliff.toml`'s git-cliff rules (version bump, note
  grouping, ASCII normalization); runs as a step of every `release.yml` run
- `.goreleaser.yaml`, `cliff.toml` — release automation config for `release.yml`: GoReleaser
  builds/archives/releases, git-cliff computes the version and release notes from commits
- `renovate.json` — dependency-update rules, including the custom manager that tracks
  `pkg/versions/dependencies.yaml`
- `.claude/skills/xp-provider-gen` → `.agents/skills/xp-provider-gen/` is a symlink, which is
  why Claude Code loads the skill automatically

## Further reading

- [docs/tutorial.md](docs/tutorial.md) — build and run a provider end-to-end locally
- [docs/provider-guide.md](docs/provider-guide.md) — what provider authors edit, and upgrading
- [docs/templates.md](docs/templates.md) — add or change generated-provider templates
- [docs/upjet-provider.md](docs/upjet-provider.md) — build a provider that wraps a Terraform provider
- [docs/architecture.md](docs/architecture.md) — how the generator is structured
- [docs/development.md](docs/development.md) — environment, tooling, and workflow
- [docs/testing.md](docs/testing.md) — unit and end-to-end testing
- [.github/WORKFLOWS.md](.github/WORKFLOWS.md) — CI/CD pipelines
- [docs/design/render-apply.md](docs/design/render-apply.md) — a proposal awaiting a maintainer decision
- [.agents/skills/xp-provider-gen/](.agents/skills/xp-provider-gen/) — using this tool to
  build a provider (author-facing; how to scaffold, configure and deploy either flavor)
- [docs/plans/](docs/plans/) — decision records: what each program decided, what it deliberately
  did not do, and the limits it accepted. Read the relevant one before revisiting a decision it
  covers: [2026-09-15-architecture-review.md](docs/plans/2026-09-15-architecture-review.md),
  [2026-09-19-consistency.md](docs/plans/2026-09-19-consistency.md),
  [2026-09-21-phase-b.md](docs/plans/2026-09-21-phase-b.md).
