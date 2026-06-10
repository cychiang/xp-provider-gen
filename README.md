# Crossplane Provider Generator

A CLI tool for scaffolding Crossplane providers with Kubebuilder v4 and crossplane-runtime v2.

## Key Features

- **🚀 Safe-Start Support**: Providers include Crossplane v2.0+ safe-start capability for selective resource activation
- **📦 Separated Controller Logic**: Setup/wiring logic isolated from business logic for better maintainability
- **🔧 Feature Flag Ready**: Automatic support for Management Policies, ChangeLogs, and metrics
- **🤖 Automated Workflows**: Built-in git operations, dependency management, and code generation
- **📝 Template Auto-Discovery**: Add new templates and they're automatically included

## What's New (v1.24.7)

- ✅ Safe-Start capability with gate-based controller activation
- ✅ Controller split: `controller.go` (business logic) + `setup.go` (wiring/features)
- ✅ Automated git workflows and build pipeline integration
- 🔧 Go 1.24.7 support

> **Breaking Change:** Controller structure changed. Existing projects need regeneration.

## Quick Start

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

## Commands

### `init` - Initialize provider project
```bash
xp-provider-gen init --domain=DOMAIN --repo=REPO [--git-name=NAME] [--git-email=EMAIL]
```

### `create api` - Add managed resource
```bash
xp-provider-gen create api --group=GROUP --version=VERSION --kind=KIND [--force]
```

## Working on This Project

Quick start for contributors:

```bash
make build       # Build the binary
make reviewable  # fmt + vet + lint + gosec + test (run before pushing)
make e2e-test    # Full scaffold → build workflow
make help        # List all targets
```

Requires Go 1.26+, Git, and gosec (`brew install gosec`). golangci-lint installs on demand.

For the full developer guide see:

- [CLAUDE.md](CLAUDE.md) — project principles and conventions
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
├── internal/controller/       # Controllers
│   ├── bucket/
│   │   ├── controller.go      # External client, CRUD logic
│   │   └── setup.go           # SetupGated + feature flags
│   ├── config/
│   │   └── config.go
│   └── register.go            # Controller registration
├── package/
│   ├── crossplane.yaml        # Provider metadata (with safe-start capability)
│   └── crds/                  # Generated CRDs
├── examples/                  # Usage examples
└── Makefile                   # Build automation
```

## License

Apache License 2.0