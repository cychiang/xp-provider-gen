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

var _ machinery.Template = &CrossplaneTypes{}

// CrossplaneTypes scaffolds the file that defines Crossplane managed resource types
type CrossplaneTypes struct {
	machinery.TemplateMixin
	machinery.MultiGroupMixin
	machinery.BoilerplateMixin
	machinery.ResourceMixin

	// Crossplane-specific fields
	ProviderType string
	ExternalName string
	Force        bool
}

// SetTemplateDefaults implements machinery.Template
func (f *CrossplaneTypes) SetTemplateDefaults() error {
	if f.Path == "" {
		// Crossplane providers always use multi-group layout: apis/${group}/${version}/
		if f.Resource.Group != "" {
			f.Path = filepath.Join("apis", "%[group]", "%[version]", "%[kind]_types.go")
		} else {
			// Fallback for resources without group
			f.Path = filepath.Join("apis", "%[version]", "%[kind]_types.go")
		}
	}

	f.Path = f.Resource.Replacer().Replace(f.Path)
	log.Info(f.Path)

	f.TemplateBody = crossplaneTypesTemplate

	if f.Force {
		f.IfExistsAction = machinery.OverwriteFile
	} else {
		f.IfExistsAction = machinery.Error
	}

	return nil
}

//nolint:lll
const crossplaneTypesTemplate = `{{ .Boilerplate }}

package {{ .Resource.Version }}

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	xpv2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"
)

// {{ .Resource.Kind }}Parameters are the configurable fields of a {{ .Resource.Kind }}.
type {{ .Resource.Kind }}Parameters struct {
	// TODO: Add fields for configuring the external resource.
	// These fields will be used to configure the desired state
	// of the external resource via the provider's API.
	ConfigurableField string ` + "`" + `json:"configurableField"` + "`" + `
}

// {{ .Resource.Kind }}Observation are the observable fields of a {{ .Resource.Kind }}.
type {{ .Resource.Kind }}Observation struct {
	// TODO: Add fields that will be observed from the external resource.
	// These fields represent the current state of the external resource
	// and will be populated by the controller during reconciliation.
	ConfigurableField string ` + "`" + `json:"configurableField"` + "`" + `
	ObservableField   string ` + "`" + `json:"observableField,omitempty"` + "`" + `
}

// A {{ .Resource.Kind }}Spec defines the desired state of a {{ .Resource.Kind }}.
type {{ .Resource.Kind }}Spec struct {
	xpv2.ManagedResourceSpec ` + "`" + `json:",inline"` + "`" + `
	ForProvider              {{ .Resource.Kind }}Parameters ` + "`" + `json:"forProvider"` + "`" + `
}

// A {{ .Resource.Kind }}Status represents the observed state of a {{ .Resource.Kind }}.
type {{ .Resource.Kind }}Status struct {
	xpv1.ResourceStatus ` + "`" + `json:",inline"` + "`" + `
	AtProvider          {{ .Resource.Kind }}Observation ` + "`" + `json:"atProvider,omitempty"` + "`" + `
}

// +kubebuilder:object:root=true

// A {{ .Resource.Kind }} is a managed resource.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed}
type {{ .Resource.Kind }} struct {
	metav1.TypeMeta   ` + "`" + `json:",inline"` + "`" + `
	metav1.ObjectMeta ` + "`" + `json:"metadata,omitempty"` + "`" + `

	Spec   {{ .Resource.Kind }}Spec   ` + "`" + `json:"spec"` + "`" + `
	Status {{ .Resource.Kind }}Status ` + "`" + `json:"status,omitempty"` + "`" + `
}

// +kubebuilder:object:root=true

// {{ .Resource.Kind }}List contains a list of {{ .Resource.Kind }}
type {{ .Resource.Kind }}List struct {
	metav1.TypeMeta ` + "`" + `json:",inline"` + "`" + `
	metav1.ListMeta ` + "`" + `json:"metadata,omitempty"` + "`" + `
	Items           []{{ .Resource.Kind }} ` + "`" + `json:"items"` + "`" + `
}

// {{ .Resource.Kind }} type metadata.
var (
	{{ .Resource.Kind }}Kind             = reflect.TypeOf({{ .Resource.Kind }}{}).Name()
	{{ .Resource.Kind }}GroupKind        = schema.GroupKind{Group: Group, Kind: {{ .Resource.Kind }}Kind}.String()
	{{ .Resource.Kind }}KindAPIVersion   = {{ .Resource.Kind }}Kind + "." + SchemeGroupVersion.String()
	{{ .Resource.Kind }}GroupVersionKind = SchemeGroupVersion.WithKind({{ .Resource.Kind }}Kind)
)

func init() {
	SchemeBuilder.Register(&{{ .Resource.Kind }}{}, &{{ .Resource.Kind }}List{})
}
`