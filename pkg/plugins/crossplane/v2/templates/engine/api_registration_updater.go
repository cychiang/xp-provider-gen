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
	"path/filepath"
	"regexp"
	"strings"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"

	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/core"
)

var _ machinery.Template = &APIRegistrationUpdater{}

// APIRegistrationUpdater updates the apis/register.go file to include new API group imports and registrations.
type APIRegistrationUpdater struct {
	machinery.TemplateMixin
	machinery.ResourceMixin
	machinery.RepositoryMixin
	machinery.BoilerplateMixin

	// Force overwrite existing content
	Force bool

	// ProviderName is the name extracted from the repository
	ProviderName string
}

// SetTemplateDefaults implements machinery.Template.
func (f *APIRegistrationUpdater) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("apis", "register.go")
	}

	f.TemplateBody = ""
	// Always overwrite to update with new API registration
	f.IfExistsAction = machinery.OverwriteFile

	return nil
}

// GetBody implements machinery.Template to allow dynamic content generation.
func (f *APIRegistrationUpdater) GetBody() string {
	existingImports, existingRegistrations := f.parseExistingContent()
	newImport, newRegistration := f.generateNewAPIEntries()
	updatedImports := f.addImportIfNeeded(existingImports, newImport)
	updatedRegistrations := f.addRegistrationIfNeeded(existingRegistrations, newRegistration)
	return f.buildTemplate(updatedImports, updatedRegistrations)
}

func (f *APIRegistrationUpdater) generateNewAPIEntries() (string, string) {
	if f.Resource.Group == "" || f.Resource.Group == "v1alpha1" {
		return "", ""
	}

	newImport := fmt.Sprintf(`%s%s "%s/apis/%s/%s"`,
		f.Resource.Group, f.Resource.Version, f.Repo, f.Resource.Group, f.Resource.Version)
	newRegistration := fmt.Sprintf("%s%s.SchemeBuilder.AddToScheme", f.Resource.Group, f.Resource.Version)

	return newImport, newRegistration
}

func (f *APIRegistrationUpdater) addImportIfNeeded(existingImports []string, newImport string) []string {
	if newImport == "" {
		return existingImports
	}

	importPath := fmt.Sprintf("/apis/%s/%s", f.Resource.Group, f.Resource.Version)
	for _, existing := range existingImports {
		if strings.Contains(existing, importPath) {
			return existingImports
		}
	}

	return append(existingImports, newImport)
}

func (f *APIRegistrationUpdater) addRegistrationIfNeeded(
	existingRegistrations []string, newRegistration string,
) []string {
	if newRegistration == "" {
		return existingRegistrations
	}

	for _, existing := range existingRegistrations {
		if existing == newRegistration {
			return existingRegistrations
		}
	}

	return append(existingRegistrations, newRegistration)
}

func (f *APIRegistrationUpdater) buildTemplate(imports, registrations []string) string {
	var importsBuilder strings.Builder
	for _, imp := range imports {
		importsBuilder.WriteString(fmt.Sprintf("\t%s\n", imp))
	}

	var registrationsBuilder strings.Builder
	for _, reg := range registrations {
		registrationsBuilder.WriteString(fmt.Sprintf("\t\t%s,\n", reg))
	}

	return fmt.Sprintf(`{{ .Boilerplate }}

// Package apis contains Kubernetes API for the %s provider.
package apis

import (
	"k8s.io/apimachinery/pkg/runtime"

%s)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes,
%s	)
}

// AddToSchemes may be used to add all resources defined in the project to a Scheme
var AddToSchemes runtime.SchemeBuilder

// AddToScheme adds all Resources to the Scheme
func AddToScheme(s *runtime.Scheme) error {
	return AddToSchemes.AddToScheme(s)
}
`, f.ProviderName, importsBuilder.String(), registrationsBuilder.String())
}

// parseExistingContent reads and parses the existing register.go file to extract API imports and registrations.
func (f *APIRegistrationUpdater) parseExistingContent() ([]string, []string) {
	config := core.NewFileParserBuilder().
		AddImportSection("imports", "import (", ")",
			regexp.MustCompile(`^\s*([a-zA-Z][a-zA-Z0-9]*v[a-zA-Z0-9]+)\s+"([^"]+/apis/[^"]+)"\s*$`)).
		AddMatchGroupSection("registrations", "AddToSchemes = append(AddToSchemes,", ")",
			regexp.MustCompile(`^\s*([a-zA-Z][a-zA-Z0-9]*v[a-zA-Z0-9]+\.SchemeBuilder\.AddToScheme),?\s*$`)).
		Build()

	results, err := core.ParseFileWithConfig(f.Path, config)
	if err != nil {
		return []string{}, []string{}
	}

	return results["imports"], results["registrations"]
}
