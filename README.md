# Crossplane Provider Generator

A command-line tool for scaffolding Crossplane providers using Kubebuilder v4 and crossplane-runtime v2. This tool generates complete, production-ready Crossplane provider projects with proper API structure, controllers, and build systems.

## Features

- 🚀 **Zero-configuration setup**: Automatically generates complete provider projects
- 🔧 **Modern tooling**: Built on Kubebuilder v4 and crossplane-runtime v2
- 📦 **Proper API structure**: Supports multiple groups, versions, and kinds without conflicts
- 🏗️ **Automated build system**: Integrates with Crossplane build tooling via git submodules
- ✅ **Quality assurance**: Includes linting, testing, and code generation workflows
- 🎯 **Developer-friendly**: Clean separation between user code and generated code

## Quick Start

### Installation

Build from source:
```bash
git clone https://github.com/cychiang/xp-provider-gen
cd xp-provider-gen
make build
./bin/xp-provider-gen --help
```

### Create Your First Provider

```bash
# Create a new provider project
mkdir my-provider && cd my-provider
xp-provider-gen init --domain=example.com --repo=github.com/example/provider-awesome

# Add managed resources with different versions
xp-provider-gen create api --group=compute --version=v1alpha1 --kind=Instance
xp-provider-gen create api --group=storage --version=v1 --kind=Bucket
xp-provider-gen create api --group=network --version=v1beta1 --kind=VPC

# Build and test
make generate && make build && make reviewable
```

The generated provider includes:
- ✅ **ProviderConfig and ClusterProviderConfig** CRDs for authentication
- ✅ **Managed resource CRDs** with proper API versioning
- ✅ **Complete controller scaffolding** following Crossplane patterns
- ✅ **Docker build configuration** and package metadata
- ✅ **Git repository setup** with Crossplane build system integration

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
# Create APIs with any version
xp-provider-gen create api --group=compute --version=v1alpha1 --kind=Instance
xp-provider-gen create api --group=storage --version=v1 --kind=Bucket
xp-provider-gen create api --group=network --version=v1beta1 --kind=VPC

# Multiple resources in same group/version (no conflicts!)
xp-provider-gen create api --group=storage --version=v1 --kind=Bucket
xp-provider-gen create api --group=storage --version=v1 --kind=Volume
```

**Options:**
- `--group` - Resource group (e.g., compute, storage, network)
- `--version` - API version (any format: v1, v1alpha1, v1beta1, v2, etc.)
- `--kind` - Resource kind (e.g., Instance, Bucket, VPC)
- `--force` - Overwrite existing files

## Generated Project Structure

```
provider-awesome/
├── apis/                       # API definitions
│   ├── v1alpha1/              # ProviderConfig types
│   │   ├── types.go           # ProviderConfig and ClusterProviderConfig
│   │   └── register.go        # Type registration and metadata
│   ├── compute/v1alpha1/      # Instance managed resource
│   │   ├── instance_types.go  # Generated as KIND_types.go (no conflicts!)
│   │   └── groupversion_info.go
│   ├── storage/v1/            # Bucket and Volume (v1 version)
│   │   ├── bucket_types.go    # Multiple APIs per group/version supported
│   │   ├── volume_types.go    # Each KIND gets its own file
│   │   └── groupversion_info.go
│   ├── doc.go                 # Package documentation
│   ├── generate.go            # Code generation configuration
│   └── register.go            # Auto-updated import registry
├── cmd/provider/              # Main provider binary
│   └── main.go               # Provider entrypoint
├── internal/
│   ├── controller/            # Resource controllers
│   │   ├── bucket/           # Bucket controller
│   │   │   └── controller.go # CRUD operations for Bucket
│   │   ├── instance/         # Instance controller
│   │   │   └── controller.go # CRUD operations for Instance
│   │   ├── volume/           # Volume controller
│   │   │   └── controller.go # CRUD operations for Volume
│   │   ├── config/           # ProviderConfig controller
│   │   │   └── config.go     # Authentication handling
│   │   └── register.go       # Controller registration
│   └── version/              # Version information
│       └── version.go
├── package/                   # Crossplane package
│   ├── crds/                 # Generated CRDs (auto-created by make generate)
│   │   ├── example.com_providerconfigs.yaml
│   │   ├── example.com_clusterproviderconfigs.yaml
│   │   ├── compute.example.com_instances.yaml
│   │   ├── storage.example.com_buckets.yaml
│   │   └── storage.example.com_volumes.yaml
│   └── crossplane.yaml       # Package metadata
├── cluster/                   # Docker build files
│   └── images/provider-awesome/
│       ├── Dockerfile        # Multi-stage build
│       └── Makefile         # Image build configuration
├── hack/                     # Code generation scripts
│   └── boilerplate.go.txt   # License header for generated files
├── build/                    # Crossplane build system (git submodule)
├── .gitignore               # Ignores build artifacts and IDE files
└── Makefile                 # Build system integration
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

## Key Features

### Smart API Structure
- **No file conflicts**: Each KIND gets its own `{kind}_types.go` file
- **Multiple APIs per group/version**: Add as many resources as needed
- **Proper versioning**: Support for any API version (v1, v1alpha1, v1beta1, etc.)
- **Clean separation**: Provider config types in `apis/v1alpha1/`, managed resources in `apis/{group}/{version}/`

### Automated Setup
- **Git integration**: Automatically initializes repository with proper `.gitignore`
- **Build system**: Integrates Crossplane build tooling via git submodules
- **Quality checks**: Includes linting, testing, and code generation workflows
- **Complete controllers**: Generates working CRUD controllers following Crossplane patterns

### Developer Experience
- **Zero configuration**: Works out of the box with sensible defaults
- **Extensible templates**: Template system with automatic discovery
- **Modern tooling**: Built on latest Kubebuilder v4 and crossplane-runtime v2
- **Clean code**: Minimal comments, focused on essential documentation

## Testing

### Unit Tests
```bash
# Run generator unit tests
make test
```

### End-to-End Testing
```bash
# Test complete provider generation workflow
cd /tmp && mkdir test-provider && cd test-provider
xp-provider-gen init --domain=test.io --repo=github.com/test/provider-test
xp-provider-gen create api --group=compute --version=v1alpha1 --kind=Instance
xp-provider-gen create api --group=storage --version=v1 --kind=Bucket
make generate && make build && make reviewable
```

### Testing Generated Providers
```bash
# In generated provider directory
make test          # Run provider unit tests
make e2e          # Run end-to-end tests (if available)
make reviewable   # Comprehensive quality checks
```

## Build Commands

| Command | Description |
|---------|-------------|
| `make build` | Build the xp-provider-gen binary |
| `make test` | Run unit tests for the generator |
| `make clean` | Clean build artifacts |
| `make lint` | Run code linting |

## Requirements

**For the generator:**
- Go 1.24.5+
- Git (for version control and submodules)

**For generated providers:**
- Go 1.24.5+
- Docker (for building provider images)
- Make (for build automation)
- Crossplane build tools (automatically included via git submodule)

## Contributing

Contributions are welcome! The project is designed for easy extension:

1. **Template system**: Add new `.tmpl` files and they're automatically discovered
2. **Clean architecture**: Well-structured codebase with clear separation of concerns
3. **Comprehensive tests**: Ensure changes work with existing functionality
4. **Documentation**: Help improve user experience and developer onboarding

## License

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) for details.
