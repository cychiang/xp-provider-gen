# Crossplane Provider Generator

CLI tool for scaffolding Crossplane providers using Kubebuilder v4 and crossplane-runtime v2.

## Quick Start

```bash
# Install
git clone https://github.com/cychiang/xp-provider-gen
cd xp-provider-gen
make build

# Create a new provider
mkdir my-provider && cd my-provider
xp-provider-gen init --domain=example.com --repo=github.com/example/provider-awesome

# Add managed resources
xp-provider-gen create api --group=compute --version=v1alpha1 --kind=Instance
xp-provider-gen create api --group=storage --version=v1alpha1 --kind=Bucket

# Build and test
make generate && make build && make reviewable
```

## Generated Project Structure

```
provider-awesome/
├── apis/
│   ├── v1alpha1/           # ProviderConfig types
│   ├── compute/v1alpha1/   # Instance managed resource
│   └── storage/v1alpha1/   # Bucket managed resource
├── cmd/provider/           # Main provider binary
├── internal/controller/    # Resource controllers
├── examples/               # YAML examples
│   ├── provider/           # ProviderConfig examples
│   ├── compute/            # Instance examples
│   └── storage/            # Bucket examples
├── package/                # Crossplane package definition
├── cluster/                # Docker build files
└── Makefile               # Build system
```

## For Developers: Adding Templates

### 1. Simple Templates (Most Common)

Just add your template file. It's automatically discovered:

```bash
# Add a new template
echo 'package {{ .Resource.Group }}' > pkg/plugins/crossplane/v2/templates/scaffolds/apis/group/doc.go.tmpl
```

That's it! The template is automatically:
- Discovered at runtime
- Named `ApisGroupDocGoType`
- Available for use

### 2. Templates with Custom Logic

If your template needs special handling, add pattern matching:

**File:** `pkg/plugins/crossplane/v2/templates/builders.go`
```go
case strings.Contains(typeStr, "mydocument"):
    product = &MyDocumentTemplateProduct{BaseTemplateProduct: NewBaseTemplateProduct(b.templateType)}
```

**File:** `pkg/plugins/crossplane/v2/templates/products_*.go`
```go
type MyDocumentTemplateProduct struct {
    *BaseTemplateProduct
}

func (t *MyDocumentTemplateProduct) GetPath() string {
    return "docs/mydocument.md"
}
```

### 3. Template Categories

Templates are categorized by path:

| Category | Paths | Used For |
|----------|-------|----------|
| **Init** | `root/`, `cmd/`, `internal/`, `apis/v1alpha1/`, `examples/provider/` | Project initialization |
| **API** | `apis/{group}/`, `internal/controller/{kind}/`, `examples/{group}/` | Adding managed resources |
| **Static** | `LICENSE`, `README.md` | Standalone files |

### Testing Your Templates

```bash
# Test auto-discovery
go test ./pkg/plugins/crossplane/v2/templates/ -v -run TestGetSupportedTypes

# Test full workflow
cd /tmp && mkdir test-provider && cd test-provider
/path/to/bin/crossplane-provider-gen init --domain=test.io --repo=github.com/test/provider
make generate && make build && make reviewable
```

## Commands

```bash
make build      # Build the generator
make test       # Run tests
make clean      # Clean build artifacts
```
