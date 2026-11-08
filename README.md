# Crossplane Provider Generator

A Go CLI that scaffolds native and upjet Crossplane providers on crossplane-runtime v2.

## Key Features

- **♻️ Upgradable core** — `update` refreshes a provider's tool-owned plumbing (wiring,
  registration, `main.go`, framework deps) without touching your business logic
- **📦 Modular layout** — the framework plumbing is tool-owned; you write a handful of named
  seams, split per kind into your `external.go` and generated `wiring.go` — see
  [the seam contract](docs/architecture.md#9-seams-the-modular-layout)
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

### Install the Generator

```bash
go install github.com/cychiang/xp-provider-gen/cmd/xp-provider-gen@latest
```

Prebuilt binaries for Linux, macOS and Windows, with checksums, are on the
[Releases](https://github.com/cychiang/xp-provider-gen/releases) page, and a multi-arch
container image is published as `ghcr.io/cychiang/xp-provider-gen`. To build from source
instead:

```bash
git clone git@github.com:cychiang/xp-provider-gen.git
cd xp-provider-gen
make build
export PATH="$PWD/bin:$PATH"
```

### Generate a Provider

```bash
# Somewhere else entirely — never inside the xp-provider-gen clone itself
mkdir my-provider && cd my-provider
xp-provider-gen init --domain=example.com --repo=github.com/example/provider-awesome

# Add managed resources
xp-provider-gen create api --group=compute --version=v1alpha1 --kind=Instance
xp-provider-gen create api --group=storage --version=v1 --kind=Bucket

# Build and validate
make generate && make build && make reviewable
```

> **Important:** Always run `init` in a separate directory to avoid polluting your workspace.

> **Single initial commit:** `init` + each `create api` fold into one `Initial commit` while the
> provider is still being scaffolded. Finish scaffolding (and make your first own commit) **before
> pushing** — folding uses `git --amend`, so pushing mid-scaffold would require a force-push. Once
> you've committed your own work, later `create api` runs add separate commits.

## Commands

`init` (add `--upjet` and the `--terraform-*` flags to wrap a Terraform provider instead), `create api`
(`--terraform-resource` on an upjet project), `create-test`, and `update`. `xp-provider-gen <command> --help`
lists every flag. What each command does is in
[docs/provider-guide.md](docs/provider-guide.md) (native), [docs/upjet-provider.md](docs/upjet-provider.md)
(upjet), and the [xp-provider-gen skill](.agents/skills/xp-provider-gen/SKILL.md).

## Working on This Project

```bash
make build         # Build the binary
make reviewable    # Everything CI enforces — run before pushing
make e2e-native    # Scaffold a native provider and run it end to end (needs Docker)
make e2e-upjet     # The same for an upjet provider (needs Docker and network)
make help          # List all targets
```

The e2e flags, including how to measure an upgrade, are in [docs/testing.md](docs/testing.md).
Requirements and the contributor workflow are in [docs/development.md](docs/development.md).

**Provider authors** start with [docs/provider-guide.md](docs/provider-guide.md) (native) /
[docs/upjet-provider.md](docs/upjet-provider.md) (upjet); **generator contributors** start with
[AGENTS.md](AGENTS.md).

### Generated Project Structure

On a native provider you write `external.go` (observe/create/update/delete), `client.go` (the API
client), `options.go` (CLI flags), each kind's `apis/<group>/<version>/<kind>_types.go`, and
`apis/v1alpha1/types.go` when the ProviderConfig needs settings. The framework wiring is tool-owned,
and the remaining files are seeded once and then yours. Every scaffold gets its own generated `docs/ownership.md`
listing exactly which files are yours for that provider — see
[docs/provider-guide.md](docs/provider-guide.md) (native) or
[docs/upjet-provider.md](docs/upjet-provider.md) (upjet, whose layout differs) for the full
picture.

## License

Apache License 2.0