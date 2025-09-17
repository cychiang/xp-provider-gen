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

The auto-discovery system provides different levels of complexity based on your needs:

#### **For Basic Templates (0 Steps)** ✅
Just create your template file - it's automatically discovered and available:

```bash
# Example: Add a new Go utility template
echo 'package utils\n\n// {{ .Resource.Kind }}Helper...' > pkg/plugins/crossplane/v2/templates/scaffolds/internal/utils/helper.go.tmpl
```

**Automatic features:**
- ✅ Runtime discovery and registration
- ✅ Auto-categorization (init/api/static based on path)
- ✅ Template type generation (`InternalUtilsHelperGoType`)
- ✅ Works immediately if pattern matches existing ones

#### **For New Template Types (1-2 Steps)**
If your template introduces a new pattern:

**Step 1:** Add the template file ✅ (Auto-discovered)

**Step 2:** Add pattern matching in `pkg/plugins/crossplane/v2/templates/builders.go`:
```go
case strings.Contains(typeStr, "utils"):
    product = &UtilsTemplateProduct{BaseTemplateProduct: NewBaseTemplateProduct(b.templateType)}
```

#### **For Convenience Methods (2-3 Steps)**
To add factory convenience methods:

**Step 1:** Add template file ✅ (Auto-discovered)

**Step 2:** Add pattern matching ✅ (If needed)

**Step 3:** Add convenience method in `factory.go`:
```go
func (f *CrossplaneTemplateFactory) Utils() (TemplateProduct, error) {
    templateType, err := f.FindTemplateTypeByPath("utils")
    if err != nil {
        return nil, err
    }
    return f.CreateInitTemplate(templateType)
}
```

#### **Template Categories**

Templates are automatically categorized by path:

| Category | Path Patterns | Use Case |
|----------|---------------|----------|
| **Init** | `root/`, `cmd/`, `internal/`, etc. | Project initialization |
| **API** | `apis/group/version/`, `internal/controller/kind/`, `examples/group/` | Creating managed resources |
| **Static** | `LICENSE` | Standalone files |

#### **What's Automatic** 🚀

1. **Discovery** - Scans all `.tmpl` files at runtime
2. **Type Generation** - `test/sample.md.tmpl` → `TestSampleMdType`
3. **Categorization** - Auto-assigns init/api/static category
4. **Registration** - Templates registered in appropriate factory
5. **Path Lookup** - `FindTemplateTypeByPath("sample")` works automatically

**Before:** 4 manual steps required
**After:** 0-3 steps depending on complexity
**Most common case:** **0 steps!** 🎉

#### **Development Workflow**

1. **Test template discovery:**
   ```bash
   go test ./pkg/plugins/crossplane/v2/templates/ -v -run TestCrossplaneTemplateFactory_GetSupportedTypes
   ```

2. **Verify complete workflow:**
   ```bash
   # Test in temp directory
   cd /tmp && mkdir test-provider && cd test-provider
   /path/to/crossplane-provider-gen init --domain=test.io --repo=github.com/test/provider
   /path/to/crossplane-provider-gen create api --group=compute --version=v1alpha1 --kind=Instance
   make generate && make build && make reviewable
   ```

3. **Debug template types:**
   Templates are auto-generated with naming pattern: `{Path}Type`
   - `root/go.mod.tmpl` → `RootGoModType`
   - `apis/group/version/types.go.tmpl` → `ApisGroupVersionTypesGoType`
   - `examples/group/kind.yaml.tmpl` → `ExamplesGroupKindYamlType`

## Build Commands

```bash
make build       # Build provider binary
make test        # Run tests
make reviewable  # Run all quality checks
make submodules  # Update build system
```