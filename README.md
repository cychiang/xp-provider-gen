# Crossplane Provider Generator

A command-line tool for scaffolding Crossplane providers using Kubebuilder v4 and crossplane-runtime v2.

## Features

- 🚀 **Quick Setup** - Initialize a complete Crossplane provider in seconds
- 📦 **Auto-Discovery** - Templates are automatically discovered and categorized
- 🔧 **Full Workflow** - From scaffolding to build-ready provider
- 🧪 **Battle-Tested** - Follows Crossplane v2 patterns and best practices

## Quick Start

### Installation

```bash
git clone https://github.com/cychiang/xp-provider-gen
cd xp-provider-gen
make build
```

### Create Your First Provider

```bash
# Create a new provider project
mkdir my-provider && cd my-provider
xp-provider-gen init --domain=example.com --repo=github.com/example/provider-awesome

# Add managed resources
xp-provider-gen create api --group=compute --version=v1alpha1 --kind=Instance
xp-provider-gen create api --group=storage --version=v1alpha1 --kind=Bucket

# Build and test
make generate && make build && make reviewable
```

That's it! You now have a fully functional Crossplane provider.

## Commands

### Initialize a Provider

```bash
xp-provider-gen init --domain=example.com --repo=github.com/example/provider-aws
```

**Options:**
- `--domain` - Domain for API groups (required)
- `--repo` - Go module repository (required)
- `--owner` - Copyright owner (optional)

### Add Managed Resources

```bash
xp-provider-gen create api --group=compute --version=v1alpha1 --kind=Instance
```

**Options:**
- `--group` - Resource group (e.g., compute, storage, network)
- `--version` - API version (e.g., v1alpha1, v1beta1)
- `--kind` - Resource kind (e.g., Instance, Bucket, VPC)
- `--force` - Overwrite existing files

## Generated Project Structure

```
provider-awesome/
├── apis/                       # API definitions
│   ├── v1alpha1/              # ProviderConfig types
│   ├── compute/v1alpha1/      # Instance managed resource
│   └── storage/v1alpha1/      # Bucket managed resource
├── cmd/provider/              # Main provider binary
├── internal/
│   ├── controller/            # Resource controllers
│   │   ├── bucket/           # Bucket controller
│   │   ├── instance/         # Instance controller
│   │   └── config/           # ProviderConfig controller
│   └── version/              # Version information
├── examples/                  # YAML examples
│   ├── provider/             # ProviderConfig examples
│   ├── compute/              # Instance examples
│   └── storage/              # Bucket examples
├── package/                   # Crossplane package
│   ├── crds/                 # Generated CRDs
│   └── crossplane.yaml       # Package metadata
├── cluster/                   # Docker build files
│   └── images/provider-name/ # Container configuration
├── hack/                     # Code generation scripts
└── Makefile                  # Build system
```

## Development Workflow

```bash
# Generate code and CRDs
make generate

# Build the provider binary
make build

# Run quality checks (lint, test)
make reviewable

# Run the provider locally
make run
```

## Template Development

The generator uses an automatic template discovery system. Simply add your template files and they're automatically available:

### Simple Templates

```bash
# Add any template - it's automatically discovered
echo 'package {{ .Resource.Group }}' > pkg/plugins/crossplane/v2/templates/scaffolds/apis/GROUP/doc.go.tmpl
```

### Template Categories

Templates are automatically categorized by their path:

| Category | Paths | Purpose |
|----------|-------|---------|
| **Init** | `root/`, `cmd/`, `internal/`, `apis/v1alpha1/`, `cluster/` | Project initialization |
| **API** | `apis/GROUP/`, `internal/controller/KIND/`, `examples/GROUP/` | Adding managed resources |
| **Static** | `LICENSE` | Standalone files |

### Path Variables

Use uppercase placeholders in template paths that get replaced automatically:

- `GROUP` → Resource group (e.g., `storage`)
- `VERSION` → API version (e.g., `v1alpha1`)
- `KIND` → Resource kind (e.g., `bucket`)
- `IMAGENAME` → Provider name (e.g., `provider-aws`)

## Testing

```bash
# Run unit tests
make test

# Test the generator end-to-end
cd /tmp && mkdir test-provider && cd test-provider
xp-provider-gen init --domain=test.io --repo=github.com/test/provider
xp-provider-gen create api --group=compute --version=v1alpha1 --kind=Instance
make generate && make build && make reviewable
```

## Build Commands

| Command | Description |
|---------|-------------|
| `make build` | Build the generator binary |
| `make test` | Run unit tests |
| `make clean` | Clean build artifacts |

## Requirements

- Go 1.24.5+
- Docker (for building providers)
- Git (for submodules)

## License

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) for details.
