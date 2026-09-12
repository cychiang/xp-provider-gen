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

## Testing instructions

| Command | Purpose |
|---------|---------|
| `make test` | Unit tests with the race detector |
| `make lint` | golangci-lint (config in `.golangci.yml`) |
| `make e2e-test` | Native flavor: scaffold → build → the generated provider's own e2e |
| `make e2e-upjet` | Upjet flavor: scaffold, generate with upjet, build, run (network) |
| `make upgrade-sim` | Simulate a generator bump against real user logic |
| `make reviewable` | mod-tidy + fmt/vet/lint/gosec/test — the same checks CI enforces |
| `make help` | List all targets |

Both e2e scripts skip their Docker-dependent stage — not fail — when Docker is
unavailable or `E2E_SKIP_DOCKER=1` is set. See [docs/testing.md](docs/testing.md)
for what each script covers.

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
  (`fmt.Errorf("...: %w", err)`), table-driven tests.
- **Templates are data.** Provider scaffolding lives in `pkg/templates/files/**` (native) and
  `pkg/templates/upjet/**` (upjet), both auto-discovered (see [docs/templates.md](docs/templates.md)).
  Add a template file — do not wire it in by hand.

## Security

- **Pin GitHub Actions to commit SHAs**, never floating tags. Renovate keeps the digests
  updated.
- `make reviewable` runs `gosec` — do not skip it to get a check to pass.

## PR instructions

- **Conventional commits**, small and focused: `feat:`, `fix:`, `refactor:`, `chore:`,
  `ci:`, `docs:`, `test:`.
- Run `make reviewable` before pushing — CI runs the same checks, so green locally means
  green in CI.

## Repository layout

- `cmd/xp-provider-gen/` — CLI entry point (Kubebuilder CLI wiring)
- `pkg/plugins/crossplane/v2/` — the plugin: commands, template engine, automation, validation
- `pkg/templates/files/`, `pkg/templates/upjet/` — embedded `.tmpl` scaffolding, one root per flavor
- `scripts/` — `e2e-test.sh` (native e2e), `e2e-upjet.sh` (upjet e2e), `upgrade-sim.sh`
  (upgrade simulation), `assert-layout.sh` (generated-layout assertions, shared with CI)
- `hack/envtest-provider-check/` — its own Go module; proves an upjet-generated provider's
  controllers actually start, used by `e2e-upjet.sh`

## Further reading

- [docs/tutorial.md](docs/tutorial.md) — build and run a provider end-to-end locally
- [docs/provider-guide.md](docs/provider-guide.md) — what provider authors edit, and upgrading
- [docs/templates.md](docs/templates.md) — add or change generated-provider templates
- [docs/upjet-provider.md](docs/upjet-provider.md) — build a provider that wraps a Terraform provider
- [docs/architecture.md](docs/architecture.md) — how the generator is structured
- [docs/development.md](docs/development.md) — environment, tooling, and workflow
- [docs/testing.md](docs/testing.md) — unit and end-to-end testing
- [.github/WORKFLOWS.md](.github/WORKFLOWS.md) — CI/CD pipelines
- [.agents/skills/xp-provider-gen/](.agents/skills/xp-provider-gen/) — using this tool to
  build a provider (author-facing; how to scaffold, configure and deploy either flavor)
