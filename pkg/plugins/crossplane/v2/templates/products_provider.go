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
)

// ProviderConfigTypesTemplateProduct implements ProviderConfig types
type ProviderConfigTypesTemplateProduct struct {
	*BaseTemplateProduct
	loader *TemplateLoader
}

func (t *ProviderConfigTypesTemplateProduct) SetTemplateDefaults() error {
	if t.Path == "" {
		t.Path = filepath.Join("apis", "v1alpha1", "types.go")
	}

	// Try to load from file first, fallback to embedded constant
	if t.loader == nil {
		t.loader = NewTemplateLoader()
	}

	if templateContent, err := t.loader.LoadTemplate("apis/v1alpha1/providerconfig_types.go.tmpl"); err == nil {
		t.TemplateBody = templateContent
	} else {
		// Fallback to embedded template for backward compatibility
		t.TemplateBody = providerConfigTypesTemplate
	}
	return nil
}

const providerConfigTypesTemplate = `{{ .Boilerplate }}

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	xpv2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"
)

// A ProviderConfigSpec defines the desired state of a ProviderConfig.
type ProviderConfigSpec struct {
	// Credentials required to authenticate to this provider.
	Credentials ProviderCredentials ` + "`" + `json:"credentials"` + "`" + `
}

// ProviderCredentials required to authenticate.
type ProviderCredentials struct {
	// Source of the provider credentials.
	Source xpv1.CredentialsSource ` + "`" + `json:"source"` + "`" + `

	xpv1.CommonCredentialSelectors ` + "`" + `json:",inline"` + "`" + `
}

// A ProviderConfigStatus reflects the observed state of a ProviderConfig.
type ProviderConfigStatus struct {
	xpv1.ProviderConfigStatus ` + "`" + `json:",inline"` + "`" + `
}

// +kubebuilder:object:root=true

// A ProviderConfig configures a {{ .ProviderName }} provider.
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="SECRET-NAME",type="string",JSONPath=".spec.credentials.secretRef.name",priority=1
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:resource:scope=Cluster,categories={crossplane,provider,{{ .ProviderName }}}
type ProviderConfig struct {
	metav1.TypeMeta   ` + "`" + `json:",inline"` + "`" + `
	metav1.ObjectMeta ` + "`" + `json:"metadata,omitempty"` + "`" + `

	Spec   ProviderConfigSpec   ` + "`" + `json:"spec"` + "`" + `
	Status ProviderConfigStatus ` + "`" + `json:"status,omitempty"` + "`" + `
}

// +kubebuilder:object:root=true

// ProviderConfigList contains a list of ProviderConfig.
type ProviderConfigList struct {
	metav1.TypeMeta ` + "`" + `json:",inline"` + "`" + `
	metav1.ListMeta ` + "`" + `json:"metadata,omitempty"` + "`" + `
	Items           []ProviderConfig ` + "`" + `json:"items"` + "`" + `
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion

// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="CONFIG-NAME",type="string",JSONPath=".providerConfigRef.name"
// +kubebuilder:printcolumn:name="RESOURCE-KIND",type="string",JSONPath=".resourceRef.kind"
// +kubebuilder:printcolumn:name="RESOURCE-NAME",type="string",JSONPath=".resourceRef.name"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,provider,{{ .ProviderName }}}
// A ProviderConfigUsage indicates that a resource is using a ProviderConfig.
type ProviderConfigUsage struct {
	metav1.TypeMeta   ` + "`" + `json:",inline"` + "`" + `
	metav1.ObjectMeta ` + "`" + `json:"metadata,omitempty"` + "`" + `

	xpv2.TypedProviderConfigUsage ` + "`" + `json:",inline"` + "`" + `
}

// +kubebuilder:object:root=true

// ProviderConfigUsageList contains a list of ProviderConfigUsage
type ProviderConfigUsageList struct {
	metav1.TypeMeta ` + "`" + `json:",inline"` + "`" + `
	metav1.ListMeta ` + "`" + `json:"metadata,omitempty"` + "`" + `
	Items           []ProviderConfigUsage ` + "`" + `json:"items"` + "`" + `
}

// +kubebuilder:object:root=true

// A ClusterProviderConfig configures a {{ .ProviderName }} provider.
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="SECRET-NAME",type="string",JSONPath=".spec.credentials.secretRef.name",priority=1
// +kubebuilder:resource:scope=Cluster,categories={crossplane,provider,{{ .ProviderName }}}
type ClusterProviderConfig struct {
	metav1.TypeMeta   ` + "`" + `json:",inline"` + "`" + `
	metav1.ObjectMeta ` + "`" + `json:"metadata,omitempty"` + "`" + `

	Spec   ProviderConfigSpec   ` + "`" + `json:"spec"` + "`" + `
	Status ProviderConfigStatus ` + "`" + `json:"status,omitempty"` + "`" + `
}

// +kubebuilder:object:root=true

// ClusterProviderConfigList contains a list of ProviderConfig.
type ClusterProviderConfigList struct {
	metav1.TypeMeta ` + "`" + `json:",inline"` + "`" + `
	metav1.ListMeta ` + "`" + `json:"metadata,omitempty"` + "`" + `
	Items           []ClusterProviderConfig ` + "`" + `json:"items"` + "`" + `
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion

// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="CONFIG-NAME",type="string",JSONPath=".providerConfigRef.name"
// +kubebuilder:printcolumn:name="RESOURCE-KIND",type="string",JSONPath=".resourceRef.kind"
// +kubebuilder:printcolumn:name="RESOURCE-NAME",type="string",JSONPath=".resourceRef.name"
// +kubebuilder:resource:scope=Cluster,categories={crossplane,provider,{{ .ProviderName }}}
// A ClusterProviderConfigUsage indicates that a resource is using a ClusterProviderConfig.
type ClusterProviderConfigUsage struct {
	metav1.TypeMeta   ` + "`" + `json:",inline"` + "`" + `
	metav1.ObjectMeta ` + "`" + `json:"metadata,omitempty"` + "`" + `

	xpv2.TypedProviderConfigUsage ` + "`" + `json:",inline"` + "`" + `
}

// +kubebuilder:object:root=true

// ClusterProviderConfigUsageList contains a list of ClusterProviderConfigUsage
type ClusterProviderConfigUsageList struct {
	metav1.TypeMeta ` + "`" + `json:",inline"` + "`" + `
	metav1.ListMeta ` + "`" + `json:"metadata,omitempty"` + "`" + `
	Items           []ClusterProviderConfigUsage ` + "`" + `json:"items"` + "`" + `
}`

