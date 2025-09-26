# GitHub Actions Workflows

This directory contains GitHub Actions workflows for automating CI/CD processes.

## Workflows

### 🧹 `lint.yml` - Code Quality
**Triggers:** Push/PR to `main`, `develop`
- Runs golangci-lint for code quality checks
- Validates Go code formatting with `gofmt`
- Ensures Go modules are tidy
- Caches Go modules for faster builds

### 🧪 `test.yml` - Testing
**Triggers:** Push/PR to `main`, `develop`
- Runs unit tests with race detection
- Tests against multiple Go versions (1.24.5, 1.24.7)
- Generates coverage reports
- Uploads coverage to Codecov
- Includes E2E workflow testing
- Creates coverage artifacts

### 🔨 `build.yml` - Build Binaries
**Triggers:** Push/PR to `main`, `develop`
- Builds cross-platform binaries:
  - Linux (amd64, arm64)
  - macOS (amd64, arm64)
  - Windows (amd64)
- Creates checksums for all binaries
- Builds Docker image
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
- GitHub Actions (minor/patch + major updates)
- Docker base images (with digest pinning)
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
4. **Security monitoring** - Dependabot keeps dependencies updated

## Required Secrets

The workflows use these GitHub secrets:
- `GITHUB_TOKEN` (automatically provided)
- No additional secrets required

## Workflow Status

View workflow status at: https://github.com/cychiang/xp-provider-gen/actions