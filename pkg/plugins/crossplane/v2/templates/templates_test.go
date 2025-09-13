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
	"strings"
	"testing"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
)

// TestTemplateFramework provides utilities for testing templates following TDD principles
type TestTemplateFramework struct {
	t      *testing.T
	config config.Config
}

// NewTestFramework creates a new test framework for template testing
func NewTestFramework(t *testing.T) *TestTemplateFramework {
	// Create a mock config for testing
	cfg := &mockConfig{
		domain: "example.com",
		repo:   "github.com/example/provider-test",
	}

	return &TestTemplateFramework{
		t:      t,
		config: cfg,
	}
}

// AssertTemplate validates that a template generates the expected content
func (f *TestTemplateFramework) AssertTemplate(template machinery.Template, expectedContent string) {
	f.t.Helper()

	// Get the template body
	body := template.GetBody()

	// Normalize whitespace for comparison
	expectedNormalized := strings.TrimSpace(expectedContent)
	actualNormalized := strings.TrimSpace(body)

	if expectedNormalized != actualNormalized {
		f.t.Errorf("Template content mismatch:\nExpected:\n%s\n\nActual:\n%s", expectedNormalized, actualNormalized)
	}
}

// AssertTemplateContains validates that a template contains specific content
func (f *TestTemplateFramework) AssertTemplateContains(template machinery.Template, expectedContent string) {
	f.t.Helper()

	body := template.GetBody()
	if !strings.Contains(body, expectedContent) {
		f.t.Errorf("Template does not contain expected content:\nExpected to contain: %s\nActual body:\n%s", expectedContent, body)
	}
}

// AssertTemplatePath validates that a template has the expected file path
func (f *TestTemplateFramework) AssertTemplatePath(template machinery.Template, expectedPath string) {
	f.t.Helper()

	actualPath := template.GetPath()
	if actualPath != expectedPath {
		f.t.Errorf("Template path mismatch: expected %s, got %s", expectedPath, actualPath)
	}
}

// AssertTemplateAction validates that a template has the expected IfExists action
func (f *TestTemplateFramework) AssertTemplateAction(template machinery.Template, expectedAction machinery.IfExistsAction) {
	f.t.Helper()

	actualAction := template.GetIfExistsAction()
	if actualAction != expectedAction {
		f.t.Errorf("Template action mismatch: expected %v, got %v", expectedAction, actualAction)
	}
}

// mockConfig implements config.Config interface for testing
type mockConfig struct {
	domain string
	repo   string
}

func (m *mockConfig) GetDomain() string                          { return m.domain }
func (m *mockConfig) GetRepository() string                      { return m.repo }
func (m *mockConfig) GetProjectName() string                     { return "provider-test" }
func (m *mockConfig) GetVersion() config.Version                 { return config.Version{} }
func (m *mockConfig) GetCliVersion() string                      { return "v4.8.0" }
func (m *mockConfig) SetCliVersion(version string) error         { return nil }
func (m *mockConfig) SetDomain(domain string) error              { return nil }
func (m *mockConfig) SetRepository(repo string) error            { return nil }
func (m *mockConfig) SetProjectName(name string) error           { return nil }
func (m *mockConfig) GetPluginChain() []string                   { return nil }
func (m *mockConfig) SetPluginChain(chain []string) error        { return nil }
func (m *mockConfig) GetResources() ([]resource.Resource, error) { return nil, nil }
func (m *mockConfig) HasResource(gvk resource.GVK) bool          { return false }
func (m *mockConfig) GetResource(gvk resource.GVK) (resource.Resource, error) {
	return resource.Resource{}, nil
}
func (m *mockConfig) AddResource(res resource.Resource) error                             { return nil }
func (m *mockConfig) UpdateResource(res resource.Resource) error                          { return nil }
func (m *mockConfig) HasGroup(group string) bool                                          { return false }
func (m *mockConfig) ListCRDVersions() []string                                           { return nil }
func (m *mockConfig) ListWebhookVersions() []string                                       { return nil }
func (m *mockConfig) ResourcesLength() int                                                { return 0 }
func (m *mockConfig) DecodePluginConfig(pluginKey string, pluginConfig interface{}) error { return nil }
func (m *mockConfig) EncodePluginConfig(pluginKey string, pluginConfig interface{}) error { return nil }
func (m *mockConfig) IsMultiGroup() bool                                                  { return false }
func (m *mockConfig) SetMultiGroup() error                                                { return nil }
func (m *mockConfig) ClearMultiGroup() error                                              { return nil }
func (m *mockConfig) MarshalYAML() ([]byte, error)                                        { return nil, nil }
func (m *mockConfig) UnmarshalYAML([]byte) error                                          { return nil }

