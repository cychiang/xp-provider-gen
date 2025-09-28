# Crossplane Provider Generator

A CLI tool for scaffolding Crossplane providers with Kubebuilder v4 and crossplane-runtime v2.

## Quick Start

### Build the Generator

```bash
git clone git@github.com:cychiang/xp-provider-gen.git
cd xp-provider-gen
make build
```

### Generate a Provider

```bash
# Initialize provider project (always use a separate directory)
mkdir my-provider && cd my-provider
./bin/xp-provider-gen init --domain=example.com --repo=github.com/example/provider-awesome

# Add managed resources
./bin/xp-provider-gen create api --group=compute --version=v1alpha1 --kind=Instance
./bin/xp-provider-gen create api --group=storage --version=v1 --kind=Bucket

# Build and validate
make generate && make build && make reviewable
```

> **Important:** Always run `init` in a separate directory to avoid polluting your workspace.

## Commands

### `init` - Initialize provider project
```bash
xp-provider-gen init --domain=DOMAIN --repo=REPO [--git-name=NAME] [--git-email=EMAIL]
```

### `create api` - Add managed resource
```bash
xp-provider-gen create api --group=GROUP --version=VERSION --kind=KIND [--force]
```

## Working on This Project

### Requirements
- Go 1.24.7+
- Git
- golangci-lint (for linting)
- gosec (for security scanning)

### Install Development Dependencies

**gosec (macOS):**
```bash
brew install gosec
```

**gosec (direct):**
```bash
go install github.com/securego/gosec/v2/cmd/gosec@v2.22.9
```

### Development Commands

```bash
make help        # Show all available commands
make build       # Build the binary
make test        # Run unit tests
make lint        # Run linter
make check       # Run all quality checks
make reviewable  # Make code ready for review
```

### End-to-End Testing

For comprehensive validation of the entire workflow:

```bash
make e2e-test
```

This command:
1. **Builds** the generator binary
2. **Creates** a test project at `/tmp/provider-template`
3. **Initializes** provider with `--domain=template.crossplane.io --repo=github.com/example/provider-template`
4. **Verifies** initial build targets (`make submodules`, `make generate`, `make reviewable`)
5. **Creates** two APIs: `sample/v1/MyType` and `sample/v1/MyValue`
6. **Validates** CRD and example generation
7. **Confirms** all build targets still work
8. **Cleans up** test artifacts

The e2e test provides confidence that the complete workflow functions correctly before committing changes.

### Working with Templates

The generator uses Go templates (`.tmpl` files) for code generation:

```bash
# Templates are located in:
pkg/templates/files/

# After modifying templates, rebuild:
make build

# Test changes with e2e test:
make e2e-test
```

**Template organization:**
- `pkg/templates/files/project/` - Project initialization templates
- `pkg/templates/files/api/` - API creation templates
- Templates support auto-discovery - add new `.tmpl` files and they're automatically included

### Generated Project Structure

```
provider-awesome/
├── apis/
│   ├── v1alpha1/              # ProviderConfig types
│   ├── compute/v1alpha1/      # Compute resources
│   └── storage/v1/            # Storage resources
├── cmd/provider/              # Provider binary
├── internal/controller/       # Controllers
├── package/crds/              # Generated CRDs
├── examples/                  # Usage examples
└── Makefile                   # Build automation
```

## License

Apache License 2.0