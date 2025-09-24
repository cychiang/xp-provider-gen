# Crossplane Provider Generator

A CLI tool for scaffolding Crossplane providers with Kubebuilder v4 and crossplane-runtime v2.

## About This Project

This project is a specialized Kubebuilder plugin that generates complete Crossplane provider projects. It automates the creation of APIs, controllers, build configurations, and all necessary scaffolding for building production-ready Crossplane providers.

**Key capabilities:**
- Scaffolds complete provider projects with modern Crossplane v2 runtime
- Supports multiple API groups, versions, and resource kinds in a single provider
- Generates conflict-free resource types with dedicated files per kind
- Includes automated git setup, build system integration, and quality checks
- Uses template-based generation with auto-discovery for extensibility

## Project Structure

This generator tool is organized as follows:

```
xp-provider-gen/
├── cmd/xp-provider-gen/       # CLI application entrypoint
├── pkg/
│   ├── plugins/crossplane/v2/ # Kubebuilder plugin implementation
│   │   ├── automation/        # Post-init automation pipeline
│   │   ├── core/             # Shared utilities (DRY principles)
│   │   ├── scaffold/         # Template execution logic
│   │   ├── templates/engine/ # Template discovery and rendering
│   │   ├── validation/       # Input validation and error handling
│   │   ├── init.go          # Init subcommand implementation
│   │   ├── createapi.go     # Create API subcommand
│   │   └── plugin.go        # Plugin registration with Kubebuilder
│   ├── templates/            # Template files and loading
│   │   ├── files/           # All .tmpl template files
│   │   └── loader.go        # Template filesystem (embed.FS)
│   └── version/             # Version information
├── bin/                      # Built binaries (after make build)
├── Makefile                  # Build automation
└── README.md                 # This documentation
```

**Architecture principles:**
- Each package has a single, focused responsibility
- No circular dependencies between packages
- Template-driven generation with auto-discovery
- Clean separation between scaffolding logic and automation
- Extensible through additional `.tmpl` files

## Quick Start

### Installation

```bash
git clone git@github.com:cychiang/xp-provider-gen.git
cd xp-provider-gen
make build
```

### Generate a Provider

```bash
# Initialize provider project (always use a separate directory)
mkdir my-provider && cd my-provider
xp-provider-gen init --domain=example.com --repo=github.com/example/provider-awesome

# Add managed resources
xp-provider-gen create api --group=compute --version=v1alpha1 --kind=Instance
xp-provider-gen create api --group=storage --version=v1 --kind=Bucket

# Build and validate
make generate && make build && make reviewable
```

> **Important:** Always run `init` in a separate directory to avoid polluting your workspace with generated files.

## Features

- **Complete scaffolding** - Generates APIs, controllers, and build configuration
- **Multiple resources** - Support for multiple groups, versions, and kinds
- **No conflicts** - Each kind gets its own `{kind}_types.go` file
- **Automated setup** - Git initialization, build system integration, quality checks
- **Modern stack** - Kubebuilder v4 + crossplane-runtime v2

## Commands

### `init` - Initialize provider project

**Usage:**
```bash
xp-provider-gen init --domain=DOMAIN --repo=REPO [--owner=OWNER]
```

**Flags:**
- `--domain` (required) - Domain for API groups
- `--repo` (required) - Go module path
- `--owner` (optional) - Copyright owner

### `create api` - Add managed resource

**Usage:**
```bash
xp-provider-gen create api --group=GROUP --version=VERSION --kind=KIND [--force]
```

**Flags:**
- `--group` (required) - Resource group (e.g., compute, storage)
- `--version` (required) - API version (e.g., v1, v1alpha1, v1beta1)
- `--kind` (required) - Resource kind (e.g., Instance, Bucket)
- `--force` (optional) - Overwrite existing files

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

## Requirements

- Go 1.24.5+
- Git
- Docker (for generated providers)
- Make (for generated providers)

## Development Workflow

```bash
# In generated provider directory
make generate    # Generate CRDs and deepcopy code
make build       # Build provider binary
make reviewable  # Run linting and tests
make run         # Run provider locally
```

## Testing

```bash
# Run generator tests
make test

# E2E workflow test
cd /tmp && mkdir test-provider && cd test-provider
xp-provider-gen init --domain=test.io --repo=github.com/test/provider-test
xp-provider-gen create api --group=compute --version=v1alpha1 --kind=Instance
make generate && make build && make reviewable
```

## Contributing

Contributions welcome! Key technologies:
- **Template system** - Add `.tmpl` files for auto-discovery
- **Kubebuilder plugin** - Extends Kubebuilder v4 functionality
- **Modular architecture** - Clear package boundaries, easy to extend

## License

Apache License 2.0