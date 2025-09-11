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

package pkg

import (
	"path/filepath"
	"strings"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &CrossplanePackage{}

// CrossplanePackage scaffolds the package/crossplane.yaml file for Crossplane provider
type CrossplanePackage struct {
	machinery.TemplateMixin
	machinery.DomainMixin
	machinery.RepositoryMixin
	
	// ProviderName is the provider name (extracted from repository)
	ProviderName string
}

// SetTemplateDefaults implements file.Template
func (f *CrossplanePackage) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("package", "crossplane.yaml")
	}

	f.TemplateBody = crossplanePackageTemplate

	f.IfExistsAction = machinery.OverwriteFile

	// Extract provider name from repository
	if f.Repo != "" {
		parts := strings.Split(f.Repo, "/")
		if len(parts) > 0 {
			f.ProviderName = parts[len(parts)-1]
		}
	}
	if f.ProviderName == "" {
		f.ProviderName = "provider-example"
	}

	return nil
}

const crossplanePackageTemplate = `apiVersion: meta.pkg.crossplane.io/v1alpha1
kind: Provider
metadata:
  name: {{ .ProviderName }}
  annotations:
    meta.crossplane.io/maintainer: Crossplane Maintainers <info@crossplane.io>
    meta.crossplane.io/source: {{ .Repo }}
    meta.crossplane.io/license: Apache-2.0
    meta.crossplane.io/description: |
      {{ .ProviderName }} Crossplane provider.
`