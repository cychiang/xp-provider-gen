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

// parseExistingContent reads and parses the existing template.go file to extract controllers.
func (f *TemplateUpdater) parseExistingContent() ([]string, []string) {
	content, err := f.readFileContent()
	if err != nil {
		return []string{}, []string{}
	}

	parser := &TemplateFileParser{
		content:       content,
		importPattern: regexp.MustCompile(`^\s*"([^"]+/internal/controller/[^"]+)"\s*$`),
		setupPattern:  regexp.MustCompile(`^\s*([a-zA-Z][a-zA-Z0-9]*\.Setup),?\s*$`),
	}

	return parser.Parse()
}

func (f *TemplateUpdater) readFileContent() (string, error) {
	file, err := os.ReadFile(f.Path)
	if err != nil {
		return "", err
	}
	return string(file), nil
}

// TemplateFileParser handles parsing of template files.
type TemplateFileParser struct {
	content       string
	importPattern *regexp.Regexp
	setupPattern  *regexp.Regexp
}

// Parse extracts imports and setup calls from template file content.
func (p *TemplateFileParser) Parse() ([]string, []string) {
	var imports, setups []string

	scanner := bufio.NewScanner(strings.NewReader(p.content))
	state := &templateParseState{}

	for scanner.Scan() {
		line := scanner.Text()

		p.updateTemplateState(state, line)
		imports = p.processTemplateImport(imports, state, line)
		setups = p.processTemplateSetup(setups, state, line)
	}

	return imports, setups
}

type templateParseState struct {
	inImports bool
	inSetups  bool
}

func (p *TemplateFileParser) updateTemplateState(state *templateParseState, line string) {
	// Update state for imports
	if strings.Contains(line, "import (") {
		state.inImports = true
		return
	}
	if state.inImports && strings.Contains(line, ")") {
		state.inImports = false
		return
	}

	// Update state for setups
	if strings.Contains(line, "for _, setup := range []func(ctrl.Manager, controller.Options) error{") {
		state.inSetups = true
		return
	}
	if state.inSetups && strings.Contains(line, "}") {
		state.inSetups = false
	}
}

func (p *TemplateFileParser) processTemplateImport(imports []string, state *templateParseState, line string) []string {
	if state.inImports {
		if importLine := p.parseImport(line); importLine != "" {
			imports = append(imports, importLine)
		}
	}
	return imports
}

func (p *TemplateFileParser) processTemplateSetup(setups []string, state *templateParseState, line string) []string {
	if state.inSetups {
		if setupLine := p.parseSetup(line); setupLine != "" {
			setups = append(setups, setupLine)
		}
	}
	return setups
}

func (p *TemplateFileParser) parseImport(line string) string {
	if matches := p.importPattern.FindStringSubmatch(line); len(matches) > 1 {
		if !strings.HasSuffix(matches[1], "/internal/controller/config") {
			return matches[1]
		}
	}
	return ""
}

func (p *TemplateFileParser) parseSetup(line string) string {
	if matches := p.setupPattern.FindStringSubmatch(line); len(matches) > 1 {
		if matches[1] != "config.Setup" {
			return matches[1]
		}
	}
	return ""
}
