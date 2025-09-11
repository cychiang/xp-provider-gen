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

🚀 **Full Crossplane Provider Generator Complete** - Complete functionality with provider-template compatibility

- ✅ Standalone CLI tool (`crossplane-provider-gen`)
- ✅ Init subcommand with complete Crossplane provider scaffolding
- ✅ ProviderConfig APIs (v1alpha1) with authentication support
- ✅ Package metadata (package/crossplane.yaml) for Crossplane registry
- ✅ Config controller for ProviderConfig management
- ✅ Version management with build-time injection
- ✅ Create API subcommand with Crossplane patterns
- ✅ Crossplane-specific templates (Parameters/Observation, crossplane-runtime)
- ✅ Group-based directory structure generation
- ✅ Controller scaffolding with ExternalClient pattern
- ✅ Build system integration with Crossplane makelib via git submodule

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

# Initialize the build system and dependencies (required after init)
make submodules
go mod tidy
make generate
make reviewable

# Create managed resource APIs
crossplane-provider-gen create api --group=compute --version=v1alpha1 --kind=Instance
crossplane-provider-gen create api --group=storage --version=v1beta1 --kind=Bucket
crossplane-provider-gen create api --group=network --version=v1alpha1 --kind=VPC

# Build the provider
make build
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

The plugin generates complete Crossplane provider structure matching provider-template:

```
my-provider/
├── PROJECT                           # Kubebuilder project config
├── go.mod                           # Go module definition
├── Makefile                         # Full Crossplane build system
├── Dockerfile                       # Container image build
├── README.md                        # Provider documentation
├── .gitignore                       # Git ignore patterns
├── .gitmodules                      # Crossplane build submodule
├── build/                           # Crossplane makelib (git submodule)
├── package/
│   └── crossplane.yaml              # Provider metadata for registry
├── cmd/provider/
│   └── main.go                      # Provider entry point
├── apis/
│   ├── v1alpha1/                    # ProviderConfig APIs
│   │   ├── doc.go                   # Package documentation
│   │   ├── register.go              # API registration
│   │   └── types.go                 # ProviderConfig, Credentials types
│   ├── compute/                     # Example managed resource group
│   │   └── v1alpha1/
│   │       ├── groupversion_info.go # Group version registration
│   │       └── instance_types.go    # Instance managed resource
│   └── storage/                     # Example managed resource group
│       └── v1beta1/
│           ├── groupversion_info.go
│           └── bucket_types.go      # Bucket managed resource
└── internal/
    ├── version/
    │   └── version.go               # Build-time version injection
    └── controller/
        ├── config/
        │   └── config.go            # ProviderConfig controller
        ├── compute/
        │   └── instance/
        │       └── instance.go      # Instance controller
        └── storage/
            └── bucket/
                └── bucket.go        # Bucket controller
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

## Implementation Status

### Phase 1: Critical Foundation ✅ **COMPLETED**
- ✅ **Init Command**: Complete Crossplane provider project scaffolding
- ✅ **ProviderConfig APIs**: Authentication and configuration management
- ✅ **Package Structure**: Crossplane registry integration
- ✅ **Build System**: Crossplane makelib integration via git submodule
- ✅ **Controller Infrastructure**: Config controller and version management
- ✅ **Create API Command**: Managed resource generation with Crossplane patterns
- ✅ **Template System**: Dynamic templates with project-specific substitution

### Phase 2: Development & CI Enhancement (Next Priority)
- [ ] **GitHub Workflows**: CI/CD pipelines for generated providers
- [ ] **Development Tools**: Enhanced linting, formatting, dependency management
- [ ] **Example Generation**: Usage examples and documentation scaffolds
- [ ] **Provider-specific templates**: Enhanced templates for AWS, GCP, Azure specifics

### Phase 3: Production Polish (Future)
- [ ] **Local Development**: Cluster setup and development workflows
- [ ] **Enhanced Documentation**: Provider-specific README templates and checklists  
- [ ] **Community Integration**: Contributing guidelines, code of conduct templates
- [ ] **Advanced Validation**: CLI experience improvements and validation

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

Thank you for your interest in contributing to the Crossplane Provider Generator! This project provides scaffolding templates for generating complete Crossplane providers following v2 patterns.

### Development Setup

```bash
# Clone the repository
git clone https://github.com/crossplane/xp-kubebuilder-plugin
cd xp-kubebuilder-plugin

