# Crossplane Provider Generator

A CLI tool for scaffolding Crossplane providers with Kubebuilder v4 and crossplane-runtime v2.

## Key Features

- **♻️ Upgradable core** — `update` refreshes a provider's tool-owned plumbing (wiring,
  registration, `main.go`, framework deps) without touching your business logic, and
  `update --adopt` retrofits providers made before the contract existed
- **📦 Modular layout** — the framework plumbing is tool-owned; you write a handful of named
  seams, split per kind into your `external.go` and generated `wiring.go` — see
  [the seam contract](docs/architecture.md#9-seams)
- **🔒 File-ownership contract** — a `// Code generated … DO NOT EDIT.` header decides what
  `update` may rewrite; enforced by a golden test and published as a generated
  `docs/ownership.md` inside every provider
- **🚀 Safe-Start support** — Crossplane v2.0+ selective resource activation, plus
  Management Policies, ChangeLogs and metrics
- **🧪 E2E out of the box** — every **native** scaffold ships uptest lifecycle tests and
  chainsaw behavior tests; `create-test` adds more
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

`init`, `create api` (add `--upjet`/`--terraform-*` flags to wrap a Terraform provider
instead), `create-test`, and `update` (`--adopt` for a pre-contract provider). Full flags,
examples, and what each one does are in
[docs/provider-guide.md](docs/provider-guide.md) (native), [docs/upjet-provider.md](docs/upjet-provider.md)
(upjet), and the [xp-provider-gen skill](.agents/skills/xp-provider-gen/SKILL.md).

## Working on This Project

```bash
make build       # Build the binary
make reviewable  # Everything CI enforces — run before pushing
make help        # List all targets
```

Requirements, the full target list and the contributor workflow are in
[docs/development.md](docs/development.md).

**For provider authors**, start with [docs/provider-guide.md](docs/provider-guide.md) (native)
or [docs/upjet-provider.md](docs/upjet-provider.md) (upjet). **For contributors to this
generator**, [AGENTS.md](AGENTS.md) indexes the rest of `docs/`.

### Generated Project Structure

Native flavor (an upjet scaffold's layout differs — see
[docs/upjet-provider.md](docs/upjet-provider.md)):

```
provider-awesome/
├── apis/
│   ├── v1alpha1/              # ProviderConfig types
│   ├── compute/v1alpha1/      # Compute resources
│   ├── storage/v1/            # Storage resources
│   └── register.go            # generated — scheme registration
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
├── test/                      # YOURS — chainsaw behavior tests + setup script
│   ├── setup.sh
│   └── behavior/               # chainsaw behavior tests
├── cluster/local/integration_tests.sh
├── hack/
│   ├── boilerplate.go.txt     # license header for generated code
│   └── xp-provider-gen.mk     # generated — the build pipeline the Makefile includes
├── docs/ownership.md          # generated — which files are yours
├── AGENTS.md                  # yours — orientation for humans and agents
├── OWNERS.md
├── LICENSE
├── .gitignore
├── package/
│   ├── crossplane.yaml        # Provider metadata (with safe-start capability)
│   └── crds/                  # Generated CRDs
├── examples/                  # usage examples — also uptest's lifecycle input (make e2e)
└── Makefile                   # YOURS — project variables, then includes hack/xp-provider-gen.mk
```

## License

Apache License 2.0