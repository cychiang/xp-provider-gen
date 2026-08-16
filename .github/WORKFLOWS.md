# GitHub Actions Workflows

This directory contains GitHub Actions workflows for automating CI/CD processes.

## Workflows

### 🧹 `lint.yml` - Code Quality
**Triggers:** Push/PR to `main`, `develop`
- Runs golangci-lint, pinned to the same version as the Makefile's
  `GOLANGCILINT_VERSION` so CI and `make lint` enforce one rule set
- Validates Go code formatting with `gofmt`
- Ensures Go modules are tidy

### 🧪 `test.yml` - Testing
**Triggers:** Push/PR to `main`, `develop`
- Runs unit tests with race detection against Go 1.26.6
- Generates coverage reports, uploads to Codecov, keeps them as artifacts
- Runs an E2E smoke test (`init` + `create api`, checked with
  `scripts/assert-layout.sh`) when source files changed. This is the layout
  assertion only — the full scaffold-build-and-run suite is `make e2e-test`,
  which needs Docker and is run locally (see [testing.md](../docs/testing.md))

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
3. Renovate runs weekly on Mondays before 6 AM Pacific
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