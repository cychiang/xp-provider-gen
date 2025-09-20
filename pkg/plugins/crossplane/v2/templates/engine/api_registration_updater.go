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
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &APIRegistrationUpdater{}

// APIRegistrationUpdater updates the apis/register.go file to include new API group imports and registrations
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

// SetTemplateDefaults implements machinery.Template
func (f *APIRegistrationUpdater) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("apis", "register.go")
	}

	f.TemplateBody = ""
	// Always overwrite to update with new API registration
	f.IfExistsAction = machinery.OverwriteFile

	return nil
}

// GetBody implements machinery.Template to allow dynamic content generation
func (f *APIRegistrationUpdater) GetBody() string {
	// Parse existing file if it exists
	existingImports, existingRegistrations := f.parseExistingContent()

	// Determine new import path and registration call for the new API group
	var newImport, newRegistration string
	if f.Resource.Group != "" && f.Resource.Group != "v1alpha1" {
		// API group (e.g., sample, compute, storage)
		newImport = fmt.Sprintf(`%s%s "%s/apis/%s/%s"`, f.Resource.Group, f.Resource.Version, f.Repo, f.Resource.Group, f.Resource.Version)
		newRegistration = fmt.Sprintf("%s%s.SchemeBuilder.AddToScheme", f.Resource.Group, f.Resource.Version)
	}

	// Note: Provider v1alpha1 types are already registered via providerv1alpha1 import in initial template

	// Add new import if it's an API group and not already present
	if newImport != "" {
		importExists := false
		for _, existing := range existingImports {
			if strings.Contains(existing, fmt.Sprintf("/apis/%s/%s", f.Resource.Group, f.Resource.Version)) {
				importExists = true
				break
			}
		}
		if !importExists {
			existingImports = append(existingImports, newImport)
		}

		// Add new registration if not already present
		regExists := false
		for _, existing := range existingRegistrations {
			if existing == newRegistration {
				regExists = true
				break
			}
		}
		if !regExists {
			existingRegistrations = append(existingRegistrations, newRegistration)
		}
	}

	// Build imports section
	var imports strings.Builder
	for _, imp := range existingImports {
		imports.WriteString(fmt.Sprintf("\t%s\n", imp))
	}

	// Build registrations section
	var registrations strings.Builder
	for _, reg := range existingRegistrations {
		registrations.WriteString(fmt.Sprintf("\t\t%s,\n", reg))
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
`, f.ProviderName, imports.String(), registrations.String())
}

// parseExistingContent reads and parses the existing register.go file to extract API imports and registrations
func (f *APIRegistrationUpdater) parseExistingContent() ([]string, []string) {
	existingImports := []string{}
	existingRegistrations := []string{}

	// Try to read from the filesystem using Go's standard library
	file, err := os.ReadFile(f.Path)
	if err != nil {
		// File doesn't exist or can't be read, return empty lists
		return existingImports, existingRegistrations
	}

	content := string(file)
	scanner := bufio.NewScanner(strings.NewReader(content))

	inImports := false
	inRegistrations := false

	// Regex patterns for parsing
	importPattern := regexp.MustCompile(`^\s*([a-zA-Z][a-zA-Z0-9]*v[a-zA-Z0-9]+)\s+"([^"]+/apis/[^"]+)"\s*$`)
	registrationPattern := regexp.MustCompile(`^\s*([a-zA-Z][a-zA-Z0-9]*v[a-zA-Z0-9]+\.SchemeBuilder\.AddToScheme),?\s*$`)

	for scanner.Scan() {
		line := scanner.Text()

		// Track import section
		if strings.Contains(line, "import (") {
			inImports = true
			continue
		}
		if inImports && strings.Contains(line, ")") {
			inImports = false
			continue
		}

		// Track AddToSchemes section
		if strings.Contains(line, "AddToSchemes = append(AddToSchemes,") {
			inRegistrations = true
			continue
		}
		if inRegistrations && strings.Contains(line, ")") {
			inRegistrations = false
			continue
		}

		// Parse imports
		if inImports && importPattern.MatchString(line) {
			existingImports = append(existingImports, strings.TrimSpace(line))
		}

		// Parse registrations
		if inRegistrations && registrationPattern.MatchString(line) {
			matches := registrationPattern.FindStringSubmatch(line)
			if len(matches) > 1 {
				existingRegistrations = append(existingRegistrations, matches[1])
			}
		}
	}

	return existingImports, existingRegistrations
}