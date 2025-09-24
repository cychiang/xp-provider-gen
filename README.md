# Crossplane Provider Generator

A CLI tool for scaffolding Crossplane providers with Kubebuilder v4 and crossplane-runtime v2.

## Quick Start

### Installation

```bash
git clone git@github.com:cychiang/xp-provider-gen.git
cd xp-provider-gen
make build
```

### Generate a Provider

```bash
# Initialize provider project
mkdir my-provider && cd my-provider
xp-provider-gen init --domain=example.com --repo=github.com/example/provider-awesome

# Add managed resources
xp-provider-gen create api --group=compute --version=v1alpha1 --kind=Instance
xp-provider-gen create api --group=storage --version=v1 --kind=Bucket

# Build and validate
make generate && make build && make reviewable
```

## Features

- **Complete scaffolding** - Generates APIs, controllers, and build configuration
- **Multiple resources** - Support for multiple groups, versions, and kinds
- **No conflicts** - Each kind gets its own `{kind}_types.go` file
- **Automated setup** - Git initialization, build system integration, quality checks
- **Modern stack** - Kubebuilder v4 + crossplane-runtime v2

## Commands

### init

Initialize a new provider project:

```bash
xp-provider-gen init --domain=example.com --repo=github.com/example/provider-aws
```

**Required flags:**
- `--domain` - Domain for API groups
- `--repo` - Go module path

**Optional flags:**
- `--owner` - Copyright owner

### create api

Add a managed resource:

```bash
xp-provider-gen create api --group=compute --version=v1alpha1 --kind=Instance
```

**Required flags:**
- `--group` - Resource group (e.g., compute, storage)
- `--version` - API version (e.g., v1, v1alpha1, v1beta1)
- `--kind` - Resource kind (e.g., Instance, Bucket)

**Optional flags:**
- `--force` - Overwrite existing files

## Generated Structure

```
provider-awesome/
├── apis/
│   ├── v1alpha1/              # ProviderConfig types
│   ├── compute/v1alpha1/      # Compute resources
│   └── storage/v1/            # Storage resources
├── cmd/provider/              # Provider binary
├── internal/controller/       # Controllers
├── package/                   # Crossplane package
├── cluster/                   # Docker build
└── Makefile                   # Build automation
```

## Development Workflow

```bash
# In generated provider directory

make generate    # Generate CRDs and deepcopy code
make build       # Build provider binary
make reviewable  # Run linting and tests
make run         # Run provider locally
```

## Requirements

**Generator:**
- Go 1.24.5+
- Git

**Generated providers:**
- Go 1.24.5+
- Docker
- Make
- Crossplane build tools (auto-included)

## Project Structure

```
xp-provider-gen/
├── cmd/xp-provider-gen/       # CLI entrypoint
├── pkg/
│   ├── plugins/crossplane/v2/ # Kubebuilder plugin
│   │   ├── automation/        # Post-init automation pipeline
│   │   ├── core/             # Shared utilities (DRY)
│   │   ├── scaffold/         # Template execution
│   │   ├── templates/engine/ # Template rendering system
│   │   ├── validation/       # Input validation
│   │   ├── init.go          # Init subcommand
│   │   ├── createapi.go     # Create API subcommand
│   │   └── plugin.go        # Plugin registration
│   ├── templates/            # Centralized templates
│   │   ├── files/           # Template files (.tmpl)
│   │   └── loader.go        # Template loader (embed.FS)
│   └── version/             # Version info
└── docs/                     # Documentation
```

## Testing

### Generator tests

```bash
make test
```

### E2E workflow test

```bash
cd /tmp && mkdir test-provider && cd test-provider
xp-provider-gen init --domain=test.io --repo=github.com/test/provider-test
xp-provider-gen create api --group=compute --version=v1alpha1 --kind=Instance
make generate && make build && make reviewable
```

## Architecture

The plugin follows a clean architecture with clear separation of concerns:

- **automation/** - Post-scaffold automation pipeline (git, build setup)
- **core/** - Shared utilities following DRY principles
- **scaffold/** - Template execution (scaffolding logic only)
- **templates/engine/** - Template discovery and rendering
- **validation/** - Input validation and error handling

Each package has a single responsibility and no circular dependencies.

See [docs/implementation-complete.md](docs/implementation-complete.md) for detailed architecture documentation.

## Contributing

Contributions welcome! The project uses:

- **Template system** - Add `.tmpl` files for auto-discovery
- **Kubebuilder plugin** - Extends Kubebuilder v4 functionality
- **Modular architecture** - Clear package boundaries, easy to extend

See [docs/](docs/) for architecture details.

## License

Apache License 2.0 - See [LICENSE](LICENSE) for details.