# CLAUDE.md

Guidance for working in this repository. Keep this file short — depth lives in [`docs/`](docs/).

## What this is

`xp-provider-gen` is a Go CLI that scaffolds [Crossplane](https://crossplane.io) providers
using Kubebuilder v4 and crossplane-runtime v2. From `init`, `create api`, `create-test` and
`update` it generates and maintains a complete, buildable provider project.

It scaffolds two **flavors**, chosen once at `init` and recorded in PROJECT:
**native** (you write the reconcile logic behind the seam contract) and **upjet**
(`init --upjet`: types and controllers are generated from a Terraform provider's
schema, and you write configuration). Each flavor is a self-contained template
root — `pkg/templates/files/` and `pkg/templates/upjet/`.

## Core principles

These two override cleverness. If a change makes the code harder to understand, stop and reconsider.

- **KISS — Keep It Simple.** Prefer the smallest, most obvious solution. No speculative
  abstraction, no configuration knobs nobody asked for, no dead code kept "for later."
  A new reader should understand a file without a guided tour.
- **DRY — Don't Repeat Yourself.** A fact, string, or rule lives in exactly one place.
  Repeated string literals become named constants; repeated logic becomes a shared
  function. The linter enforces part of this (`goconst`, `dupl`).

Everything below serves these two.

## Working agreements

- **Verify before claiming done.** Run `make reviewable` (fmt, vet, lint, gosec, tidy,
  test). CI runs the same checks — green locally means green in CI.
- **Match the surrounding code.** Idiomatic Go: small files, explicit error wrapping
  (`fmt.Errorf("...: %w", err)`), table-driven tests.
- **Templates are data.** Provider scaffolding lives in `pkg/templates/files/**/*.tmpl`
  and is auto-discovered (see [docs/templates.md](docs/templates.md)). Add a template
  file — do not wire it in by hand.
- **Conventional commits**, small and focused: `feat:`, `fix:`, `refactor:`, `chore:`,
  `ci:`, `docs:`, `test:`.
- **Pin GitHub Actions to commit SHAs**, never floating tags (supply-chain safety).
  Renovate keeps the digests updated.

## Essential commands

| Command | Purpose |
|---------|---------|
| `make build` | Build the `bin/xp-provider-gen` binary |
| `make test` | Unit tests with the race detector |
| `make lint` | golangci-lint (config in `.golangci.yml`) |
| `make e2e-test` | Full scaffold → build → the generated provider's own e2e |
| `make upgrade-sim` | Simulate a generator bump against real user logic |
| `make e2e-upjet` | Upjet flavor: scaffold, generate with upjet, build (network) |
| `make reviewable` | Everything CI runs; do this before pushing |
| `make help` | List all targets |

## Where things are

- `cmd/xp-provider-gen/` — CLI entry point (Kubebuilder CLI wiring)
- `pkg/plugins/crossplane/v2/` — the plugin: commands, template engine, automation, validation
- `pkg/templates/files/` — embedded `.tmpl` scaffolding for generated providers
- `scripts/` — `e2e-test.sh` (end-to-end), `upgrade-sim.sh` (upgrade simulation),
  `assert-layout.sh` (generated-layout assertions, shared with CI)

## Deeper docs

- [docs/tutorial.md](docs/tutorial.md) — build and run a provider end-to-end locally
- [docs/provider-guide.md](docs/provider-guide.md) — what provider authors edit, and upgrading
- [docs/templates.md](docs/templates.md) — add or change generated-provider templates
- [docs/upjet-provider.md](docs/upjet-provider.md) — build a provider that wraps a Terraform provider
- [docs/architecture.md](docs/architecture.md) — how the generator is structured
- [docs/development.md](docs/development.md) — environment, tooling, and workflow
- [docs/testing.md](docs/testing.md) — unit and end-to-end testing
- [.github/WORKFLOWS.md](.github/WORKFLOWS.md) — CI/CD pipelines
