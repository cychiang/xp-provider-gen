# GitHub Actions Workflows

This directory contains GitHub Actions workflows for automating CI/CD processes. Each
workflow file's own comments have the exact trigger paths and step-by-step detail; this page
is the map, not a transcript of them.

## Workflows

| Workflow | Triggers | What it does |
|---|---|---|
| 🧹 `lint.yml` — Lint | push/PR to `main`, `develop` | golangci-lint at the version pinned here (Renovate bumps only this file — the same version must be set by hand in three other places, see [docs/development.md](../docs/development.md#requirements)), a `gofmt` check, a `go mod tidy` check, and `make check-workflow-paths` |
| 🧪 `test.yml` — Test | push/PR to `main`, `develop` | Unit tests with coverage (Codecov + artifact); when source files changed, the native-flavor e2e (`scripts/e2e-native.sh`) with its Docker-dependent step 12 skipped (`E2E_SKIP_DOCKER=1`) — step 12 itself runs in `e2e-native-full.yml` |
| 🧭 `e2e-native-full.yml` — E2E Native (full) & Upgrade | daily; `workflow_dispatch`; PRs touching the generator/template/version surfaces this exercises | The same native e2e with step 12 included (no `E2E_SKIP_DOCKER`) — the generated provider's own uptest+chainsaw suite against a real kind cluster — plus `make e2e-upgrade` |
| 🧱 `e2e-upjet.yml` — E2E Upjet | daily; `workflow_dispatch`; PRs touching the upjet flavor's templates/plugin/version surfaces | `scripts/e2e-upjet.sh`: the real upjet generation pipeline (Terraform download, schema read, docs scrape), build, and — with Docker (`E2E_SKIP_DOCKER=0`, explicit) — the generated provider's own e2e |
| ⚙️ `go-version.yml` — Go version alignment | push/PR to `main`, `develop` | `make check-go-version`: asserts go.mod and the Dockerfile agree with `pkg/versions/dependencies.yaml`'s `go_version`, and that it's new enough for every pinned dependency |
| 🔨 `build.yml` — Build | push/PR to `main`, `develop` | Cross-platform binaries + checksums, builds and smoke-tests the repository `Dockerfile`, uploads artifacts |
| 🚀 `release.yml` — Release | git tags (`v*`) | Full test suite, release binaries, changelog, GitHub release, Docker image push — **not yet exercised: no tags have been cut** (see [SECURITY.md](../SECURITY.md)) |
| 🔒 `ci.yml` — Security & Additional Checks | push/PR to `main`, `develop` | gosec and Trivy scans; results upload to the repository's Security tab |

## Docker Images

Once a release ships, `release.yml` publishes multi-platform Docker images to
`ghcr.io/cychiang/xp-provider-gen`. No tags exist yet, so no images have been published.

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
```bash
# Create and push a tag
git tag v1.2.3
git push origin v1.2.3

# Release workflow will automatically:
# 1. Run tests
# 2. Build binaries for all platforms
# 3. Create GitHub release
# 4. Push Docker images
```

### Manual Workflow Triggers
```bash
# Re-run the native e2e (full, with step 12) + upgrade E2E test on demand
gh workflow run e2e-native-full.yml

# Re-run the upjet e2e on a specific branch
gh workflow run e2e-upjet.yml --ref feature-branch
```

## Development Workflow

1. **Feature development** — `lint`, `test`, and `go-version` run on every PR; `e2e-native-full`
   and `e2e-upjet` run only on PRs that touch the surfaces they cover
2. **Merge to main** — the same push-triggered workflows run again; `build.yml` builds binaries
3. **Create a release tag** — `release.yml` creates a GitHub release with assets (not yet
   exercised — see above)
4. **Security monitoring** — Renovate keeps dependencies updated (see above)

## Required Secrets

The workflows use these GitHub secrets:
- `GITHUB_TOKEN` (automatically provided)
- No additional secrets required

## Workflow Status

View workflow status at: https://github.com/cychiang/xp-provider-gen/actions