// ProviderConfigRegisterTemplateProduct implements ProviderConfig registration
type ProviderConfigRegisterTemplateProduct struct {
	*BaseTemplateProduct
}

func (t *ProviderConfigRegisterTemplateProduct) SetTemplateDefaults() error {
	if t.Path == "" {
		t.Path = filepath.Join("apis", "v1alpha1", "register.go")
	}
	t.TemplateBody = providerConfigRegisterTemplate
	return nil
}

const providerConfigRegisterTemplate = `{{ .Boilerplate }}

package v1alpha1

import (
	"reflect"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

// Package type metadata.
const (
	Group   = "{{ .Domain }}"
	Version = "v1alpha1"
)

var (
	// SchemeGroupVersion is group version used to register these objects
	SchemeGroupVersion = schema.GroupVersion{Group: Group, Version: Version}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion}
)

// ProviderConfig type metadata.
var (
	ProviderConfigKind             = reflect.TypeOf(ProviderConfig{}).Name()
	ProviderConfigGroupKind        = schema.GroupKind{Group: Group, Kind: ProviderConfigKind}.String()
	ProviderConfigGroupVersionKind = SchemeGroupVersion.WithKind(ProviderConfigKind)
)

// ProviderConfigUsage type metadata.
var (
	ProviderConfigUsageKind             = reflect.TypeOf(ProviderConfigUsage{}).Name()
	ProviderConfigUsageGroupVersionKind = SchemeGroupVersion.WithKind(ProviderConfigUsageKind)

	ProviderConfigUsageListKind             = reflect.TypeOf(ProviderConfigUsageList{}).Name()
	ProviderConfigUsageListGroupVersionKind = SchemeGroupVersion.WithKind(ProviderConfigUsageListKind)
)

// ClusterProviderConfig type metadata
var (
	ClusterProviderConfigKind             = reflect.TypeOf(ClusterProviderConfig{}).Name()
	ClusterProviderConfigGroupKind        = schema.GroupKind{Group: Group, Kind: ClusterProviderConfigKind}.String()
	ClusterProviderConfigGroupVersionKind = SchemeGroupVersion.WithKind(ClusterProviderConfigKind)
)

// ClusterProviderConfigUsage type metadata.
var (
	ClusterProviderConfigUsageKind             = reflect.TypeOf(ClusterProviderConfigUsage{}).Name()
	ClusterProviderConfigUsageGroupVersionKind = SchemeGroupVersion.WithKind(ClusterProviderConfigUsageKind)

	ClusterProviderConfigUsageListKind             = reflect.TypeOf(ClusterProviderConfigUsageList{}).Name()
	ClusterProviderConfigUsageListGroupVersionKind = SchemeGroupVersion.WithKind(ClusterProviderConfigUsageListKind)
)

func init() {
	SchemeBuilder.Register(&ProviderConfig{}, &ProviderConfigList{})
	SchemeBuilder.Register(&ProviderConfigUsage{}, &ProviderConfigUsageList{})
	SchemeBuilder.Register(&ClusterProviderConfig{}, &ClusterProviderConfigList{})
	SchemeBuilder.Register(&ClusterProviderConfigUsage{}, &ClusterProviderConfigUsageList{})
}`

