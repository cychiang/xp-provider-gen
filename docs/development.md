# Development

## Requirements

- **Go 1.26+**
- **Git**
- **golangci-lint** — installed automatically by `make lint` if missing
- **gosec** — security scanner

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
| `make reviewable` | `mod-tidy` + `check` — run this before pushing |
| `make e2e-test` | Build, then run the end-to-end scaffold test |
| `make upgrade-sim` | Simulate a generator version bump against real user logic |

`make reviewable` mirrors what CI enforces. If it passes locally, CI should pass too.

## Typical workflow

1. Make a focused change. Keep it [KISS and DRY](../CLAUDE.md#core-principles).
2. `make reviewable` — fix anything it reports.
3. `make e2e-test` if you touched templates, the engine, or the automation pipeline.
4. Commit with a [conventional commit](https://www.conventionalcommits.org/) message
   (`feat:`, `fix:`, `refactor:`, `chore:`, `ci:`, `docs:`, `test:`), small and focused.
5. Open a PR. CI runs lint, tests, e2e, build, and security scans.

## Working with templates

Provider scaffolding lives in `pkg/templates/files/**/*.tmpl` and is auto-discovered:
drop a file in and it appears in every generated provider. The full contributor flow —
path placeholders, the ownership header, the golden-test step — is in
[templates.md](templates.md).

## Updating an existing provider

`xp-provider-gen update` refreshes tool-owned files in place and never touches user
code. The contract, the review workflow, and `--adopt` are documented in
[provider-guide.md](provider-guide.md#5-upgrading).

## Dependency manifest

`pkg/versions/dependencies.yaml` is the single source of truth for the framework/Kubernetes
versions a generated provider declares. It is rendered into the provider's `go.mod`, tracked by
a Renovate custom manager (so each dependency gets its own bump PR against this repo), and
applied to existing providers by `update`. To change a generated provider's dependency
versions, edit this file (or let Renovate do it) — never hardcode versions in a template.

Generated providers target **Go 1.26** (`pkg/versions.GoVersion`, rendered into `go.mod`) and
lint with golangci-lint 2.12.2 (`Makefile.tmpl`). Keep the generated `go` directive at the
language version (`1.26.0`) with no `toolchain` pin — golangci-lint reads the system GOROOT, so
pinning a toolchain patch above golangci-lint's build version breaks `make reviewable` in
generated projects.

## Coding conventions

- Idiomatic Go, formatted by `gofumpt`/`gci` (run `make lint-fix`).
- Small, focused files; explicit error wrapping with `fmt.Errorf("...: %w", err)`.
- No repeated string literals — extract a named constant (the `goconst` linter enforces this).
- Table-driven tests (see [testing.md](testing.md)).

## CI/CD

Pipelines are documented in [.github/WORKFLOWS.md](../.github/WORKFLOWS.md). All GitHub Actions
are pinned to commit SHAs (with a version comment) for supply-chain safety; Renovate keeps the
digests updated.
