# Development

## Requirements

- **Go** — the version in `go.mod`'s `go` directive. Renovate bumps it with the Dockerfile's
  golang tag in one PR. It is independent of the Go generated providers declare
  ([`pkg/versions/dependencies.yaml`](../pkg/versions/dependencies.yaml)'s `go_version`, which
  Renovate bumps alone); `make check-go-version` (`.github/workflows/go-version.yml`) fails CI
  if `go_version` is below what a pinned dependency requires.
- **Git**
- **golangci-lint** — installed automatically by `make lint` if missing, at the version in
  [`pkg/versions/dependencies.yaml`](../pkg/versions/dependencies.yaml)'s `golangci_lint_version`.
  That key is its only pin: the `Makefile` and `.github/workflows/lint.yml` read it, and both
  flavors' `hack/xp-provider-gen.mk.tmpl` render it. Renovate bumps that one line.
- **gosec** — security scanner
- **Docker** — without it, `make e2e-native` and `make e2e-upjet` fail before their first stage (use
  `E2E_ARGS=--mode=fast` to run only the stages that need none, ending in `VERDICT PARTIAL`);
  `make e2e-upjet` also needs network access for its non-Docker stages (downloading Terraform and a
  provider schema)

```bash
# gosec (macOS)
brew install gosec
# gosec (direct)
go install github.com/securego/gosec/v2/cmd/gosec@latest
```

## Commands

Run `make help` for the full target list. A few aren't obvious from their name alone:

| Command | Purpose |
|---------|---------|
| `make reviewable` | `mod-tidy` + `check` (fmt/vet/lint/gosec/test) + `check-consistency` — run this before pushing; mirrors what CI enforces |
| `make coverage` | Coverage report at `coverage/coverage.html` |
| `make e2e-native E2E_ARGS=--from=latest` / `make e2e-upjet E2E_ARGS=--from=latest` | Upgrades a provider scaffolded by the newest published release to this build; `--from=<tag>` names another, and `--report=<file>` writes the run's JSON report — see [testing.md](testing.md) |
| `make check-go-version` | Verifies `pkg/versions/dependencies.yaml`'s `go_version` is new enough for every pinned dependency (network) |

## Consistency gate

`make check-consistency` runs `hack/check-consistency.sh`, six checks that stop drift from
growing back: stale script names, a `Makefile` whose `.PHONY` or `make help` disagrees with its
targets, retired terminology, non-gerund error strings,
markdown links/anchors that no longer resolve. To add a check, append a `report Cn "<title>" "<violations>"` block to the script; CI
and `make reviewable` both run it.

## Typical workflow

1. Make a focused change. Keep it [KISS and DRY](../AGENTS.md#code-style).
2. `make reviewable` — fix anything it reports.
3. `make e2e-native` if you touched templates, the engine, or the automation pipeline.
4. Commit with a [conventional commit](https://www.conventionalcommits.org/) message
   (`feat:`, `fix:`, `refactor:`, `chore:`, `ci:`, `docs:`, `test:`), small and focused.
5. Open a PR. CI runs lint, tests, e2e, build, and security scans.

## Working with templates

Provider scaffolding lives in `pkg/templates/files/**` (native) and `pkg/templates/upjet/**`
(upjet), both `*.tmpl` and auto-discovered: drop a file in and it appears in every generated
provider of that flavor. The full contributor flow — path placeholders, the generated header,
the golden-test step — is in [templates.md](templates.md).

## Updating an existing provider

`xp-provider-gen update` refreshes tool-owned files in place and never touches user
code. The contract, the review workflow, and `--adopt` are documented in
[provider-guide.md](provider-guide.md#5-upgrading).

## Dependency manifest

`pkg/versions/dependencies.yaml` is the single source of truth for the framework/Kubernetes
versions a generated provider declares (plus an `upjet_dependencies` block layered on top for
the upjet flavor). It is rendered into the provider's `go.mod`, tracked by a Renovate custom
manager that groups the whole file into one PR (`provider framework dependencies` — bumping the
entries individually would break the e2e), and applied to existing providers by `update`. To
change a generated provider's dependency versions, edit this file (or let Renovate do it) —
never hardcode versions in a template.

Generated providers target the Go version in `pkg/versions/dependencies.yaml`'s `go_version`
(`pkg/versions.GoVersion`, rendered into `go.mod`) and lint with the pinned golangci-lint
(`golangci_lint_version` in the same manifest). Keep the generated `go` directive at that language version with
no `toolchain` pin — golangci-lint reads the GOROOT of the Go that runs it, so a Go above
golangci-lint's build version breaks `make reviewable` in generated projects. The generated
`hack/xp-provider-gen.mk` caps that Go by exporting `GOTOOLCHAIN`, which also covers a newer host.

## CI/CD

Pipelines are documented in [.github/WORKFLOWS.md](../.github/WORKFLOWS.md). All GitHub Actions
are pinned to commit SHAs (with a version comment) for supply-chain safety; Renovate keeps the
digests updated.
