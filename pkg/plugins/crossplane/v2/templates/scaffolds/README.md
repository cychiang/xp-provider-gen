# Crossplane Provider Templates

This directory contains template files used by the Crossplane Kubebuilder plugin to generate new provider projects. The structure mirrors the target project layout so developers can easily understand where templates will be placed in the generated project.

## Directory Structure

```
templates/
├── root/                      # Root-level files (go.mod, Makefile, README.md, etc.)
├── cmd/provider/              # Main provider executable
├── apis/                      # API definitions
│   ├── group/version/         # Group and version specific templates
│   └── v1alpha1/              # Provider config API templates
├── internal/
│   ├── controller/            # Controller implementations
│   │   └── kind/              # Resource-specific controllers
│   └── version/               # Version information
├── cluster/
│   ├── images/provider/       # Container build files
│   └── examples/              # Example configurations
└── package/                   # Crossplane package definitions
```

## Template File Naming

- All template files use the `.tmpl` extension
- File names should match the target file name (e.g., `go.mod.tmpl` → `go.mod`)
- Directory structure mirrors the target project structure

## Template Variables

Templates use Go's text/template syntax with the following available variables:

### Common Variables
- `{{ .Repo }}` - Repository URL (e.g., `github.com/example/provider-aws`)
- `{{ .Domain }}` - API domain (e.g., `aws.example.com`)
- `{{ .ProviderName }}` - Provider name (e.g., `provider-aws`)
- `{{ .Boilerplate }}` - Standard license header

### Resource-Specific Variables
- `{{ .Resource.Group }}` - API group (e.g., `compute`)
- `{{ .Resource.Version }}` - API version (e.g., `v1alpha1`)
- `{{ .Resource.Kind }}` - Resource kind (e.g., `Instance`)
- `{{ .Resource.QualifiedGroup }}` - Fully qualified group (e.g., `compute.aws.example.com`)

## Adding New Templates

**ZERO STEPS!** - Just create your template file:

```bash
# Simply create your template file in the appropriate directory - that's it!
echo "your template content..." > pkg/plugins/crossplane/v2/templates/scaffolds/category/your-template.tmpl
```

The system **automatically discovers and registers** all `.tmpl` files at runtime:
- ✅ Auto-detects template category (init/api/static) from file path
- ✅ Generates TemplateType constants automatically
- ✅ Registers with appropriate factory automatically
- ✅ No manual registration or generation commands needed

**Reduced from 4 manual steps to 0 steps!** 🚀

## Migration from Embedded Templates

The legacy system used embedded Go string constants. This new system:

1. **Improves maintainability** - Templates are separate files, easier to edit
2. **Better IDE support** - Syntax highlighting and validation for template content
3. **Clear organization** - Directory structure shows target project layout
4. **Version control friendly** - Template changes are easier to review
5. **Developer friendly** - Easy to understand where files will be generated

## Usage

Templates are loaded using Go's `embed.FS` system:

```go
// Create file-based factory
factory := templates.NewFileBasedFactory(config)

// Create template
template, err := factory.CreateInitTemplate(templates.GoModTemplateType)
if err != nil {
    return err
}

// Set defaults and render
err = template.SetTemplateDefaults()
if err != nil {
    return err
}
```

## Template Development Guidelines

1. **Keep templates focused** - Each template should generate one specific file
2. **Use clear variable names** - Make template intent obvious
3. **Include helpful comments** - Explain complex template logic
4. **Test thoroughly** - Verify templates generate valid, working code
5. **Follow Go conventions** - Generated Go code should be idiomatic
6. **Include TODO markers** - Help developers know what to customize

## Benefits of File-Based Templates

### For Template Authors
- **Better editing experience** - Syntax highlighting, validation
- **Easier debugging** - Clear separation between logic and content
- **Version control clarity** - Template changes are easily visible

### For Plugin Maintainers
- **Reduced code duplication** - No need to escape strings and manage Go constants
- **Cleaner code organization** - Logic separate from template content
- **Easier testing** - Templates can be tested independently

### For Plugin Users
- **Transparent generation** - Easy to see what files will be created
- **Customization friendly** - Can understand and modify templates if needed
- **Learning aid** - Template structure teaches Crossplane provider patterns