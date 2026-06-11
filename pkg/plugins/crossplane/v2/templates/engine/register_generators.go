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

package engine

import (
	"fmt"
	"strings"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
)

// The register files are regenerated in full from the project's resource list
// on every create/update, so they never need to be parsed and merged.

// baseSchemeAlias is the import alias for the always-present ProviderConfig API.
const baseSchemeAlias = "providerv1alpha1"

// apiGroupVersion is one (group, version) scheme registration in apis/register.go.
// Two kinds in the same group/version share a single scheme builder, so entries
// are keyed by group/version, not by kind.
type apiGroupVersion struct {
	Alias string // e.g. samplev1
	Path  string // e.g. github.com/example/provider-test/apis/sample/v1
}

// controllerPackage is one controller setup wired into internal/controller/register.go.
// Controllers are per-kind (each kind has its own internal/controller/<kind> package).
type controllerPackage struct {
	Path  string // import path, e.g. .../internal/controller/mytype
	Setup string // setup expression, e.g. mytype.SetupGated
}

// uniqueGroupVersions returns the base providerv1alpha1 scheme followed by one
// entry per distinct managed (group, version), in first-seen order.
func uniqueGroupVersions(repo string, resources []resource.Resource) []apiGroupVersion {
	groups := []apiGroupVersion{
		{Alias: baseSchemeAlias, Path: repo + "/apis/v1alpha1"},
	}
	seen := map[string]bool{}
	for _, res := range resources {
		if res.Group == "" {
			continue
		}
		key := res.Group + "/" + res.Version
		if seen[key] {
			continue
		}
		seen[key] = true
		groups = append(groups, apiGroupVersion{
			// ImportAlias sanitizes the group+version into a valid Go identifier
			// (e.g. strips '-'/'.'), matching kubebuilder's convention.
			Alias: res.ImportAlias(),
			Path:  fmt.Sprintf("%s/apis/%s/%s", repo, res.Group, res.Version),
		})
	}
	return groups
}

// controllerPackages returns the base config controller followed by one entry
// per distinct managed kind, in first-seen order.
func controllerPackages(repo string, resources []resource.Resource) []controllerPackage {
	controllers := []controllerPackage{
		{Path: repo + "/internal/controller/config", Setup: "config.Setup"},
	}
	seen := map[string]bool{}
	for _, res := range resources {
		pkg := strings.ToLower(res.Kind)
		if pkg == "" || seen[pkg] {
			continue
		}
		seen[pkg] = true
		controllers = append(controllers, controllerPackage{
			Path:  fmt.Sprintf("%s/internal/controller/%s", repo, pkg),
			Setup: pkg + ".SetupGated",
		})
	}
	return controllers
}

// APIRegisterGenerator renders apis/register.go from the full resource list.
type APIRegisterGenerator struct {
	machinery.TemplateMixin
	machinery.BoilerplateMixin

	ProviderName string
	Groups       []apiGroupVersion
}

var _ machinery.Template = &APIRegisterGenerator{}

// NewAPIRegisterGenerator builds the apis/register.go generator for the given resources.
func NewAPIRegisterGenerator(repo, providerName string, resources []resource.Resource) *APIRegisterGenerator {
	return &APIRegisterGenerator{
		ProviderName: providerName,
		Groups:       uniqueGroupVersions(repo, resources),
	}
}

func (f *APIRegisterGenerator) SetTemplateDefaults() error {
	f.Path = "apis/register.go"
	f.IfExistsAction = machinery.OverwriteFile
	f.TemplateBody = apiRegisterTemplate
	return nil
}

const apiRegisterTemplate = `{{ .Boilerplate }}

// Package apis contains Kubernetes API for the {{ .ProviderName }} provider.
package apis

import (
	"k8s.io/apimachinery/pkg/runtime"

{{- range .Groups }}
	{{ .Alias }} "{{ .Path }}"
{{- end }}
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes,
{{- range .Groups }}
		{{ .Alias }}.SchemeBuilder.AddToScheme,
{{- end }}
	)
}

// AddToSchemes may be used to add all resources defined in the project to a Scheme
var AddToSchemes runtime.SchemeBuilder

// AddToScheme adds all Resources to the Scheme
func AddToScheme(s *runtime.Scheme) error {
	return AddToSchemes.AddToScheme(s)
}
`

// ControllerRegisterGenerator renders internal/controller/register.go from the full resource list.
type ControllerRegisterGenerator struct {
	machinery.TemplateMixin
	machinery.BoilerplateMixin

	ProviderName string
	Controllers  []controllerPackage
}

var _ machinery.Template = &ControllerRegisterGenerator{}

// NewControllerRegisterGenerator builds the internal/controller/register.go generator.
func NewControllerRegisterGenerator(
	repo, providerName string, resources []resource.Resource,
) *ControllerRegisterGenerator {
	return &ControllerRegisterGenerator{
		ProviderName: providerName,
		Controllers:  controllerPackages(repo, resources),
	}
}

func (f *ControllerRegisterGenerator) SetTemplateDefaults() error {
	f.Path = "internal/controller/register.go"
	f.IfExistsAction = machinery.OverwriteFile
	f.TemplateBody = controllerRegisterTemplate
	return nil
}

const controllerRegisterTemplate = `{{ .Boilerplate }}

package controller

import (
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"

{{- range .Controllers }}
	"{{ .Path }}"
{{- end }}
)

// Setup creates all {{ .ProviderName }} controllers with the supplied logger and adds them to
// the supplied manager.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	for _, setup := range []func(ctrl.Manager, controller.Options) error{
{{- range .Controllers }}
		{{ .Setup }},
{{- end }}
	} {
		if err := setup(mgr, o); err != nil {
			return err
		}
	}
	return nil
}
`