# Build the generator
make build

# Test the generator
mkdir test-provider && cd test-provider
../bin/crossplane-provider-gen init --domain=example.com --repo=github.com/example/provider-test
make submodules && go mod tidy && make generate && make reviewable
```

### Adding New Templates

The plugin uses a structured template system for scaffolding files. Here's how to add new templates:

#### Template Structure

Templates are organized under `pkg/plugins/crossplane/v2/scaffolds/`:

```
pkg/plugins/crossplane/v2/scaffolds/
├── init.go                    # Init command scaffolder
├── internal/templates/        # Init command templates
│   ├── apis/                 # API registration templates
│   ├── controllers/          # Controller templates
│   ├── hack/                 # Build system templates
│   ├── pkg/                  # Package metadata templates
│   ├── providerconfig/       # ProviderConfig API templates
│   └── version/              # Version management templates
└── templates/                # Create API command templates
    ├── api/                  # Managed resource API templates
    └── controllers/          # Managed resource controller templates
```

#### Creating a New Template

1. **Create the template file** in the appropriate directory:

```go
// pkg/plugins/crossplane/v2/scaffolds/internal/templates/myfeature/myfile.go
package myfeature

import (
    "path/filepath"
    "sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &MyFile{}

type MyFile struct {
    machinery.TemplateMixin
    machinery.DomainMixin
    machinery.RepositoryMixin
    
    // Custom fields for template substitution
    ProviderName string
    CustomField  string
}

func (f *MyFile) SetTemplateDefaults() error {
    if f.Path == "" {
        f.Path = filepath.Join("path", "to", "myfile.ext")
    }
    f.TemplateBody = myFileTemplate
    f.IfExistsAction = machinery.OverwriteFile
    return nil
}

const myFileTemplate = `// Template content here
// Use {{ .Repo }}, {{ .Domain }}, {{ .ProviderName }} for substitution
package mypackage

// Your template content with Go template syntax
`
```

2. **Add template to scaffolder** in `init.go` or `createapi.go`:

```go
// Add to the Execute() call in the appropriate scaffolder
&myfeature.MyFile{
    TemplateMixin: machinery.TemplateMixin{},
    DomainMixin: machinery.DomainMixin{Domain: domain},
    RepositoryMixin: machinery.RepositoryMixin{Repo: repo},
    ProviderName: providerName,
    CustomField: "value",
},
```

#### Template Guidelines

- **Follow Crossplane v2 patterns**: Use `crossplane-runtime/v2` imports and patterns
- **Use proper kubebuilder annotations**: Include necessary `+kubebuilder:` annotations
- **Template substitution**: Use `{{ .Repo }}`, `{{ .Domain }}`, `{{ .ProviderName }}` for dynamic values
- **File organization**: Group related templates in logical directories
- **Consistent naming**: Use descriptive names matching the generated file purpose
- **Error handling**: Ensure templates are syntactically correct and handle edge cases

#### Testing Templates

Always test new templates by:

1. **Building the generator**: `make build`
2. **Generating a test provider**: Create a temporary directory and run init
3. **Verifying the workflow**: Ensure `make submodules && go mod tidy && make generate && make reviewable` passes
4. **Checking output**: Verify generated files match expected structure and content

### Areas for Contribution

1. **Enhanced Templates**: Improve existing templates with better defaults and patterns
2. **Provider-specific Templates**: Add cloud-specific templates (AWS, GCP, Azure patterns)
3. **GitHub Workflows**: Add CI/CD pipeline templates for generated providers
4. **Examples**: Add template generation for example resources and documentation
5. **Testing**: Expand test coverage for template generation and validation
6. **CLI Enhancements**: Improve user experience with better validation and help text

## Resources

- [Kubebuilder Plugin Development](https://book.kubebuilder.io/plugins/extending)
- [Crossplane Provider Development](https://docs.crossplane.io/contribute/provider-development-guide/)
- [Crossplane Runtime](https://github.com/crossplane/crossplane-runtime)
- [Provider Template](https://github.com/crossplane/provider-template)

---

**Current Status:** Create API functionality is complete and working! Next priority is implementing the init subcommand for full provider project bootstrapping.