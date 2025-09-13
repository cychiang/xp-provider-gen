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
	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

// ProviderConfigTypes creates ProviderConfig types
func ProviderConfigTypes(cfg config.Config) machinery.Template {
	return SimpleFile(cfg, "apis/v1alpha1/types.go", providerConfigTypesTemplate)
}

const providerConfigTypesTemplate = `package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"github.com/crossplane/crossplane-runtime/apis/common/v1"
)

// ProviderConfig defines the configuration for {{ .ProviderName }}.
type ProviderConfig struct {
	metav1.TypeMeta   ` + "`json:\",inline\"`" + `
	metav1.ObjectMeta ` + "`json:\"metadata,omitempty\"`" + `
	
	Spec ProviderConfigSpec ` + "`json:\"spec\"`" + `
}

// ProviderConfigSpec defines the desired state of ProviderConfig
type ProviderConfigSpec struct {
	// TODO: Add configuration fields specific to your provider
	// Credentials SecretStoreConfigRef ` + "`json:\"credentials\"`" + `
}

// ProviderConfigStatus defines the observed state of ProviderConfig
type ProviderConfigStatus struct {
	v1.ProviderConfigStatus ` + "`json:\",inline\"`" + `
}

// +kubebuilder:object:root=true

// A ProviderConfig configures a {{ .ProviderName }} provider.
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,providerconfig}
type ProviderConfig struct {
	metav1.TypeMeta   ` + "`json:\",inline\"`" + `
	metav1.ObjectMeta ` + "`json:\"metadata,omitempty\"`" + `

	Spec   ProviderConfigSpec   ` + "`json:\"spec\"`" + `
	Status ProviderConfigStatus ` + "`json:\"status,omitempty\"`" + `
}

// +kubebuilder:object:root=true

// ProviderConfigList contains a list of ProviderConfig
type ProviderConfigList struct {
	metav1.TypeMeta ` + "`json:\",inline\"`" + `
	metav1.ListMeta ` + "`json:\"metadata,omitempty\"`" + `
	Items           []ProviderConfig ` + "`json:\"items\"`" + `
}`