// CrossplanePackageTemplateProduct implements package/crossplane.yaml
type CrossplanePackageTemplateProduct struct {
	*BaseTemplateProduct
}

func (t *CrossplanePackageTemplateProduct) SetTemplateDefaults() error {
	if t.Path == "" {
		t.Path = filepath.Join("package", "crossplane.yaml")
	}
	t.TemplateBody = crossplanePackageTemplate
	return nil
}

const crossplanePackageTemplate = `apiVersion: meta.pkg.crossplane.io/v1alpha1
kind: Provider
metadata:
  name: {{ .ProviderName }}
  annotations:
    meta.crossplane.io/maintainer: {{ .ProviderName }} Maintainers <noreply@crossplane.io>
    meta.crossplane.io/source: {{ .Repo }}
    meta.crossplane.io/license: Apache-2.0
    meta.crossplane.io/description: |
      {{ .ProviderName }} is a Crossplane provider for {{ .ProviderName }}.
    meta.crossplane.io/readme: |
      This ` + "`" + `provider-{{ .ProviderName }}` + "`" + ` repository is the Crossplane infrastructure provider for
      {{ .ProviderName }}. The provider that is built from the source code in this repository can be
      installed into a Crossplane control plane and adds the following new functionality:

      * Custom Resource Definitions (CRDs) that model {{ .ProviderName }} infrastructure and services
      * Controllers to provision these resources in {{ .ProviderName }} based on the users desired state captured in CRDs they create
      * Implementations of Crossplane's portable resource abstractions, enabling {{ .ProviderName }} resources to fulfill a user's general need for cloud services

spec:
  controller:
    image: {{ .Repo }}:latest`

// ConfigControllerTemplateProduct implements config controller
type ConfigControllerTemplateProduct struct {
	*BaseTemplateProduct
	loader *TemplateLoader
}

func (t *ConfigControllerTemplateProduct) SetTemplateDefaults() error {
	if t.Path == "" {
		t.Path = filepath.Join("internal", "controller", "config", "config.go")
	}

	// Load from scaffolds file
	if t.loader == nil {
		t.loader = NewTemplateLoader()
	}

	if templateContent, err := t.loader.LoadTemplate("internal/controller/config/config.go.tmpl"); err == nil {
		t.TemplateBody = templateContent
	} else {
		// Fallback to embedded template for backward compatibility
		t.TemplateBody = configControllerTemplate
	}
	return nil
}

