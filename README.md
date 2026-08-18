# Crossplane Provider Generator

A CLI tool for scaffolding Crossplane providers with Kubebuilder v4 and crossplane-runtime v2.

## Key Features

- **♻️ Upgradable core** — `update` refreshes a provider's tool-owned plumbing (wiring,
  registration, `main.go`, framework deps) without touching your business logic, and
  `update --adopt` retrofits providers made before the contract existed
- **📦 Modular layout** — the framework plumbing is tool-owned; you write six named seams
  (`NewClient`, `Client`, `Flags`, `Configure`, `NewExternal`, `ReconcilerOptions`), split
  per kind into your `external.go` and generated `wiring.go`
- **🔒 File-ownership contract** — a `// Code generated … DO NOT EDIT.` header decides what
  `update` may rewrite; enforced by a golden test and published as a generated
  `docs/ownership.md` inside every provider
- **🚀 Safe-Start support** — Crossplane v2.0+ selective resource activation, plus
  Management Policies, ChangeLogs and metrics
- **🧪 E2E out of the box** — every scaffold ships uptest lifecycle tests and chainsaw
  behavior tests; `create-test` adds more
- **📝 Template auto-discovery** — drop a `.tmpl` in and it appears in every provider;
  registration files are generated deterministically, never parsed and merged
- **📌 Tracked dependencies** — one version manifest, bumped by Renovate, applied to
  existing providers by `update`

## Quick Start

Prefer a guided walkthrough? [The tutorial](docs/tutorial.md) builds a complete
provider and runs it on a local kind cluster in about thirty minutes.

### Build the Generator

```bash
git clone git@github.com:cychiang/xp-provider-gen.git
cd xp-provider-gen
make build
```

### Generate a Provider

```bash
# Initialize provider project (always use a separate directory)
mkdir my-provider && cd my-provider
./bin/xp-provider-gen init --domain=example.com --repo=github.com/example/provider-awesome

# Add managed resources
./bin/xp-provider-gen create api --group=compute --version=v1alpha1 --kind=Instance
./bin/xp-provider-gen create api --group=storage --version=v1 --kind=Bucket

# Build and validate
make generate && make build && make reviewable
```

> **Important:** Always run `init` in a separate directory to avoid polluting your workspace.

> **Single initial commit:** `init` + each `create api` fold into one `Initial commit` while the
> provider is still being scaffolded. Finish scaffolding (and make your first own commit) **before
> pushing** — folding uses `git --amend`, so pushing mid-scaffold would require a force-push. Once
> you've committed your own work, later `create api` runs add separate commits.

## Commands

### `init` - Initialize provider project
```bash
xp-provider-gen init --domain=DOMAIN --repo=REPO [--git-name=NAME] [--git-email=EMAIL]
```

### `create api` - Add managed resource
```bash
xp-provider-gen create api --group=GROUP --version=VERSION --kind=KIND [--force]
```

### `create-test` - Scaffold a chainsaw behavior test
```bash
# Run inside a generated provider; prompts for name and kind when omitted.
xp-provider-gen create-test --name drift-check --kind MyType
```

### `update` - Refresh an existing provider's tool-owned core
```bash
# Run inside a generated provider with a clean working tree; review the diff, then commit.
xp-provider-gen update            # refresh registration, controller wiring, main.go, framework deps
xp-provider-gen update --adopt    # one-time: retrofit a provider made before the ownership contract
```
Tool-owned files (carrying the `DO NOT EDIT` header) are refreshed; your `external.go`,
`internal/provider/client.go`, `internal/provider/options.go`, `*_types.go`, and `go.mod`
requires are preserved. The result is left uncommitted for review.

> Providers generated before the modular layout — those with `controller.go` / `setup.go`
> per kind and no `internal/provider` package — must be **regenerated**. There is no
> in-place migration for that change.

See [docs/provider-guide.md](docs/provider-guide.md) for the full build-and-upgrade workflow.

## Working on This Project

```bash
make build       # Build the binary
make reviewable  # Everything CI enforces — run before pushing
make help        # List all targets
```

Requirements, the full target list and the contributor workflow are in
[docs/development.md](docs/development.md).

For the full developer guide see:

- [CLAUDE.md](CLAUDE.md) — project principles and conventions
- [docs/provider-guide.md](docs/provider-guide.md) — **for provider authors**: what to write, where, and how to upgrade
- [docs/architecture.md](docs/architecture.md) — how the generator works
- [docs/development.md](docs/development.md) — environment, tooling, and workflow
- [docs/testing.md](docs/testing.md) — unit and end-to-end testing

### Generated Project Structure

```
provider-awesome/
├── apis/
│   ├── v1alpha1/              # ProviderConfig types
│   ├── compute/v1alpha1/      # Compute resources
│   └── storage/v1/            # Storage resources
├── cmd/provider/              # Provider binary
├── internal/
│   ├── provider/              # Provider-wide concerns
│   │   ├── client.go          # YOURS — build the API client from credentials
│   │   ├── options.go         # YOURS — CLI flags, controller options
│   │   └── connector.go       # generated — ProviderConfig + credential resolution
│   └── controller/
│       ├── bucket/
│       │   ├── external.go    # YOURS — observe/create/update/delete
│       │   └── wiring.go      # generated — SetupGated, reconciler construction
│       ├── config/
│       │   └── config.go
│       └── register.go        # Controller registration
├── docs/ownership.md          # generated — which files are yours
├── AGENTS.md                  # yours — orientation for humans and agents
├── package/
│   ├── crossplane.yaml        # Provider metadata (with safe-start capability)
│   └── crds/                  # Generated CRDs
├── examples/                  # Usage examples
└── Makefile                   # Build automation
```

## License

Apache License 2.0