# Crossplane Provider Generator

CLI tool for scaffolding Crossplane providers with crossplane-runtime v2.

## Installation

```bash
git clone https://github.com/crossplane/xp-kubebuilder-plugin
cd xp-kubebuilder-plugin
make build
```

## Usage

### Initialize Provider

```bash
./bin/crossplane-provider-gen init --domain=example.com --repo=github.com/example/provider-name
cd $PROJECT_DIR
make submodules && go mod tidy && make generate
```

### Add Managed Resources

```bash
./bin/crossplane-provider-gen create api --group=compute --version=v1alpha1 --kind=Instance
make generate && make build
```

## Project Structure

```
provider-name/
├── apis/
│   ├── v1alpha1/           # ProviderConfig APIs
│   └── GROUP/VERSION/      # Managed resource APIs
├── cmd/provider/           # Provider binary
├── internal/controller/    # Controllers
├── examples/               # YAML examples
├── package/                # Crossplane package
└── Makefile               # Build system
```

## Template System

Templates are organized in `pkg/plugins/crossplane/v2/templates/scaffolds/`:

```
scaffolds/
├── root/                   # Root-level files (Makefile, README, etc.)
├── apis/                   # API-related templates
├── cmd/provider/           # Main binary templates
├── internal/controller/    # Controller templates
├── examples/               # Example YAML templates
└── cluster/                # Container build templates
```

### Adding New Templates

1. **Create template file**: Add `.tmpl` file in appropriate `scaffolds/` subdirectory
2. **Create product**: Add corresponding `*TemplateProduct` in `templates/products_*.go`
3. **Register template**: Add to factory in `templates/factory.go`
4. **Add to scaffolds**: Include in `scaffolds/init.go` or `createapi.go`

Example:
```go
// 1. Create scaffolds/examples/new-feature.yaml.tmpl
// 2. Create NewFeatureTemplateProduct in products_api.go
// 3. Register NewFeatureTemplateType in factory.go
// 4. Add to API scaffolding in createapi.go
```

## Build Commands

```bash
make build       # Build provider binary
make test        # Run tests
make reviewable  # Run all quality checks
make submodules  # Update build system
```