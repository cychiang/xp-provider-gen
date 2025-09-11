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

package cluster

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &ClusterDockerfile{}

// ClusterDockerfile scaffolds the cluster/images/provider/Dockerfile
type ClusterDockerfile struct {
	machinery.TemplateMixin
	machinery.BoilerplateMixin
	machinery.RepositoryMixin
	
	// ProviderName is the name extracted from the repository
	ProviderName string
}

// SetTemplateDefaults implements machinery.Template
func (f *ClusterDockerfile) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("cluster", "images", f.ProviderName, "Dockerfile")
	}

	f.TemplateBody = clusterDockerfileTemplate
	f.IfExistsAction = machinery.OverwriteFile

	return nil
}

const clusterDockerfileTemplate = `FROM gcr.io/distroless/static@sha256:d9f9472a8f4541368192d714a995eb1a99bab1f7071fc8bde261d7eda3b667d8

ARG TARGETOS
ARG TARGETARCH

ADD bin/$TARGETOS\_$TARGETARCH/provider /usr/local/bin/crossplane-{{ .ProviderName }}

USER 65532
ENTRYPOINT ["crossplane-{{ .ProviderName }}"]
`