const configControllerTemplate = `{{ .Boilerplate }}

package config

import (
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/providerconfig"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"

	"{{ .Repo }}/apis/v1alpha1"
)

// Setup adds a controller that reconciles ProviderConfigs by accounting for
// their current usage.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	if err := setupNamespacedProviderConfig(mgr, o); err != nil {
		return err
	}
	return setupClusterProviderConfig(mgr, o)
}

func setupNamespacedProviderConfig(mgr ctrl.Manager, o controller.Options) error {
	name := providerconfig.ControllerName(v1alpha1.ProviderConfigGroupVersionKind.GroupKind().String())

	of := resource.ProviderConfigKinds{
		Config:    v1alpha1.ProviderConfigGroupVersionKind,
		Usage:     v1alpha1.ProviderConfigUsageGroupVersionKind,
		UsageList: v1alpha1.ProviderConfigUsageListGroupVersionKind,
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v1alpha1.ProviderConfig{}).
		Watches(&v1alpha1.ProviderConfigUsage{}, &resource.EnqueueRequestForProviderConfig{}).
		Complete(ratelimiter.NewReconciler(name, providerconfig.NewReconciler(mgr, of,
			providerconfig.WithLogger(o.Logger.WithValues("controller", name)),
			providerconfig.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name)))), o.GlobalRateLimiter))
}

func setupClusterProviderConfig(mgr ctrl.Manager, o controller.Options) error {
	name := providerconfig.ControllerName(v1alpha1.ClusterProviderConfigGroupVersionKind.GroupKind().String())

	of := resource.ProviderConfigKinds{
		Config:    v1alpha1.ClusterProviderConfigGroupVersionKind,
		Usage:     v1alpha1.ClusterProviderConfigUsageGroupVersionKind,
		UsageList: v1alpha1.ClusterProviderConfigUsageListGroupVersionKind,
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v1alpha1.ClusterProviderConfig{}).
		Watches(&v1alpha1.ClusterProviderConfigUsage{}, &resource.EnqueueRequestForProviderConfig{}).
		Complete(ratelimiter.NewReconciler(name, providerconfig.NewReconciler(mgr, of,
			providerconfig.WithLogger(o.Logger.WithValues("controller", name)),
			providerconfig.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name)))), o.GlobalRateLimiter))
}`

// ControllerRegisterTemplateProduct implements controller registration
type ControllerRegisterTemplateProduct struct {
	*BaseTemplateProduct
	loader *TemplateLoader
}

func (t *ControllerRegisterTemplateProduct) SetTemplateDefaults() error {
	if t.Path == "" {
		t.Path = filepath.Join("internal", "controller", "register.go")
	}

	// Load from scaffolds file
	if t.loader == nil {
		t.loader = NewTemplateLoader()
	}

	if templateContent, err := t.loader.LoadTemplate("internal/controller/register.go.tmpl"); err == nil {
		t.TemplateBody = templateContent
	} else {
		// Fallback to embedded template for backward compatibility
		t.TemplateBody = controllerRegisterTemplate
	}
	return nil
}

const controllerRegisterTemplate = `{{ .Boilerplate }}

package controller

import (
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"

	"{{ .Repo }}/internal/controller/config"
)

// Setup creates all {{ .ProviderName }} controllers with the supplied logger and adds them to
// the supplied manager.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	for _, setup := range []func(ctrl.Manager, controller.Options) error{
		config.Setup,
	} {
		if err := setup(mgr, o); err != nil {
			return err
		}
	}
	return nil
}`

// LicenseTemplateProduct implements LICENSE file
type LicenseTemplateProduct struct {
	*BaseTemplateProduct
}

func (t *LicenseTemplateProduct) SetTemplateDefaults() error {
	if t.Path == "" {
		t.Path = "LICENSE"
	}
	t.TemplateBody = licenseTemplate
	return nil
}

