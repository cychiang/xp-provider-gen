/*
Copyright 2025 The Kubernetes Authors.

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

package api

import (
	log "log/slog"
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &CrossplaneGroup{}

// CrossplaneGroup scaffolds the groupversion_info.go file for Crossplane providers
type CrossplaneGroup struct {
	machinery.TemplateMixin
	machinery.MultiGroupMixin
	machinery.BoilerplateMixin
	machinery.ResourceMixin
}

// SetTemplateDefaults implements machinery.Template
func (f *CrossplaneGroup) SetTemplateDefaults() error {
	if f.Path == "" {
		// Crossplane providers always use multi-group layout: apis/${group}/${version}/
		if f.Resource.Group != "" {
			f.Path = filepath.Join("apis", "%[group]", "%[version]", "groupversion_info.go")
		} else {
			// Fallback for resources without group
			f.Path = filepath.Join("apis", "%[version]", "groupversion_info.go")
		}
	}

	f.Path = f.Resource.Replacer().Replace(f.Path)
	log.Info(f.Path)
	f.TemplateBody = crossplaneGroupTemplate

	return nil
}

//nolint:lll
const crossplaneGroupTemplate = `{{ .Boilerplate }}

// Package {{ .Resource.Version }} contains the {{ .Resource.Version }} group {{ .Resource.Group }} resources of the Crossplane provider.
// +kubebuilder:object:generate=true
// +groupName={{ .Resource.QualifiedGroup }}
// +versionName={{ .Resource.Version }}
package {{ .Resource.Version }}

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

// Package type metadata.
const (
	Group   = "{{ .Resource.QualifiedGroup }}"
	Version = "{{ .Resource.Version }}"
)

var (
	// SchemeGroupVersion is group version used to register these objects
	SchemeGroupVersion = schema.GroupVersion{Group: Group, Version: Version}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion}
)
`