// Test functions for each template

func TestBoilerplate(t *testing.T) {
	_ = NewTestFramework(t)

	boilerplate := DefaultBoilerplate()

	// Test that boilerplate contains copyright notice
	if !strings.Contains(boilerplate, "Copyright 2025 The Crossplane Authors") {
		t.Error("Boilerplate should contain copyright notice")
	}

	// Test that boilerplate contains Apache license
	if !strings.Contains(boilerplate, "Apache License, Version 2.0") {
		t.Error("Boilerplate should contain Apache License")
	}
}

func TestGoModTemplate(t *testing.T) {
	framework := NewTestFramework(t)

	template := GoMod(framework.config)

	// Test path
	framework.AssertTemplatePath(template, "go.mod")

	// Test content contains expected elements
	framework.AssertTemplateContains(template, "module {{ .Repo }}")
	framework.AssertTemplateContains(template, "go 1.24")
	framework.AssertTemplateContains(template, "github.com/crossplane/crossplane-runtime")
}

func TestGitIgnoreTemplate(t *testing.T) {
	framework := NewTestFramework(t)

	template := GitIgnore(framework.config)

	// Test path
	framework.AssertTemplatePath(template, ".gitignore")

	// Test content contains expected patterns
	framework.AssertTemplateContains(template, "*.exe")
	framework.AssertTemplateContains(template, "bin/")
	framework.AssertTemplateContains(template, ".DS_Store")
}

func TestREADMETemplate(t *testing.T) {
	framework := NewTestFramework(t)

	template := README(framework.config)

	// Test path
	framework.AssertTemplatePath(template, "README.md")

	// Test content contains expected sections
	framework.AssertTemplateContains(template, "# {{ .ProviderName }}")
	framework.AssertTemplateContains(template, "## Getting Started")
	framework.AssertTemplateContains(template, "Crossplane provider")
}

func TestMakefileTemplate(t *testing.T) {
	framework := NewTestFramework(t)

	template := Makefile(framework.config)

	// Test path
	framework.AssertTemplatePath(template, "Makefile")

	// Test content contains expected targets
	framework.AssertTemplateContains(template, "# Makefile for {{ .ProviderName }}")
	framework.AssertTemplateContains(template, "build:")
	framework.AssertTemplateContains(template, "generate:")
}

func TestMainGoTemplate(t *testing.T) {
	framework := NewTestFramework(t)

	template := MainGo(framework.config)

	// Test path
	framework.AssertTemplatePath(template, "cmd/provider/main.go")

	// Test content contains expected elements
	framework.AssertTemplateContains(template, "package main")
	framework.AssertTemplateContains(template, "func main()")
	framework.AssertTemplateContains(template, "{{ .Repo }}/internal/controller")
}

func TestProviderConfigTypesTemplate(t *testing.T) {
	framework := NewTestFramework(t)

	template := ProviderConfigTypes(framework.config)

	// Test path
	framework.AssertTemplatePath(template, "apis/v1alpha1/types.go")

	// Test content contains expected types
	framework.AssertTemplateContains(template, "type ProviderConfig struct")
	framework.AssertTemplateContains(template, "type ProviderConfigSpec struct")
	framework.AssertTemplateContains(template, "+kubebuilder:object:root=true")
}

func TestProviderConfigRegisterTemplate(t *testing.T) {
	framework := NewTestFramework(t)

	template := ProviderConfigRegister(framework.config)

	// Test path
	framework.AssertTemplatePath(template, "apis/v1alpha1/register.go")

	// Test content contains expected registration
	framework.AssertTemplateContains(template, "SchemeBuilder")
	framework.AssertTemplateContains(template, "SchemeGroupVersion")
	framework.AssertTemplateContains(template, "func init()")
}

func TestCrossplanePackageTemplate(t *testing.T) {
	framework := NewTestFramework(t)

	template := CrossplanePackage(framework.config)

	// Test path
	framework.AssertTemplatePath(template, "package/crossplane.yaml")

	// Test content contains expected package metadata
	framework.AssertTemplateContains(template, "apiVersion: meta.pkg.crossplane.io/v1alpha1")
	framework.AssertTemplateContains(template, "kind: Provider")
	framework.AssertTemplateContains(template, "name: {{ .ProviderName }}")
}