const licenseTemplate = `                                 Apache License
                           Version 2.0, January 2004
                        http://www.apache.org/licenses/

   TERMS AND CONDITIONS FOR USE, REPRODUCTION, AND DISTRIBUTION

   1. Definitions.

      "License" shall mean the terms and conditions for use, reproduction,
      and distribution as defined by Sections 1 through 9 of this document.

      "Licensor" shall mean the copyright owner or entity granting the License.

      "Legal Entity" shall mean the union of the acting entity and all
      other entities that control, are controlled by, or are under common
      control with that entity. For the purposes of this definition,
      "control" means (i) the power, direct or indirect, to cause the
      direction or management of such entity, whether by contract or
      otherwise, or (ii) ownership of fifty percent (50%) or more of the
      outstanding shares, or (iii) beneficial ownership of such entity.

      "You" (or "Your") shall mean an individual or Legal Entity
      exercising permissions granted by this License.

      "Source" form shall mean the preferred form for making modifications,
      including but not limited to software source code, documentation
      source, and configuration files.

      "Object" form shall mean any form resulting from mechanical
      transformation or translation of a Source form, including but
      not limited to compiled object code, generated documentation,
      and conversions to other media types.

      "Work" shall mean the work of authorship, whether in Source or
      Object form, made available under the License, as indicated by a
      copyright notice that is included in or attached to the work
      (which shall not include communications that are solely included in
      or attached to the work for informational purposes; the copyright
      notice is not itself a part of the work).

      "Derivative Works" shall mean any work, whether in Source or Object
      form, that is based upon (or derived from) the Work and for which the
      editorial revisions, annotations, elaborations, or other modifications
      represent, as a whole, an original work of authorship. For the purposes
      of this License, Derivative Works shall not include works that remain
      separable from, or merely link (or bind by name) to the interfaces of,
      the Work and derivative works thereof.

      "Contribution" shall mean any work of authorship, including
      the original version of the Work and any modifications or additions
      to that Work or Derivative Works thereof, that is intentionally
      submitted to Licensor for inclusion in the Work by the copyright owner
      or by an individual or Legal Entity authorized to submit on behalf of
      the copyright owner. For the purposes of this definition, "submitted"
      means any form of electronic, verbal, or written communication sent
      to the Licensor or its representatives, including but not limited to
      communication on electronic mailing lists, source code control
      systems, and issue tracking systems that are managed by, or on behalf
      of, the Licensor for the purpose of discussing and improving the Work,
      but excluding communication that is conspicuously marked or otherwise
      designated in writing by the copyright owner as "Not a Contribution."

      "Contributor" shall mean Licensor and any individual or Legal Entity
      on behalf of whom a Contribution has been received by Licensor and
      subsequently incorporated within the Work.

   2. Grant of Copyright License. Subject to the terms and conditions of
      this License, each Contributor hereby grants to You a perpetual,
      worldwide, non-exclusive, no-charge, royalty-free, irrevocable
      copyright license to use, reproduce, modify, distribute, and prepare
      Derivative Works of, publicly display, publicly perform, sublicense,
      and distribute the Work and such Derivative Works in Source or Object
      form.

   3. Grant of Patent License. Subject to the terms and conditions of
      this License, each Contributor hereby grants to You a perpetual,
      worldwide, non-exclusive, no-charge, royalty-free, irrevocable
      (except as stated in this section) patent license to make, have made,
      use, offer to sell, sell, import, and otherwise transfer the Work,
      where such license applies only to those patent claims licensable
      by such Contributor that are necessarily infringed by their
      Contribution(s) alone or by combination of their Contribution(s)
      with the Work to which such Contribution(s) was submitted. If You
      institute patent litigation against any entity (including a
      cross-claim or counterclaim in a lawsuit) alleging that the Work
      or a Contribution incorporated within the Work constitutes direct
      or contributory patent infringement, then any patent licenses
      granted to You under this License for that Work shall terminate
      as of the date such litigation is filed.

   4. Redistribution. You may reproduce and distribute copies of the
      Work or Derivative Works thereof in any medium, with or without
      modifications, and in Source or Object form, provided that You
      meet the following conditions:

      (a) You must give any other recipients of the Work or
          Derivative Works a copy of this License; and

      (b) You must cause any modified files to carry prominent notices
          stating that You changed the files; and

      (c) You must retain, in the Source form of any Derivative Works
          that You distribute, all copyright, trademark, patent,
          attribution and other notices from the Source form of the Work,
          excluding those notices that do not pertain to any part of
          the Derivative Works; and

      (d) If the Work includes a "NOTICE" text file as part of its
          distribution, then any Derivative Works that You distribute must
          include a readable copy of the attribution notices contained
          within such NOTICE file, excluding those notices that do not
          pertain to any part of the Derivative Works, in at least one
          of the following places: within a NOTICE text file distributed
          as part of the Derivative Works; within the Source form or
          documentation, if provided along with the Derivative Works; or,
          within a display generated by the Derivative Works, if and
          wherever such third-party notices normally appear. The contents
          of the NOTICE file are for informational purposes only and
          do not modify the License. You may add Your own attribution
          notices within Derivative Works that You distribute, alongside
          or as an addendum to the NOTICE text from the Work, provided
          that such additional attribution notices cannot be construed
          as modifying the License.

      You may add Your own copyright notice to Your modifications and
      may provide additional or different license terms and conditions
      for use, reproduction, or distribution of Your modifications, or
      for any such Derivative Works as a whole, provided Your use,
      reproduction, and distribution of the Work otherwise complies with
      the conditions stated in this License.

   5. Submission of Contributions. Unless You explicitly state otherwise,
      any Contribution intentionally submitted for inclusion in the Work
      by You to the Licensor shall be under the terms and conditions of
      this License, without any additional terms or conditions.
      Notwithstanding the above, nothing herein shall supersede or modify
      the terms of any separate license agreement you may have executed
      with Licensor regarding such Contributions.

   6. Trademarks. This License does not grant permission to use the trade
      names, trademarks, service marks, or product names of the Licensor,
      except as required for reasonable and customary use in describing the
      origin of the Work and reproducing the content of the NOTICE file.

   7. Disclaimer of Warranty. Unless required by applicable law or
      agreed to in writing, Licensor provides the Work (and each
      Contributor provides its Contributions) on an "AS IS" BASIS,
      WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
      implied, including, without limitation, any warranties or conditions
      of TITLE, NON-INFRINGEMENT, MERCHANTABILITY, or FITNESS FOR A
      PARTICULAR PURPOSE. You are solely responsible for determining the
      appropriateness of using or redistributing the Work and assume any
      risks associated with Your exercise of permissions under this License.

   8. Limitation of Liability. In no event and under no legal theory,
      whether in tort (including negligence), contract, or otherwise,
      unless required by applicable law (such as deliberate and grossly
      negligent acts) or agreed to in writing, shall any Contributor be
      liable to You for damages, including any direct, indirect, special,
      incidental, or consequential damages of any character arising as a
      result of this License or out of the use or inability to use the
      Work (including but not limited to damages for loss of goodwill,
      work stoppage, computer failure or malfunction, or any and all
      other commercial damages or losses), even if such Contributor
      has been advised of the possibility of such damages.

   9. Accepting Warranty or Support. You may choose to offer, and to
      charge a fee for, warranty, support, indemnity or other liability
      obligations and/or rights consistent with this License. However, in
      accepting such obligations, You may act only on Your own behalf and on
      Your sole responsibility, not on behalf of any other Contributor, and
      only if You agree to indemnify, defend, and hold each Contributor
      harmless for any liability incurred by, or claims asserted against,
      such Contributor by reason of your accepting any such warranty or support.

   END OF TERMS AND CONDITIONS`

