package templates

import (
	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

func APIGroup(cfg config.Config) machinery.Template {
	return NewAPITemplate(cfg, "apis/%[group]/%[version]/groupversion_info.go", crossplaneGroupTemplate)
}

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
)`
