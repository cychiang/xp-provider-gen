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

package providerconfig

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &ProviderConfigDoc{}

// ProviderConfigDoc scaffolds the api/<version>/doc.go file for ProviderConfig
type ProviderConfigDoc struct {
	machinery.TemplateMixin
	machinery.DomainMixin
	machinery.RepositoryMixin
	
	// Domain is the provider domain name (e.g., "example.com")
	Domain string
	// ProviderName is the provider name (extracted from repository)
	ProviderName string
}

// SetTemplateDefaults implements file.Template
func (f *ProviderConfigDoc) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("apis", "v1alpha1", "doc.go")
	}

	f.TemplateBody = providerConfigDocTemplate

	f.IfExistsAction = machinery.OverwriteFile

	return nil
}

const providerConfigDocTemplate = `// Package v1alpha1 contains the core resources of the {{ .ProviderName }} provider.
// +kubebuilder:object:generate=true
// +groupName={{ .Domain }}
// +versionName=v1alpha1
package v1alpha1
`