# GitHub Actions Workflows

This directory contains GitHub Actions workflows for automating CI/CD processes. Each
workflow file's own comments have the exact trigger paths and step-by-step detail; this page
is the map, not a transcript of them.

## Workflows

| Workflow | Triggers | What it does |
|---|---|---|
| 🧹 `lint.yml` — Lint | push/PR to `main`, `develop` | golangci-lint at the version pinned here (Renovate bumps only this file — the same version must be set by hand in three other places, see [docs/development.md](../docs/development.md#requirements)), a `gofmt` check, a `go mod tidy` check, and `make check-consistency` |
| 🧪 `test.yml` — Test | push/PR to `main`, `develop` | Unit tests with coverage (Codecov + artifact); when source files changed, the native-flavor e2e (`scripts/e2e-native.sh`) with its Docker-dependent step 12 skipped (`E2E_SKIP_DOCKER=1`) — step 12 itself runs in `e2e-native-full.yml` |
| 🧭 `e2e-native-full.yml` — E2E Native (full) & Upgrade | daily; `workflow_dispatch`; PRs touching the generator/template/version surfaces this exercises | The same native e2e with step 12 included (no `E2E_SKIP_DOCKER`) — the generated provider's own uptest+chainsaw suite against a real kind cluster — plus `make e2e-upgrade` |
| 🧱 `e2e-upjet.yml` — E2E Upjet | daily; `workflow_dispatch`; PRs touching the upjet flavor's templates/plugin/version surfaces | `scripts/e2e-upjet.sh`: the real upjet generation pipeline (Terraform download, schema read, docs scrape), build, and — with Docker (`E2E_SKIP_DOCKER=0`, explicit) — the generated provider's own e2e |
| ⚙️ `go-version.yml` — Go version alignment | push/PR to `main`, `develop` | `make check-go-version`: asserts go.mod and the Dockerfile agree with `pkg/versions/dependencies.yaml`'s `go_version`, and that it's new enough for every pinned dependency |
| 🔨 `build.yml` — Build | push/PR to `main`, `develop` | Cross-platform binaries + checksums, builds and smoke-tests the repository `Dockerfile`, uploads artifacts |
| 🚀 `release.yml` — Release | `workflow_dispatch` (**Run workflow** button; `dry_run` input, on by default) | Full test suite, [git-cliff](https://git-cliff.org) computes the next version and grouped release notes from commit messages, [GoReleaser](https://goreleaser.com) builds/archives/checksums and creates the GitHub Release, then the docker job pushes the multi-platform image. `dry_run=true` (the default) rehearses every step — including `hack/cliff-golden.sh`'s check of the git-cliff rules — without tagging, releasing or pushing anything |
| 🔖 `pr-title.yml` — PR title | PR opened/edited/synchronize/reopened | [`amannn/action-semantic-pull-request`](https://github.com/amannn/action-semantic-pull-request) checks the PR title (and, for a single-commit PR, that its commit subject matches too) against the conventional-commit types and an ASCII-plus-typographic-punctuation pattern — the same subjects `cliff.toml` turns into release notes |
| 🔒 `security.yml` — Security | push/PR to `main`, `develop` | gosec and Trivy scans; results upload to the repository's Security tab |

## Docker Images

A real (non-dry-run) run of `release.yml` publishes multi-platform (`linux/amd64`,
`linux/arm64`) Docker images to `ghcr.io/cychiang/xp-provider-gen`, tagged with the
release version and `<major>.<minor>`.

## Dependencies

**Automated dependency updates** via [Renovate Bot](https://docs.renovatebot.com/):
- Go modules (grouped by type: Kubernetes, Crossplane, testing)
- GitHub Actions — pinned to commit SHAs (`helpers:pinGitHubActionDigests`) and updated by Renovate
- **Generated-provider dependency versions** — a Renovate **custom (regex) manager** tracks
  `pkg/versions/dependencies.yaml` (the manifest rendered into generated `go.mod`), so each
  generated-provider dependency gets its own bump PR against this repo
- Security vulnerability alerts (high priority)
- Dependency Dashboard for overview

### Renovate Setup
1. Install [Renovate GitHub App](https://github.com/apps/renovate)
2. Configure via `renovate.json` (already included)
3. Renovate runs weekly, before 6 AM UTC on Mondays
4. Creates grouped PRs for related dependencies
5. Provides detailed release notes and changelogs

## Usage Examples

### Creating a Release

Actions → **Release** → **Run workflow**. Leave `dry_run` checked (the default) to
rehearse the whole pipeline — tests, `hack/cliff-golden.sh`, a GoReleaser `--snapshot`
build of all five archives, a two-platform (unpushed) Docker build — without tagging,
releasing or pushing anything. Uncheck it to cut a real release: the workflow computes
the next version and release notes from commit messages with git-cliff, tags that
commit, builds and publishes with GoReleaser, then builds and pushes the Docker image.
The version is never typed in by hand.

```bash
# Equivalent from the CLI
gh workflow run release.yml -f dry_run=false
```

Release assets carry one `<project>_<version>_checksums.txt` (GoReleaser's default) instead
of a `.sha256` file per archive. `hack/cliff-golden.sh` pins the exact behavior of `cliff.toml`'s
version-bump and note-grouping rules against a fixture; it runs as a step of every `release.yml`
run — dry-run and real alike — and can also be run locally once `git-cliff` is on `PATH`.

### Manual Workflow Triggers
```bash
# Re-run the native e2e (full, with step 12) + upgrade E2E test on demand
gh workflow run e2e-native-full.yml

# Re-run the upjet e2e on a specific branch
gh workflow run e2e-upjet.yml --ref feature-branch
```

## Development Workflow

1. **Feature development** — `lint`, `test`, `pr-title`, and `go-version` run on every PR;
   `e2e-native-full` and `e2e-upjet` run only on PRs that touch the surfaces they cover
2. **Merge to main** — the same push-triggered workflows run again; `build.yml` builds binaries
3. **Cut a release** — run `release.yml` from the Actions tab (see "Creating a Release" above)
4. **Security monitoring** — Renovate keeps dependencies updated (see above)

## Required Secrets

The workflows use these GitHub secrets:
- `GITHUB_TOKEN` (automatically provided)
- No additional secrets required

## Workflow Status

View workflow status at: https://github.com/cychiang/xp-provider-gen/actions