// ClusterDockerfileTemplateProduct implements cluster Dockerfile generation
type ClusterDockerfileTemplateProduct struct {
	*BaseTemplateProduct
	loader *TemplateLoader
}

func (t *ClusterDockerfileTemplateProduct) SetTemplateDefaults() error {
	if t.Path == "" {
		providerName := extractProviderName(t.Repo)
		t.Path = filepath.Join("cluster", "images", providerName, "Dockerfile")
	}

	// Load from scaffolds file
	if t.loader == nil {
		t.loader = NewTemplateLoader()
	}

	if templateContent, err := t.loader.LoadTemplate("cluster/images/IMAGE_NAME/Dockerfile.tmpl"); err == nil {
		t.TemplateBody = templateContent
	} else {
		// Fallback to embedded template for backward compatibility
		t.TemplateBody = clusterDockerfileTemplate
	}
	return nil
}

const clusterDockerfileTemplate = `FROM gcr.io/distroless/static@sha256:d9f9472a8f4541368192d714a995eb1a99bab1f7071fc8bde261d7eda3b667d8

ARG TARGETOS
ARG TARGETARCH

ADD bin/$TARGETOS\_$TARGETARCH/provider /usr/local/bin/crossplane-{{ .ProviderName }}-provider

USER 65532
ENTRYPOINT ["crossplane-{{ .ProviderName }}-provider"]`

