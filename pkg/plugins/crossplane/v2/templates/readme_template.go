package templates

import (
	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

func README(cfg config.Config) machinery.Template {
	return SimpleFile(cfg, "README.md", readmeTemplate)
}

const readmeTemplate = `# {{ .ProviderName }}

{{ .ProviderName }} is a Crossplane provider for managing external resources.

## Getting Started

This provider is built using the [Crossplane Provider Generator](https://github.com/crossplane/xp-kubebuilder-plugin).

### Prerequisites

- Go 1.24+
- Docker
- Kubernetes cluster

### Building

` + "```bash" + `
make build
` + "```" + `

### Running

` + "```bash" + `
make run
` + "```" + `

## Development

This provider follows Crossplane v2 patterns and uses crossplane-runtime v2.

For more information, see the [Crossplane Provider Development Guide](https://docs.crossplane.io/contribute/provider-development-guide/).`
