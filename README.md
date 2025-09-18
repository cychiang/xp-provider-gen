# Crossplane Provider Generator

A command-line tool for scaffolding Crossplane providers using Kubebuilder v4 and crossplane-runtime v2.

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

# Add managed resources with different versions
xp-provider-gen create api --group=compute --version=v1alpha1 --kind=Instance
xp-provider-gen create api --group=storage --version=v1 --kind=Bucket
xp-provider-gen create api --group=network --version=v1beta1 --kind=VPC

# Build and test
make generate && make build && make reviewable
```

That's it! You now have a fully functional Crossplane provider with:
- ✅ Generated ProviderConfig and ClusterProviderConfig CRDs
- ✅ Generated managed resource CRDs
- ✅ Complete controller scaffolding
- ✅ Ready-to-build Docker configuration

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
│   │   ├── providerconfig_types.go
│   │   └── register.go
│   ├── compute/v1alpha1/      # Instance managed resource
│   │   ├── instance_types.go   # Generated as KIND_types.go
│   │   └── groupversion_info.go
│   ├── storage/v1/            # Bucket and Volume (v1 version)
│   │   ├── bucket_types.go     # No conflicts - separate files!
│   │   ├── volume_types.go     # Multiple APIs per group/version
│   │   └── groupversion_info.go
│   ├── doc.go
│   ├── generate.go            # Code generation configuration
│   └── register.go            # Auto-updated import registry
├── cmd/provider/              # Main provider binary
├── internal/
│   ├── controller/            # Resource controllers
│   │   ├── bucket/           # Bucket controller
│   │   ├── instance/         # Instance controller
│   │   ├── volume/           # Volume controller
│   │   └── config/           # ProviderConfig controller
│   └── version/              # Version information
├── examples/                  # YAML examples
│   ├── provider/             # ProviderConfig examples
│   ├── compute/              # Instance examples
│   └── storage/              # Bucket and Volume examples
├── package/                   # Crossplane package
│   ├── crds/                 # Generated CRDs
│   │   ├── example.com_providerconfigs.yaml
│   │   ├── example.com_clusterproviderconfigs.yaml
│   │   ├── compute.example.com_instances.yaml
│   │   ├── storage.example.com_buckets.yaml
│   │   └── storage.example.com_volumes.yaml
│   └── crossplane.yaml       # Package metadata
├── cluster/                   # Docker build files
│   └── images/provider-name/ # Container configuration
├── hack/                     # Code generation scripts
├── build/                    # Crossplane build system (submodule)
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

### Automatic Template System

**No registration needed!** The system automatically:

1. **Discovers** all `.tmpl` files in `pkg/plugins/crossplane/v2/templates/scaffolds/`
2. **Categorizes** templates by their path
3. **Generates** TemplateType constants dynamically
4. **Registers** with appropriate factories
5. **Makes available** for immediate use

### Template Categories (Auto-Detected)

Templates are automatically categorized by their file path:

| Category | Path Patterns | When Used | Examples |
|----------|---------------|-----------|----------|
| **Init** | `root/`, `cmd/`, `internal/`, `apis/v1alpha1/`, `cluster/` | Project initialization | `Makefile.tmpl`, `main.go.tmpl` |
| **API** | `apis/GROUP/`, `internal/controller/KIND/`, `examples/GROUP/` | Adding managed resources | `KIND_types.go.tmpl`, `controller.go.tmpl` |
| **Static** | `LICENSE` | Standalone files | `LICENSE.tmpl` |

### Magic Path Variables

Use uppercase placeholders in template paths - they're replaced automatically:

| Variable | Replaced With | Example |
|----------|---------------|---------|
| `GROUP` | Resource group | `storage` |
| `VERSION` | API version | `v1alpha1` |
| `KIND` | Resource kind | `bucket` → `bucket_types.go` |
| `IMAGENAME` | Provider name | `provider-aws` |

### Template Variables

Available in all templates:

```go
// Project-level variables
{{ .Repo }}         // github.com/example/provider-aws
{{ .Domain }}       // aws.example.com
{{ .ProviderName }} // provider-aws
{{ .Boilerplate }}  // License header

// Resource-specific variables (API templates only)
{{ .Resource.Group }}          // compute
{{ .Resource.Version }}        // v1alpha1
{{ .Resource.Kind }}           // Instance
{{ .Resource.QualifiedGroup }} // compute.aws.example.com
```

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

## Contributing

We welcome contributions! The zero-step template system makes it easy to add new features:

1. **Add templates**: Just create `.tmpl` files - they're automatically discovered
2. **Fix bugs**: The codebase is well-structured and easy to navigate
3. **Improve docs**: Help make the project more accessible

## License

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) for details.