func TestConfigControllerTemplate(t *testing.T) {
	framework := NewTestFramework(t)

	template := ConfigController(framework.config)

	// Test path
	framework.AssertTemplatePath(template, "internal/controller/config/config.go")

	// Test content contains expected controller setup
	framework.AssertTemplateContains(template, "func Setup(")
	framework.AssertTemplateContains(template, "ProviderConfig")
	framework.AssertTemplateContains(template, "{{ .Repo }}/apis/v1alpha1")
}

func TestControllerRegisterTemplate(t *testing.T) {
	framework := NewTestFramework(t)

	template := ControllerRegister(framework.config)

	// Test path
	framework.AssertTemplatePath(template, "internal/controller/register.go")

	// Test content contains expected controller registration
	framework.AssertTemplateContains(template, "func Setup(")
	framework.AssertTemplateContains(template, "config.Setup")
	framework.AssertTemplateContains(template, "{{ .ProviderName }} controllers")
}

func TestVersionGoTemplate(t *testing.T) {
	framework := NewTestFramework(t)

	template := VersionGo(framework.config)

	// Test path
	framework.AssertTemplatePath(template, "internal/version/version.go")

	// Test content contains expected version management
	framework.AssertTemplateContains(template, "var Version string")
	framework.AssertTemplateContains(template, "func GetVersion()")
	framework.AssertTemplateContains(template, "debug.ReadBuildInfo")
}

func TestClusterDockerfileTemplate(t *testing.T) {
	framework := NewTestFramework(t)

	template := ClusterDockerfile(framework.config)

	// Test path
	framework.AssertTemplatePath(template, "cluster/images/provider/Dockerfile")

	// Test content contains expected Docker directives
	framework.AssertTemplateContains(template, "FROM alpine:")
	framework.AssertTemplateContains(template, "COPY provider")
	framework.AssertTemplateContains(template, "ENTRYPOINT")
}

func TestClusterMakefileTemplate(t *testing.T) {
	framework := NewTestFramework(t)

	template := ClusterMakefile(framework.config)

	// Test path
	framework.AssertTemplatePath(template, "cluster/images/provider/Makefile")

	// Test content contains expected build directives
	framework.AssertTemplateContains(template, "# Cluster image Makefile")
	framework.AssertTemplateContains(template, "{{ .ProviderName }}")
	framework.AssertTemplateContains(template, "include")
}

func TestLicenseTemplate(t *testing.T) {
	framework := NewTestFramework(t)

	template := License(framework.config)

	// Test path
	framework.AssertTemplatePath(template, "LICENSE")

	// Test content contains expected license text
	framework.AssertTemplateContains(template, "Apache License")
	framework.AssertTemplateContains(template, "Version 2.0, January 2004")
	framework.AssertTemplateContains(template, "TERMS AND CONDITIONS")
}

// API Template Tests (these require resource context)

func TestAPITypesTemplate(t *testing.T) {
	framework := NewTestFramework(t)

	template := APITypes(framework.config, false)

	// Test path pattern (will be resolved with resource context)
	expectedPath := "apis/%[group]/%[version]/%[kind]_types.go"
	framework.AssertTemplatePath(template, expectedPath)

	// Test content contains expected CRD structure
	framework.AssertTemplateContains(template, "{{ .Resource.Kind }}Parameters")
	framework.AssertTemplateContains(template, "{{ .Resource.Kind }}Observation")
	framework.AssertTemplateContains(template, "+kubebuilder:object:root=true")
}

func TestAPIGroupTemplate(t *testing.T) {
	framework := NewTestFramework(t)

	template := APIGroup(framework.config)

	// Test path pattern
	expectedPath := "apis/%[group]/%[version]/groupversion_info.go"
	framework.AssertTemplatePath(template, expectedPath)

	// Test content contains expected group registration
	framework.AssertTemplateContains(template, "SchemeGroupVersion")
	framework.AssertTemplateContains(template, "{{ .Resource.QualifiedGroup }}")
	framework.AssertTemplateContains(template, "+kubebuilder:object:generate=true")
}

func TestControllerTemplate(t *testing.T) {
	framework := NewTestFramework(t)

	template := Controller(framework.config, false)

	// Test path pattern
	expectedPath := "internal/controller/%[group]/%[kind]/%[kind].go"
	framework.AssertTemplatePath(template, expectedPath)

	// Test content contains expected controller structure
	framework.AssertTemplateContains(template, "func Setup(")
	framework.AssertTemplateContains(template, "managed.NewReconciler")
	framework.AssertTemplateContains(template, "{{ .Resource.Kind }} managed resources")
}
