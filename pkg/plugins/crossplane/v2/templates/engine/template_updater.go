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
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"

	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/core"
)

var _ machinery.Template = &TemplateUpdater{}

// TemplateUpdater updates the register.go file to include new controller imports and setup calls.
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

// SetTemplateDefaults implements machinery.Template.
func (f *TemplateUpdater) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("internal", "controller", "register.go")
	}

	f.TemplateBody = ""
	// Always overwrite to update with new controller
	f.IfExistsAction = machinery.OverwriteFile

	return nil
}

// GetBody implements machinery.Template to allow dynamic content generation.
func (f *TemplateUpdater) GetBody() string {
	existingImports, existingSetups := f.parseExistingContent()

	allImports := f.buildImportsList(existingImports)
	allSetups := f.buildSetupsList(existingSetups)

	importsSection := f.buildImportsSection(allImports)
	setupsSection := f.buildSetupsSection(allSetups)

	return fmt.Sprintf(`{{ .Boilerplate }}

package controller

import (
%s)

// Setup creates all %s controllers with the supplied logger and adds them to
// the supplied manager.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	for _, setup := range []func(ctrl.Manager, controller.Options) error{
%s	} {
		if err := setup(mgr, o); err != nil {
			return err
		}
	}
	return nil
}
`, importsSection, f.ProviderName, setupsSection)
}

// parseExistingContent reads and parses the existing register.go file to extract controllers.
func (f *TemplateUpdater) parseExistingContent() ([]string, []string) {
	config := core.NewFileParserBuilder().
		AddMatchGroupSection("imports", "import (", ")",
			regexp.MustCompile(`^\s*"([^"]+/internal/controller/[^"]+)"\s*$`)).
		AddMatchGroupSection("setups", "for _, setup := range []func(ctrl.Manager, controller.Options) error{", "}",
			regexp.MustCompile(`^\s*([a-zA-Z][a-zA-Z0-9]*\.Setup),?\s*$`)).
		Build()

	results, err := core.ParseFileWithConfig(f.Path, config)
	if err != nil {
		return []string{}, []string{}
	}

	return results["imports"], results["setups"]
}

// buildImportsList builds the complete list of imports including config and new resource.
func (f *TemplateUpdater) buildImportsList(existingImports []string) []string {
	configImport := fmt.Sprintf("%s/internal/controller/config", f.Repo)
	allImports := f.ensureStringInList(existingImports, configImport, true)

	if f.Resource != nil {
		newImport := fmt.Sprintf("%s/internal/controller/%s", f.Repo, strings.ToLower(f.Resource.Kind))
		allImports = f.ensureStringInList(allImports, newImport, false)
	}

	return allImports
}

// buildSetupsList builds the complete list of setups including config and new resource.
func (f *TemplateUpdater) buildSetupsList(existingSetups []string) []string {
	allSetups := f.ensureStringInList(existingSetups, "config.Setup", true)

	if f.Resource != nil {
		newSetup := fmt.Sprintf("%s.Setup", strings.ToLower(f.Resource.Kind))
		allSetups = f.ensureStringInList(allSetups, newSetup, false)
	}

	return allSetups
}

// ensureStringInList adds a string to the list if not already present.
// If prepend is true, adds to the beginning; otherwise appends to the end.
func (f *TemplateUpdater) ensureStringInList(list []string, item string, prepend bool) []string {
	for _, existing := range list {
		if existing == item {
			return list
		}
	}

	if prepend {
		return append([]string{item}, list...)
	}
	return append(list, item)
}

// buildImportsSection creates the imports section string.
func (f *TemplateUpdater) buildImportsSection(imports []string) string {
	var section strings.Builder
	section.WriteString("\tctrl \"sigs.k8s.io/controller-runtime\"\n")
	section.WriteString("\n")
	section.WriteString("\t\"github.com/crossplane/crossplane-runtime/v2/pkg/controller\"\n")
	section.WriteString("\n")
	for _, imp := range imports {
		section.WriteString(fmt.Sprintf("\t\"%s\"\n", imp))
	}
	return section.String()
}

// buildSetupsSection creates the setups section string.
func (f *TemplateUpdater) buildSetupsSection(setups []string) string {
	var section strings.Builder
	for _, setup := range setups {
		section.WriteString(fmt.Sprintf("\t\t%s,\n", setup))
	}
	return section.String()
}
