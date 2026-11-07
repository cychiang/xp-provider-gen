# GitHub Actions Workflows

This directory contains GitHub Actions workflows for automating CI/CD processes. Each
workflow file's own comments have the exact trigger paths and step-by-step detail; this page
is the map, not a transcript of them.

## Workflows

| Workflow | Triggers | What it does |
|---|---|---|
| 🧹 `lint.yml` — Lint | push/PR to `main`, `develop` | golangci-lint at the version pinned here (Renovate bumps it together with the three other places that pin it, and `make check-consistency` fails if they disagree, see [docs/development.md](../docs/development.md#requirements)), a `gofmt` check, a `go mod tidy` check, and `make check-consistency` |
| 🧪 `test.yml` — Test | push/PR to `main`, `develop` | Unit tests with coverage (Codecov + artifact); when source files changed, the native-flavor e2e (`test/e2e`) in fast mode, which ends in `VERDICT PARTIAL` naming the cluster stages it did not run — those run in `e2e-native-full.yml`; per-stage logs upload as an artifact |
| 🧭 `e2e-native-full.yml` — E2E Native (full) & Upgrade | weekly; `workflow_dispatch`; PRs touching the generator/template/version surfaces this exercises | The same native e2e in full mode (every stage; it exits 1 before the first stage if Docker is missing) — the generated provider's own uptest+chainsaw suite and a real external object created, changed and deleted on a kind cluster, ending in `VERDICT COMPLETE`; per-stage logs and `report.json` upload as an artifact — plus one upgrade job, on the schedule and `workflow_dispatch` only, never on a PR (`make e2e-native E2E_ARGS=--from=latest`): the newest published release's own binary scaffolds a provider, then this build updates it, and its `report.json` carries the stage times and the upgrade counts |
| 🧱 `e2e-upjet.yml` — E2E Upjet | weekly; `workflow_dispatch`; PRs touching the upjet flavor's templates/plugin/version surfaces or `test/e2e` | The upjet e2e (`test/e2e`) in full mode (it exits 1 before the first stage if Docker is missing): the real upjet generation pipeline (Terraform download, schema read, docs scrape), build, the Terraform provider bump, the generated provider's own uptest+chainsaw suite, and a real ConfigMap created, changed, drift-corrected and deleted through the deployed provider on a kind cluster, ending in `VERDICT COMPLETE` — plus one upgrade job, on the schedule and `workflow_dispatch` only, never on a PR (`make e2e-upjet E2E_ARGS=--from=latest`); both jobs upload `report.json` with the logs |
| ⚙️ `go-version.yml` — Go version alignment | push/PR to `main`, `develop` | `make check-go-version`: asserts go.mod and the Dockerfile agree with `pkg/versions/dependencies.yaml`'s `go_version`, and that it's new enough for every pinned dependency |
| 🧰 `init-stable-go.yml` — Scaffold on stable Go | weekly; `workflow_dispatch`; PRs touching the templates, plugin or version surfaces | Installs the newest stable Go, builds the generator, runs native `init` and `create api`, then an upjet `init`, `create api`, `make generate` and `make reviewable`. Fails when a generated provider's `GOTOOLCHAIN` cap in `hack/xp-provider-gen.mk` stops working, which every job on the pinned Go would miss |
| 🔨 `build.yml` — Build | push/PR to `main`, `develop` | Cross-platform binaries + checksums, builds and smoke-tests the repository `Dockerfile`, uploads artifacts |
| 🚀 `release.yml` — Release | `workflow_dispatch` (**Run workflow** button; `dry_run` input, on by default) | `go test ./...` as a final sanity check (not the race-detector suite `test.yml` already ran on the PR), [git-cliff](https://git-cliff.org) computes the next version and grouped release notes from commit messages, [GoReleaser](https://goreleaser.com) builds/archives/checksums and creates the GitHub Release, then the docker job pushes the multi-platform image. `dry_run=true` (the default) rehearses every step — including `hack/cliff-golden.sh`'s check of the git-cliff rules — without tagging, releasing or pushing anything |
| 🔖 `pr-title.yml` — PR title | PR opened/edited/synchronize/reopened | [`amannn/action-semantic-pull-request`](https://github.com/amannn/action-semantic-pull-request) checks the PR title (and, for a single-commit PR, that its commit subject matches too) against the conventional-commit types and an ASCII-plus-typographic-punctuation pattern — the same subjects `cliff.toml` turns into release notes |
| 🔒 `security.yml` — Security | push/PR to `main`, `develop` | gosec and Trivy scans; results upload to the repository's Security tab |

## Docker Images

A real (non-dry-run) run of `release.yml` publishes multi-platform (`linux/amd64`,
`linux/arm64`) Docker images to `ghcr.io/cychiang/xp-provider-gen`, tagged with the
release version and `<major>.<minor>`.

## Dependencies

**Automated dependency updates** via [Renovate Bot](https://docs.renovatebot.com/) (schedule and
grouping rules live in `renovate.json`, the source of truth):
- Go modules — grouped where they must move together (`kubernetes-api`, `kubernetes-tooling`,
  `crossplane packages`, `provider framework dependencies`); the testing libraries
  (testify/ginkgo/gomega) automerge patches individually, ungrouped
- GitHub Actions — pinned to commit SHAs (`helpers:pinGitHubActionDigests`) and updated by Renovate
- **Generated-provider dependency versions** — a Renovate **custom (regex) manager** tracks
  `pkg/versions/dependencies.yaml` (the manifest rendered into generated `go.mod`) and groups
  them into one PR (`provider framework dependencies`) — crossplane-runtime, crossplane/apis,
  controller-runtime and k8s.io/* must bump together or the e2e breaks
- Security vulnerability alerts (high priority)
- Dependency Dashboard for overview

## Usage Examples

### Creating a Release

Actions → **Release** → **Run workflow**. Leave `dry_run` checked (the default) to
rehearse the whole pipeline — tests, `hack/cliff-golden.sh`, a GoReleaser `--snapshot`
build of all five archives, a two-platform (unpushed) Docker build — without tagging,
releasing or pushing anything. Uncheck it to cut a real release: the workflow computes
the next version and release notes from commit messages with git-cliff, tags that
commit, builds and publishes with GoReleaser, then builds and pushes the Docker image.
The version is never typed in by hand.
The run list labels each run "Release (dry run)" or "Release (publish)"; only a
"(publish)" run creates the tag and the Release.

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
# Re-run the native e2e (full mode) + upgrade E2E test on demand
gh workflow run e2e-native-full.yml

# Re-run the upjet e2e on a specific branch
gh workflow run e2e-upjet.yml --ref feature-branch
```

## Development Workflow

1. **Every PR** — `lint`, `test`, `build`, `security`, `pr-title`, and `go-version` all run;
   `e2e-native-full`, `e2e-upjet` and `init-stable-go` run only on PRs that touch the surfaces they cover
2. **Merge to main** — the same push-triggered workflows run again
3. **Cut a release** — run `release.yml` from the Actions tab (see "Creating a Release" above).
   No list of releases is kept: the upgrade jobs resolve `--from=latest` through `gh`, which
   ignores pre-releases and drafts. The one hand-kept table is `toolchains` in
   `test/e2e/plan.go`. A release whose scaffold carries the `GOTOOLCHAIN` pin in
   `hack/xp-provider-gen.mk` needs no entry, and the head templates carry it. Add the tag only
   if the new release's `init` needs a Go the host may lack.
4. **Security monitoring** — Renovate keeps dependencies updated (see above)