// DocGoTemplateProduct implements doc.go file generation
type DocGoTemplateProduct struct {
	*BaseTemplateProduct
	loader *TemplateLoader
}

func (t *DocGoTemplateProduct) SetTemplateDefaults() error {
	if t.Path == "" {
		t.Path = filepath.Join("apis", "v1alpha1", "doc.go")
	}

	// Load from scaffolds file
	if t.loader == nil {
		t.loader = NewTemplateLoader()
	}

	if templateContent, err := t.loader.LoadTemplate("apis/doc.go.tmpl"); err == nil {
		t.TemplateBody = templateContent
	} else {
		// Fallback to embedded template for backward compatibility
		t.TemplateBody = docGoTemplate
	}
	return nil
}

const docGoTemplate = `{{ .Boilerplate }}

// Package v1alpha1 contains the core resources of the {{ .ProviderName }} provider.
// +kubebuilder:object:generate=true
// +groupName={{ .Domain }}
// +versionName=v1alpha1
package v1alpha1`

// ClusterMakefileTemplateProduct implements cluster Makefile generation
type ClusterMakefileTemplateProduct struct {
	*BaseTemplateProduct
	loader *TemplateLoader
}

func (t *ClusterMakefileTemplateProduct) SetTemplateDefaults() error {
	if t.Path == "" {
		providerName := extractProviderName(t.Repo)
		t.Path = filepath.Join("cluster", "images", providerName, "Makefile")
	}

	// Load from scaffolds file
	if t.loader == nil {
		t.loader = NewTemplateLoader()
	}

	if templateContent, err := t.loader.LoadTemplate("cluster/images/IMAGE_NAME/Makefile.tmpl"); err == nil {
		t.TemplateBody = templateContent
	} else {
		// Fallback to embedded template for backward compatibility
		t.TemplateBody = clusterMakefileTemplate
	}
	return nil
}

const clusterMakefileTemplate = `# ====================================================================================
# Setup Project

include ../../../build/makelib/common.mk

# ====================================================================================
#  Options

include ../../../build/makelib/imagelight.mk

# ====================================================================================
# Targets

img.build:
	@$(INFO) docker build $(IMAGE)
	@$(MAKE) BUILD_ARGS="--load" img.build.shared
	@$(OK) docker build $(IMAGE)

img.publish:
	@$(INFO) Skipping image publish for $(IMAGE)
	@echo Publish is deferred to xpkg machinery
	@$(OK) Image publish skipped for $(IMAGE)

img.build.shared:
	@cp Dockerfile $(IMAGE_TEMP_DIR) || $(FAIL)
	@cp -r $(OUTPUT_DIR)/bin/ $(IMAGE_TEMP_DIR)/bin || $(FAIL)
	@docker buildx build $(BUILD_ARGS) \
		--platform $(IMAGE_PLATFORMS) \
		-t $(IMAGE) \
		$(IMAGE_TEMP_DIR) || $(FAIL)

img.promote:
	@$(INFO) Skipping image promotion from $(FROM_IMAGE) to $(TO_IMAGE)
	@echo Promote is deferred to xpkg machinery
	@$(OK) Image promotion skipped for $(FROM_IMAGE) to $(TO_IMAGE)`

// VersionGoTemplateProduct implements version.go generation
type VersionGoTemplateProduct struct {
	*BaseTemplateProduct
}

func (t *VersionGoTemplateProduct) SetTemplateDefaults() error {
	if t.Path == "" {
		t.Path = filepath.Join("internal", "version", "version.go")
	}
	t.TemplateBody = versionGoTemplate
	return nil
}

const versionGoTemplate = `{{ .Boilerplate }}

// Package version contains the version of this repo
package version

// Version will be overridden with the current version at build time using the -X linker flag
var Version = "0.0.0"`