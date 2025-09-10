# Crossplane Provider Generator

A standalone tool for scaffolding Crossplane provider projects and managed resources with proper crossplane-runtime integration.

## Overview

A standalone tool for generating Crossplane providers with best practices:
- Bootstrap complete Crossplane provider projects
- Generate managed resource APIs with crossplane-runtime integration
- Scaffold controllers following Crossplane best practices
- Create external client patterns for cloud API integration
- Use group-based directory structure (`apis/${group}/${version}/`)

## Status

🚀 **Standalone Tool Complete** - Full functionality implemented and working

- ✅ Standalone CLI tool (`crossplane-provider-gen`)
- ✅ Init subcommand for project scaffolding
- ✅ Create API subcommand with Crossplane patterns
- ✅ Crossplane-specific templates (Parameters/Observation, crossplane-runtime)
- ✅ Group-based directory structure generation
- ✅ Controller scaffolding with ExternalClient pattern
- ✅ Clean PROJECT file handling following kubebuilder patterns

## Quick Start

### Prerequisites
- Go 1.24+

### Installation

**Option 1: Build from Source**
```bash
git clone https://github.com/crossplane/xp-kubebuilder-plugin
cd xp-kubebuilder-plugin
make build
```

**Option 2: Download Binary** (when available)
```bash
# Download the latest release for your platform
curl -L -o crossplane-provider-gen https://github.com/crossplane/xp-kubebuilder-plugin/releases/latest/download/crossplane-provider-gen-linux-amd64
chmod +x crossplane-provider-gen
```

### Usage

**Standalone Crossplane Provider Generator**
```bash
# Initialize a new Crossplane provider project
mkdir my-crossplane-provider && cd my-crossplane-provider
crossplane-provider-gen init --domain=example.com --repo=github.com/example/my-crossplane-provider

# Create managed resource APIs
crossplane-provider-gen create api --group=compute --version=v1alpha1 --kind=Instance
crossplane-provider-gen create api --group=storage --version=v1beta1 --kind=Bucket
crossplane-provider-gen create api --group=network --version=v1alpha1 --kind=VPC
```


## Commands

| Command | Status | Description |
|---------|---------|-------------|
| `init` | ✅ Complete | Bootstrap Crossplane provider project |
| `create api` | ✅ Complete | Generate managed resource with controller |

### Create API Flags

The `create api` command supports these Crossplane-specific flags:

| Flag | Default | Description |
|------|---------|-------------|
| `--external-name` | `lowercase(kind)` | External resource name for cloud APIs |
| `--generate-client` | `true` | Generate external client interface |
| `--force` | `false` | Overwrite existing files |

### Example Commands

```bash
# Basic managed resource
crossplane-provider-gen create api --group=compute --version=v1alpha1 --kind=Instance

# With custom external name
crossplane-provider-gen create api --group=compute --version=v1alpha1 --kind=Instance \
  --external-name=ec2-instance

# Storage resource
crossplane-provider-gen create api --group=storage --version=v1beta1 --kind=Bucket

# Database resource
crossplane-provider-gen create api --group=database --version=v1alpha1 --kind=PostgreSQL
```

## Generated Structure

The plugin generates Crossplane-compatible directory structure with proper group organization:

```
my-provider/
├── PROJECT                           # Kubebuilder project config
├── apis/
│   ├── compute/
│   │   └── v1alpha1/
│   │       ├── groupversion_info.go  # Group version registration
│   │       └── instance_types.go     # Instance managed resource
│   ├── storage/
│   │   └── v1beta1/
│   │       ├── groupversion_info.go
│   │       └── bucket_types.go       # Bucket managed resource
│   └── network/
│       └── v1alpha1/
│           ├── groupversion_info.go
│           └── vpc_types.go          # VPC managed resource
└── internal/
    └── controller/
        ├── compute/
        │   └── instance/
        │       └── instance.go       # Instance controller
        ├── storage/
        │   └── bucket/
        │       └── bucket.go         # Bucket controller
        └── network/
            └── vpc/
                └── vpc.go            # VPC controller
```

## Generated Features

✅ **Crossplane-Specific API Types**
- `Parameters` struct for configuring external resources
- `Observation` struct for status from external resources
- `xpv2.ManagedResourceSpec` embedding for Crossplane resource spec
- `xpv1.ResourceStatus` embedding for Crossplane resource status
- Proper kubebuilder annotations for CRD generation

✅ **Controller Implementation**
- Uses `crossplane-runtime/v2` patterns
- Implements `managed.ExternalClient` interface
- External connector pattern for provider credentials
- Proper reconciliation logic with Create/Update/Delete/Observe methods
- Support for ProviderConfig and ClusterProviderConfig

✅ **Group-Based Organization**
- APIs organized by group: `apis/${group}/${version}/`
- Controllers organized by group: `internal/controller/${group}/${kind}/`
- Matches Crossplane provider-template structure

## Planned Features

### Init Command (TODO)
- Provider project structure with proper dependencies
- ProviderConfig CRD and controller
- Crossplane-specific Makefile and build system
- Package metadata for Crossplane registry

## Implementation Roadmap

### Phase 1: Core ✅ Complete
- ✅ Create API subcommand with Crossplane patterns
- ✅ Crossplane-specific templates and scaffolding
- ✅ Group-based directory structure

### Phase 2: Enhancement (High Priority)
- [ ] Init subcommand implementation
- [ ] Provider-specific templates (AWS, GCP, Azure)
- [ ] Enhanced template customization
- [ ] Integration testing

### Phase 3: Polish (Medium Priority) 
- [ ] Improved CLI experience and validation
- [ ] Documentation generation
- [ ] Code generation helpers
- [ ] Provider packaging support

## Development

```bash
# Build the plugin
go build -o bin/kubebuilder ./cmd/kubebuilder-with-crossplane

# Test the plugin
cd test-project
../bin/kubebuilder create api --group=compute --version=v1alpha1 --kind=Instance

# Run tests (when available)
go test ./...

# Check plugin registration
./bin/kubebuilder --help | grep crossplane
```

## Architecture

This plugin uses kubebuilder's internal plugin system:
- Implements `plugin.Full` interface
- Uses kubebuilder's resource model and config system  
- Integrates with `machinery.Filesystem` for scaffolding
- Follows official naming conventions: `crossplane.go.kubebuilder.io`

## Why This Plugin?

**vs Standard Kubebuilder:**
- Uses crossplane-runtime patterns instead of controller-runtime
- Generates external service client interfaces
- Crossplane-specific resource types and status management

**vs Crossplane provider-template:**
- Interactive command-line generation
- Provider specialization for different clouds
- Incremental API development

## Contributing

The plugin currently has working `create api` functionality. Areas for contribution:

1. **Init subcommand**: Bootstrap complete provider projects 
2. **Provider-specific templates**: Enhance templates for AWS/GCP/Azure specifics
3. **Testing**: Add comprehensive test coverage
4. **Documentation**: Improve examples and use cases
5. **CLI enhancements**: Better validation and user experience

## Resources

- [Kubebuilder Plugin Development](https://book.kubebuilder.io/plugins/extending)
- [Crossplane Provider Development](https://docs.crossplane.io/contribute/provider-development-guide/)
- [Crossplane Runtime](https://github.com/crossplane/crossplane-runtime)
- [Provider Template](https://github.com/crossplane/provider-template)

---

**Current Status:** Create API functionality is complete and working! Next priority is implementing the init subcommand for full provider project bootstrapping.