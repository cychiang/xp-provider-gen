/*
Copyright 2025 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package templates

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &ReadMe{}

// ReadMe scaffolds a README.md file for a Crossplane provider
type ReadMe struct {
	machinery.TemplateMixin
	machinery.ProjectNameMixin
}

// SetTemplateDefaults implements machinery.Template
func (f *ReadMe) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("README.md")
	}

	f.TemplateBody = readmeTemplate

	return nil
}

const readmeTemplate = `# {{ .ProjectName }}
A Crossplane provider for managing cloud resources.

## Overview

This provider is built using the [Crossplane](https://crossplane.io/) framework and follows the [provider development guide](https://docs.crossplane.io/contribute/provider-development-guide/).

## Getting Started

### Prerequisites

- Kubernetes cluster (1.29+)
- Crossplane installed in the cluster

### Installation

1. Install the provider:

` + "`" + `bash
kubectl apply -f https://raw.githubusercontent.com/crossplane-contrib/{{ .ProjectName }}/main/package/provider.yaml
` + "`" + `

2. Create a ProviderConfig:

` + "`" + `yaml
apiVersion: pkg.crossplane.io/v1
kind: Provider  
metadata:
  name: {{ .ProjectName }}
spec:
  package: crossplane/{{ .ProjectName }}:latest
` + "`" + `

## Development

### Building

` + "`" + `bash
# Build the provider binary
make build

# Build the docker image
make docker-build

# Run tests
make test
` + "`" + `

### Local Development

` + "`" + `bash
# Run the provider locally
make run
` + "`" + `

### Adding Resources

Use the Crossplane provider generator to add new managed resources:

` + "`" + `bash
crossplane-provider-gen create api --group=<group> --version=<version> --kind=<kind>
` + "`" + `

## Contributing

Contributions are welcome! Please read our [contributing guide](CONTRIBUTING.md) for details on our code of conduct and the process for submitting pull requests.

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.
`