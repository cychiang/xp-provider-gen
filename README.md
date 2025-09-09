# Kubebuilder Crossplane Plugin

A kubebuilder plugin that generates Crossplane provider projects and managed resources.

## Overview

This plugin extends kubebuilder to support Crossplane-specific patterns:
- Bootstrap complete Crossplane provider projects
- Generate managed resource APIs with crossplane-runtime integration
- Scaffold controllers following Crossplane best practices
- Create provider-specific configurations and build systems

## Status

🏗️ **Skeleton Complete** - Plugin interfaces implemented, scaffolding logic pending

- ✅ Plugin architecture with kubebuilder v4.8.0 SDK
- ✅ All subcommands (init, create api, create webhook, edit)  
- ✅ Proper kubebuilder integration
- ⏳ Template implementation needed

## Quick Start

### Prerequisites
- Go 1.24+
- Kubebuilder v4.8.0+

### Development
```bash
git clone https://github.com/crossplane/xp-kubebuilder-plugin
cd xp-kubebuilder-plugin
make dev-setup
make build
```

### Usage (Future)
```bash
# Initialize provider project
kubebuilder init --plugins=crossplane.go.kubebuilder.io/v1 \
  --provider-name=mycloud

# Create managed resource API
kubebuilder create api --plugins=crossplane.go.kubebuilder.io/v1 \
  --group=compute --version=v1alpha1 --kind=Instance \
  --provider-type=aws
```

## Commands

| Command | Description |
|---------|-------------|
| `init` | Bootstrap Crossplane provider project |
| `create api` | Generate managed resource with controller |
| `create webhook` | Create webhooks (rarely needed) |
| `edit` | Modify project configuration |

## Planned Features

### Init Command
- Provider project structure with proper dependencies
- ProviderConfig CRD and controller
- Crossplane-specific Makefile and build system

### Create API Command
- Managed resource types with Crossplane patterns
- Controllers using crossplane-runtime
- External client interfaces for cloud APIs
- Support for AWS, GCP, Azure, and custom providers

### Generated Structure
```
my-provider/
├── main.go                      # Controller manager
├── go.mod                       # crossplane-runtime deps
├── Makefile                     # Package build targets
├── apis/v1alpha1/
│   └── providerconfig_types.go  # Provider configuration
├── apis/compute/v1alpha1/
│   └── instance_types.go        # Managed resources
├── internal/controller/
│   └── compute/instance.go      # Resource controllers
└── package/                     # Crossplane metadata
```

## Implementation Roadmap

### Phase 1: Core (High Priority)
- [ ] Init subcommand implementation
- [ ] Create API subcommand with Crossplane patterns
- [ ] Basic templates and scaffolding

### Phase 2: Enhancement (Medium Priority)
- [ ] Provider-specific templates (AWS, GCP, Azure)
- [ ] Enhanced template system
- [ ] Code generation integration

### Phase 3: Polish (Lower Priority)
- [ ] Improved CLI experience and validation
- [ ] Documentation generation
- [ ] Testing utilities

## Development

```bash
# Build and test
make build
make test
make validate

# Plugin info
make plugin-info
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

This plugin needs implementation of scaffolding templates:

1. Implement subcommand scaffolding logic
2. Create Crossplane-specific file templates
3. Add comprehensive testing
4. Improve documentation

## Resources

- [Kubebuilder Plugin Development](https://book.kubebuilder.io/plugins/extending)
- [Crossplane Provider Development](https://docs.crossplane.io/contribute/provider-development-guide/)
- [Crossplane Runtime](https://github.com/crossplane/crossplane-runtime)
- [Provider Template](https://github.com/crossplane/provider-template)

---

**Next Steps:** Implement Phase 1 scaffolding, starting with the init subcommand and create api functionality.