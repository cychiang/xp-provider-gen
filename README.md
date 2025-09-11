# Crossplane Provider Generator

A standalone CLI tool for scaffolding Crossplane providers with kubebuilder patterns and crossplane-runtime integration.

## Installation

**Build from source:**
```bash
git clone https://github.com/crossplane/xp-kubebuilder-plugin
cd xp-kubebuilder-plugin
go build -o bin/crossplane-provider-gen ./cmd/crossplane-provider-gen
```

## Usage

### Create a new provider project

```bash
mkdir my-provider && cd my-provider
crossplane-provider-gen init --domain=example.com --repo=github.com/example/my-provider

# Initialize dependencies (required after init)
make submodules
go mod tidy
make generate
make reviewable
```

### Add managed resources

```bash
# Create managed resources
crossplane-provider-gen create api --group=compute --version=v1alpha1 --kind=Instance
crossplane-provider-gen create api --group=storage --version=v1alpha1 --kind=Bucket

# Build the provider
make build
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

### Adding Templates

Templates are located in `pkg/plugins/crossplane/v2/scaffolds/`. Each template implements the `machinery.Template` interface and supports Go template substitution with `{{ .Repo }}`, `{{ .Domain }}`, and `{{ .ProviderName }}`.

Key areas for contribution:
- Enhanced API/controller templates
- Cloud-specific patterns (AWS, GCP, Azure)
- CI/CD workflow templates
- Documentation and examples

## Resources

- [Crossplane Provider Development](https://docs.crossplane.io/contribute/provider-development-guide/)
- [Crossplane Runtime v2](https://github.com/crossplane/crossplane-runtime)
- [Provider Template](https://github.com/crossplane/provider-template)