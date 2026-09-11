# GitHub Actions Workflows

This directory contains GitHub Actions workflows for automating CI/CD processes.

## Workflows

### 🧹 `lint.yml` - Code Quality
**Triggers:** Push/PR to `main`, `develop`
- Runs golangci-lint at the version pinned in this workflow (`version:`). The same version
  must also be set by hand in three other places — see
  [docs/development.md](../docs/development.md#requirements) — since Renovate bumps only
  this workflow's pin
- Validates Go code formatting with `gofmt`
- Ensures Go modules are tidy

### 🧪 `test.yml` - Testing
**Triggers:** Push/PR to `main`, `develop`
- Runs unit tests with race detection against Go 1.26.6
- Generates coverage reports, uploads to Codecov, keeps them as artifacts
- When source files changed, runs the real native-flavor e2e
  (`scripts/e2e-test.sh`): init, two APIs, CRD/example generation, the
  `update` / `update --adopt` / `create-test` lifecycle, ownership-header
  checks. `E2E_SKIP_DOCKER=1` is set explicitly so its Step E — the generated
  provider's own uptest+chainsaw suite against a kind cluster, minutes of
  Docker/cluster time — is skipped deliberately rather than by accident (the
  runner does have a docker daemon). Run Step E locally with
  `./scripts/e2e-test.sh` (no env var) or `make e2e-test` — see
  [testing.md](../docs/testing.md)

### 🧱 `e2e-upjet.yml` - Upjet Flavor E2E
**Triggers:** Daily schedule; PRs touching `pkg/templates/upjet/**`,
`scripts/e2e-upjet.sh`, or `hack/envtest-provider-check/**`
- Runs `scripts/e2e-upjet.sh`: scaffold an upjet provider wrapping
  hashicorp/kubernetes, configure a resource, run the real upjet generation
  pipeline (downloads Terraform, reads the provider schema, scrapes docs),
  build, then prove the generated provider actually **runs** — binary
  `--help`, scheme registration against an unreachable API server, and (when
  envtest/kubebuilder-tools assets can be resolved; skipped with a warning
  otherwise, never failing the build) controller registration against a real
  ephemeral API server. Not run on every PR because it needs network and
  takes several minutes

### 🔨 `build.yml` - Build Binaries
**Triggers:** Push/PR to `main`, `develop`
- Builds cross-platform binaries:
  - Linux (amd64, arm64)
  - macOS (amd64, arm64)
  - Windows (amd64)
- Creates checksums for all binaries
- Builds the repository `Dockerfile` (the same image `release.yml` publishes) and
  smoke-tests it
- Uploads build artifacts

### 🚀 `release.yml` - Release Management
**Triggers:** Git tags (`v*`)
- Runs full test suite
- Builds release binaries for all platforms
- Creates release archives with checksums
- Generates changelog from git commits
- Creates GitHub release with assets
- Builds and pushes Docker images to GitHub Container Registry
- Supports semantic versioning and pre-releases

### 🔒 `ci.yml` - Security & Additional Checks
**Triggers:** Push/PR to `main`, `develop`
- Runs Gosec security scanner
- Performs Trivy vulnerability scanning
- Uploads security findings to GitHub Security tab

## Docker Images

Release workflow publishes multi-platform Docker images to:
- `ghcr.io/cychiang/xp-provider-gen:latest`
- `ghcr.io/cychiang/xp-provider-gen:v1.2.3`
- `ghcr.io/cychiang/xp-provider-gen:v1.2`
- `ghcr.io/cychiang/xp-provider-gen:v1`

## Dependencies

**Automated dependency updates** via [Renovate Bot](https://docs.renovatebot.com/):
- Go modules (grouped by type: Kubernetes, Crossplane, testing)
- GitHub Actions — pinned to commit SHAs (`helpers:pinGitHubActionDigests`) and updated by Renovate
- Docker base images (with digest pinning)
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
# Re-run failed workflow
gh workflow run build.yml

# Run workflow on specific branch
gh workflow run test.yml --ref feature-branch
```

### Using Released Binaries
```bash
# Download from GitHub releases
curl -L https://github.com/cychiang/xp-provider-gen/releases/download/v1.2.3/xp-provider-gen_1.2.3_linux_amd64.tar.gz

# Or use Docker
docker run --rm ghcr.io/cychiang/xp-provider-gen:v1.2.3 --help
```

## Development Workflow

1. **Feature development** - Lint and test workflows run on PRs
2. **Merge to main** - All workflows run, binaries are built
3. **Create release tag** - Release workflow creates GitHub release with assets
4. **Security monitoring** - Renovate keeps dependencies updated (see above)

## Required Secrets

The workflows use these GitHub secrets:
- `GITHUB_TOKEN` (automatically provided)
- No additional secrets required

## Workflow Status

View workflow status at: https://github.com/cychiang/xp-provider-gen/actions