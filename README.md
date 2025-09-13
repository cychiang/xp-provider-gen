# Crossplane Provider Generator

A standalone CLI tool for scaffolding Crossplane providers with kubebuilder patterns and crossplane-runtime v2 integration. Features a revolutionary single-responsibility template architecture with comprehensive test framework for Test-Driven Development.

## Installation

**Build from source:**
```bash
git clone https://github.com/crossplane/xp-kubebuilder-plugin
cd xp-kubebuilder-plugin
make build
```

## Quick Start

### Create a new provider project

```bash
mkdir my-provider && cd my-provider
../bin/crossplane-provider-gen init --domain=example.com --repo=github.com/example/my-provider

# Initialize dependencies (required after init)
make submodules
go mod tidy
make generate
make reviewable
```

### Add managed resources

```bash
# Create managed resources
../bin/crossplane-provider-gen create api --group=compute --version=v1alpha1 --kind=Instance
../bin/crossplane-provider-gen create api --group=storage --version=v1alpha1 --kind=Bucket

# Build and test
make build
make test
```

## Commands

| Command | Description |
|---------|-------------|
| `init` | Bootstrap a Crossplane provider project |
| `create api` | Generate a managed resource with controller |

### Flags

**init command:**
- `--domain` - Domain for API groups (required)
- `--repo` - Go module name (required)

**create api command:**
- `--group` - API group name (required)
- `--version` - API version (required)  
- `--kind` - Resource kind (required)
- `--generate-client` - Generate external client interface (default: true)
- `--force` - Overwrite existing files (default: false)

## Generated Project Structure

```
my-provider/
├── apis/
│   ├── v1alpha1/              # ProviderConfig APIs
│   └── compute/v1alpha1/      # Managed resource APIs
├── cmd/provider/main.go       # Provider entry point
├── internal/controller/       # Controllers
├── package/crossplane.yaml    # Provider metadata
└── Makefile                   # Build system
```

## Generated Features

- **Crossplane v2 runtime patterns** with proper `Parameters`/`Observation` structs
- **External client interfaces** for cloud API integration
- **ProviderConfig APIs** for authentication and configuration
- **Controller scaffolding** following Crossplane best practices
- **Build system integration** via git submodules
- **PROJECT file tracking** for resource management

## Contributing

### Development Setup

```bash
# Clone and build
git clone https://github.com/crossplane/xp-kubebuilder-plugin
cd xp-kubebuilder-plugin
go build -o bin/crossplane-provider-gen ./cmd/crossplane-provider-gen

# Test generation
mkdir test-provider && cd test-provider
../bin/crossplane-provider-gen init --domain=test.io --repo=github.com/test/provider-test
make submodules && go mod tidy && make generate && make reviewable
```

## Architecture

### Single-Responsibility Template System

This project features a revolutionary template architecture where **each template has its own file** following Single Responsibility Principle:

```
pkg/plugins/crossplane/v2/templates/
├── factory.go              # Clean factory pattern (122 lines)
├── base.go                 # Template infrastructure
├── boilerplate.go          # Centralized Apache 2.0 license
├── templates_test.go       # Comprehensive test framework (18 tests)
├── gomod_template.go       # go.mod template only
├── makefile_template.go    # Makefile template only
├── readme_template.go      # README.md template only
├── gitignore_template.go   # .gitignore template only
├── main_go_template.go     # cmd/provider/main.go only
├── api_types_template.go   # CRD types only
├── controller_template.go  # Controller implementation only
├── provider_config_*.go    # ProviderConfig types & registration
├── crossplane_package.go   # package/crossplane.yaml only
├── cluster_*.go           # Container build files only
├── license.go             # LICENSE file only
└── [17 more single-purpose templates]
```

### Test-Driven Development Ready

The test framework enables TDD for template development:

```bash
# Test all templates
make test

# Run specific template validation
go test -v ./pkg/plugins/crossplane/v2/templates/

# TDD workflow example:
# 1. Write test for new template first
# 2. Run tests (should fail)
# 3. Implement template
# 4. Run tests (should pass)
```

### Template Development

Each template implements the `machinery.Template` interface with Go template substitution supporting `{{ .Repo }}`, `{{ .Domain }}`, and `{{ .ProviderName }}`.

**Adding a new template:**
1. Create `new_feature_template.go` with single template function
2. Add factory method in `factory.go`
3. Write test in `templates_test.go`
4. Run `make test` to validate

Key areas for contribution:
- Enhanced API/controller templates  
- Cloud-specific patterns (AWS, GCP, Azure)
- CI/CD workflow templates
- Documentation and examples

## Resources

- [Crossplane Provider Development](https://docs.crossplane.io/contribute/provider-development-guide/)
- [Crossplane Runtime v2](https://github.com/crossplane/crossplane-runtime)
- [Provider Template](https://github.com/crossplane/provider-template)