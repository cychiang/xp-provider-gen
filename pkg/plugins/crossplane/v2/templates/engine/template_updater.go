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

var _ machinery.Template = &TemplateUpdater{}

// TemplateUpdater updates the register.go file to include new controller imports and setup calls
type TemplateUpdater struct {
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
func (f *TemplateUpdater) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("internal", "controller", "register.go")
	}

	f.TemplateBody = ""
	// Always overwrite to update with new controller
	f.IfExistsAction = machinery.OverwriteFile

	return nil
}

// GetBody implements machinery.Template to allow dynamic content generation
func (f *TemplateUpdater) GetBody() string {
	// Parse existing file if it exists
	existingImports, existingSetups := f.parseExistingContent()

	// Determine new import path and setup call
	// Controllers are always created in internal/controller/{kind}, regardless of API group
	newImport := fmt.Sprintf("%s/internal/controller/%s", f.Repo, strings.ToLower(f.Resource.Kind))
	newSetup := fmt.Sprintf("%s.Setup", strings.ToLower(f.Resource.Kind))

	// Add new import if not already present
	importExists := false
	for _, existing := range existingImports {
		if existing == newImport {
			importExists = true
			break
		}
	}
	if !importExists {
		existingImports = append(existingImports, newImport)
	}

	// Add new setup if not already present
	setupExists := false
	for _, existing := range existingSetups {
		if existing == newSetup {
			setupExists = true
			break
		}
	}
	if !setupExists {
		existingSetups = append(existingSetups, newSetup)
	}

	// Build imports section
	var imports strings.Builder
	for _, imp := range existingImports {
		imports.WriteString(fmt.Sprintf("\t\"%s\"\n", imp))
	}

	// Build setups section
	var setups strings.Builder
	for _, setup := range existingSetups {
		setups.WriteString(fmt.Sprintf("\t\t%s,\n", setup))
	}

	return fmt.Sprintf(`{{ .Boilerplate }}

package controller

import (
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"

	"{{ .Repo }}/internal/controller/config"
%s)

// Setup creates all %s controllers with the supplied logger and adds them to
// the supplied manager.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	for _, setup := range []func(ctrl.Manager, controller.Options) error{
		config.Setup,
%s	} {
		if err := setup(mgr, o); err != nil {
			return err
		}
	}
	return nil
}
`, imports.String(), f.ProviderName, setups.String())
}

// parseExistingContent reads and parses the existing template.go file to extract controllers
func (f *TemplateUpdater) parseExistingContent() ([]string, []string) {
	existingImports := []string{}
	existingSetups := []string{}

	// Try to read from the filesystem using Go's standard library
	file, err := os.ReadFile(f.Path)
	if err != nil {
		// File doesn't exist or can't be read, return empty lists
		return existingImports, existingSetups
	}

	content := string(file)
	scanner := bufio.NewScanner(strings.NewReader(content))

	inImports := false
	inSetups := false

	// Regex patterns for parsing
	importPattern := regexp.MustCompile(`^\s*"([^"]+/internal/controller/[^"]+)"\s*$`)
	setupPattern := regexp.MustCompile(`^\s*([a-zA-Z][a-zA-Z0-9]*\.Setup),?\s*$`)

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

		// Track setup function section
		if strings.Contains(line, "for _, setup := range []func(ctrl.Manager, controller.Options) error{") {
			inSetups = true
			continue
		}
		if inSetups && strings.Contains(line, "}") {
			inSetups = false
			continue
		}

		// Parse imports (skip config import as it's always present)
		if inImports && importPattern.MatchString(line) {
			matches := importPattern.FindStringSubmatch(line)
			if len(matches) > 1 && !strings.HasSuffix(matches[1], "/internal/controller/config") {
				existingImports = append(existingImports, matches[1])
			}
		}

		// Parse setup calls (skip config.Setup as it's always present)
		if inSetups && setupPattern.MatchString(line) {
			matches := setupPattern.FindStringSubmatch(line)
			if len(matches) > 1 && matches[1] != "config.Setup" {
				existingSetups = append(existingSetups, matches[1])
			}
		}
	}

	return existingImports, existingSetups
}
