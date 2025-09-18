# Crossplane Provider Generator

A command-line tool for scaffolding Crossplane providers using Kubebuilder v4 and crossplane-runtime v2.

## Features

- 🚀 **Quick Setup** - Initialize a complete Crossplane provider in seconds
- 📦 **Auto-Discovery** - Templates are automatically discovered and categorized
- 🔧 **Full Workflow** - From scaffolding to build-ready provider
- 🧪 **Battle-Tested** - Follows Crossplane v2 patterns and best practices
- ✅ **Complete CRD Generation** - Generates ProviderConfig, ClusterProviderConfig, and all managed resource CRDs
- 🔄 **Multi-API Support** - Create multiple APIs in the same group/version without conflicts
- 🎯 **Zero-Step Templates** - Add new templates instantly without registration

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

## Contributing: Zero-Step Template Development

This project features a revolutionary **zero-step template system** - just add your template file and it's automatically discovered!

### 🚀 Add Templates in Zero Steps

```bash
# Old way: 4+ manual steps (register, generate, compile, test)
# New way: 0 steps - just create your template file!

echo 'package {{ .Resource.Group }}' > pkg/plugins/crossplane/v2/templates/scaffolds/apis/GROUP/doc.go.tmpl
# That's it! Template is automatically:
# ✅ Discovered at runtime
# ✅ Categorized by path
# ✅ Registered with factory
# ✅ Ready to use immediately
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

### Benefits for Contributors

- **🎯 Zero friction** - Add templates instantly without boilerplate
- **🔍 Clear structure** - Directory layout matches generated project
- **✨ IDE friendly** - Full syntax highlighting and validation
- **🚀 Fast iteration** - No compilation step for template changes
- **📚 Self-documenting** - Template location shows where files are generated

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

## Recent Improvements

### ✅ Fixed CRD Generation Issues
- **ProviderConfig CRDs** - Now properly generates `providerconfigs.yaml` and `clusterproviderconfigs.yaml`
- **Complete CRD Set** - Generates all required CRDs including ProviderConfigUsage types
- **Automatic Discovery** - CRDs are discovered and generated automatically during `make generate`

### ✅ Multi-API Support
- **No More Conflicts** - Create multiple APIs in the same group/version (e.g., `storage/v1alpha1/Bucket` and `storage/v1alpha1/Volume`)
- **KIND-Specific Files** - Each resource gets its own types file (e.g., `bucket_types.go`, `volume_types.go`)
- **Isolated Development** - Work on multiple resources without overwriting each other

### ✅ Enhanced Template System
- **Zero Registration** - Add templates instantly without manual registration
- **Path-Based Discovery** - Template category automatically detected from file path
- **Runtime Discovery** - Templates discovered and registered at runtime

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
