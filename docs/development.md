# Development

## Requirements

- **Go** — the version in [`pkg/versions/dependencies.yaml`](../pkg/versions/dependencies.yaml)'s
  `go_version`; `make check-go-version` (`.github/workflows/go-version.yml`) fails CI if go.mod
  or the Dockerfile fall out of step with it.
- **Git**
- **golangci-lint** — installed automatically by `make lint` if missing, at the version
  pinned in `GOLANGCILINT_VERSION`. The same version must be set by hand in four places:
  `Makefile`, `.github/workflows/lint.yml`, `pkg/templates/files/hack/xp-provider-gen.mk.tmpl`
  and `pkg/templates/upjet/hack/xp-provider-gen.mk.tmpl`. Renovate currently bumps only `lint.yml`; align
  the other three in the same PR.
- **gosec** — security scanner
- **Docker** — without it, both `make e2e-native` and `make e2e-upjet` skip their
  Docker-dependent stage rather than failing; `make e2e-upjet` also needs network access
  for its non-Docker stages (downloading Terraform and a provider schema)

```bash
# gosec (macOS)
brew install gosec
# gosec (direct)
go install github.com/securego/gosec/v2/cmd/gosec@latest
```

## Commands

| Command | Purpose |
|---------|---------|
| `make build` | Build `bin/xp-provider-gen` |
| `make test` | Unit tests with the race detector |
| `make coverage` | Coverage report at `coverage/coverage.html` |
| `make fmt` / `make vet` | Format / vet |
| `make lint` / `make lint-fix` | golangci-lint (config: `.golangci.yml`) |
| `make gosec` | Security scan |
| `make mod-tidy` / `make mod-verify` | Module hygiene |
| `make check` | fmt + vet + lint + gosec + test |
| `make reviewable` | `mod-tidy` + `check` + `check-consistency` — run this before pushing |
| `make e2e-native` | Build, then run the end-to-end scaffold test |
| `make e2e-upgrade` | Run a generator version bump against real user logic (native flavor) |
| `make e2e-upjet` | Scaffold an upjet provider and run the real upjet pipeline (network) |
| `make check-go-version` | Verify go.mod/Dockerfile agree with `pkg/versions/dependencies.yaml`'s `go_version` (network) |
| `make check-consistency` | Assert the repo's naming, skeleton and terminology conventions (part of `reviewable`) |

`make reviewable` mirrors what CI enforces. If it passes locally, CI should pass too.

## Consistency gate

`make check-consistency` runs `hack/check-consistency.sh`, eight checks that stop drift from
growing back: stale script names, a `Makefile` whose `.PHONY` or `make help` disagrees with its
targets, e2e scripts that diverge from one skeleton or `/tmp` prefix, retired terminology,
non-gerund error strings, and workflow `paths:` filters. To add a check, append a
`report Cn "<title>" "<violations>"` block to the script; CI and `make reviewable` both run it.

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
(`hack/xp-provider-gen.mk.tmpl`). Keep the generated `go` directive at that language version with
no `toolchain` pin — golangci-lint reads the system GOROOT, so pinning a toolchain patch above
golangci-lint's build version breaks `make reviewable` in generated projects.

## Coding conventions

- Idiomatic Go, formatted by `gofumpt`/`gci` (run `make lint-fix`).
- Small, focused files; explicit error wrapping with `fmt.Errorf("...: %w", err)`.
- No repeated string literals — extract a named constant (the `goconst` linter enforces this).
- Table-driven tests (see [testing.md](testing.md)).

## CI/CD

Pipelines are documented in [.github/WORKFLOWS.md](../.github/WORKFLOWS.md). All GitHub Actions
are pinned to commit SHAs (with a version comment) for supply-chain safety; Renovate keeps the
digests updated.
