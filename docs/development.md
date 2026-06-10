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

`make reviewable` mirrors what CI enforces. If it passes locally, CI should pass too.

## Typical workflow

1. Make a focused change. Keep it [KISS and DRY](../CLAUDE.md#core-principles).
2. `make reviewable` — fix anything it reports.
3. `make e2e-test` if you touched templates, the engine, or the automation pipeline.
4. Commit with a [conventional commit](https://www.conventionalcommits.org/) message
   (`feat:`, `fix:`, `refactor:`, `chore:`, `ci:`, `docs:`, `test:`), small and focused.
5. Open a PR. CI runs lint, tests, e2e, build, and security scans.

## Working with templates

Provider scaffolding lives under `pkg/templates/files/` as `.tmpl` files embedded via
`go:embed`. They are **auto-discovered** — see [architecture.md](architecture.md#4-template-engine).

To add or change generated output:

1. Add or edit a `.tmpl` file under `pkg/templates/files/`. Path conventions:
   - Static / per-project files: their real path (e.g. `cmd/provider/main.go.tmpl`).
   - Per-API files: use the `GROUP`/`VERSION`/`KIND` path placeholders
     (e.g. `apis/GROUP/VERSION/KIND_types.go.tmpl`).
2. Use Kubebuilder machinery template actions inside the file (`{{ .Resource.Kind }}`,
   `{{ .Boilerplate }}`, …) and the path replacements (`{GROUP}`, `{VERSION}`, `{KIND}`,
   `{IMAGENAME}`).
3. `make build && make e2e-test` to verify the generated project still builds.

You do **not** register templates in Go code — discovery handles it.

## Coding conventions

- Idiomatic Go, formatted by `gofumpt`/`gci` (run `make lint-fix`).
- Small, focused files; explicit error wrapping with `fmt.Errorf("...: %w", err)`.
- No repeated string literals — extract a named constant (the `goconst` linter enforces this).
- Table-driven tests (see [testing.md](testing.md)).

## CI/CD

Pipelines are documented in [.github/WORKFLOWS.md](../.github/WORKFLOWS.md). All GitHub Actions
are pinned to commit SHAs (with a version comment) for supply-chain safety; Renovate keeps the
digests updated.
