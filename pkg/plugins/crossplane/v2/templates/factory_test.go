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
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
)

type mockConfig struct{}

func (m *mockConfig) GetDomain() string                                          { return "example.com" }
func (m *mockConfig) SetDomain(domain string) error                              { return nil }
func (m *mockConfig) GetRepository() string                                      { return "github.com/example/provider-test" }
func (m *mockConfig) SetRepository(repository string) error                      { return nil }
func (m *mockConfig) GetVersion() config.Version                                 { return config.Version{Number: 3} }
func (m *mockConfig) GetCliVersion() string                                      { return "4.0.0" }
func (m *mockConfig) SetCliVersion(version string) error                         { return nil }
func (m *mockConfig) GetProjectName() string                                     { return "provider-test" }
func (m *mockConfig) SetProjectName(name string) error                           { return nil }
func (m *mockConfig) GetPluginChain() []string                                   { return []string{"crossplane.go.kubebuilder.io/v2"} }
func (m *mockConfig) SetPluginChain(pluginChain []string) error                  { return nil }
func (m *mockConfig) IsMultiGroup() bool                                         { return false }
func (m *mockConfig) SetMultiGroup() error                                       { return nil }
func (m *mockConfig) ClearMultiGroup() error                                     { return nil }
func (m *mockConfig) ResourcesLength() int                                       { return 0 }
func (m *mockConfig) GetResources() ([]resource.Resource, error)                 { return []resource.Resource{}, nil }
func (m *mockConfig) HasResource(gvk resource.GVK) bool                          { return false }
func (m *mockConfig) GetResource(gvk resource.GVK) (resource.Resource, error)    { return resource.Resource{}, nil }
func (m *mockConfig) AddResource(res resource.Resource) error                    { return nil }
func (m *mockConfig) UpdateResource(res resource.Resource) error                 { return nil }
func (m *mockConfig) HasGroup(group string) bool                                 { return false }
func (m *mockConfig) ListCRDVersions() []string                                  { return []string{} }
func (m *mockConfig) ListWebhookVersions() []string                              { return []string{} }
func (m *mockConfig) EncodePluginConfig(key string, pluginConfig interface{}) error { return nil }
func (m *mockConfig) DecodePluginConfig(key string, pluginConfig interface{}) error { return nil }
func (m *mockConfig) MarshalYAML() ([]byte, error)                               { return []byte{}, nil }
func (m *mockConfig) UnmarshalYAML(data []byte) error                            { return nil }

func newMockConfig() config.Config {
	return &mockConfig{}
}

func TestCrossplaneTemplateFactory_NewFactory(t *testing.T) {
	cfg := newMockConfig()
	factory := NewFactory(cfg)

	if factory == nil {
		t.Error("Factory should not be nil")
	}

	concreteFactory, ok := factory.(*CrossplaneTemplateFactory)
	if !ok {
		t.Error("Factory should be of type CrossplaneTemplateFactory")
	}

	if concreteFactory.config != cfg {
		t.Error("Factory should have correct config")
	}
}

func TestCrossplaneTemplateFactory_GetSupportedTypes(t *testing.T) {
	cfg := newMockConfig()
	factory := NewFactory(cfg)

	types := factory.GetSupportedTypes()
	if len(types) == 0 {
		t.Error("Factory should support multiple template types")
	}

	expectedPatterns := []string{
		"gomod", "makefile", "readme", "license", "maingo",
		"apis", "generatego", "boilerplate",
	}

	typeStrings := make([]string, len(types))
	for i, templateType := range types {
		typeStrings[i] = strings.ToLower(string(templateType))
	}

	for _, pattern := range expectedPatterns {
		found := false
		for _, typeStr := range typeStrings {
			if strings.Contains(typeStr, pattern) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected pattern %s not found in discovered template types", pattern)
		}
	}

	t.Logf("Auto-discovery found %d template types", len(types))

	t.Logf("Found %d init, %d API, %d static templates",
		len(factory.(*CrossplaneTemplateFactory).initRegistry),
		len(factory.(*CrossplaneTemplateFactory).apiRegistry),
		len(factory.(*CrossplaneTemplateFactory).staticRegistry))
}

func TestCrossplaneTemplateFactory_CreateInitTemplate(t *testing.T) {
	cfg := newMockConfig()
	factory := NewFactory(cfg)

	templates, err := factory.GetInitTemplates()
	if err != nil {
		t.Fatalf("GetInitTemplates failed: %v", err)
	}

	if len(templates) == 0 {
		t.Error("GetInitTemplates should return templates")
	}

	expectedPatterns := []string{"gomod", "makefile", "readme", "maingo"}
	for _, pattern := range expectedPatterns {
		found := false
		for _, template := range templates {
			templateType := strings.ToLower(string(template.GetTemplateType()))
			if strings.Contains(templateType, pattern) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected init template pattern %s not found", pattern)
		}
	}
}

func TestCrossplaneTemplateFactory_CreateAPITemplate(t *testing.T) {
	cfg := newMockConfig()
	factory := NewFactory(cfg)

	res := &resource.Resource{
		GVK: resource.GVK{
			Group:   "compute",
			Version: "v1alpha1",
			Kind:    "Instance",
		},
		Plural: "instances",
	}

	templates, err := factory.GetAPITemplates(WithResource(res))
	if err != nil {
		t.Fatalf("GetAPITemplates failed: %v", err)
	}

	if len(templates) == 0 {
		t.Error("GetAPITemplates should return templates")
	}

	expectedPatterns := []string{"types", "groupversion", "controller"}
	for _, pattern := range expectedPatterns {
		found := false
		for _, template := range templates {
			templateType := strings.ToLower(string(template.GetTemplateType()))
			if strings.Contains(templateType, pattern) {
				found = true
				break
			}
		}
		if !found {
			t.Logf("Available API template types: %v", func() []string {
				var types []string
				for _, tmpl := range templates {
					types = append(types, string(tmpl.GetTemplateType()))
				}
				return types
			}())
			t.Errorf("Expected API template pattern %s not found", pattern)
		}
	}
}

func TestCrossplaneTemplateFactory_CreateStaticTemplate(t *testing.T) {
	cfg := newMockConfig()
	factory := NewFactory(cfg)

	templates, err := factory.GetStaticTemplates()
	if err != nil {
		t.Fatalf("GetStaticTemplates failed: %v", err)
	}

	if len(templates) == 0 {
		t.Error("GetStaticTemplates should return templates")
	}

	expectedPatterns := []string{"license"}
	for _, pattern := range expectedPatterns {
		found := false
		for _, template := range templates {
			templateType := strings.ToLower(string(template.GetTemplateType()))
			if strings.Contains(templateType, pattern) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected static template pattern %s not found", pattern)
		}
	}
}

func TestCrossplaneTemplateFactory_ConvenienceMethods(t *testing.T) {
	cfg := newMockConfig()
	factory := NewFactory(cfg)

	t.Run("Init_templates", func(t *testing.T) {
		templates, err := factory.GetInitTemplates()
		if err != nil {
			t.Errorf("GetInitTemplates failed: %v", err)
		}
		if len(templates) == 0 {
			t.Error("GetInitTemplates should return templates")
		}
		for _, template := range templates {
			if template == nil {
				t.Error("Template should not be nil")
			}
		}
	})

	t.Run("Static_templates", func(t *testing.T) {
		templates, err := factory.GetStaticTemplates()
		if err != nil {
			t.Errorf("GetStaticTemplates failed: %v", err)
		}
		if len(templates) == 0 {
			t.Error("GetStaticTemplates should return templates")
		}
		for _, template := range templates {
			if template == nil {
				t.Error("Template should not be nil")
			}
		}
	})

	t.Run("API_templates", func(t *testing.T) {
		res := &resource.Resource{
			GVK: resource.GVK{
				Group:   "compute",
				Version: "v1alpha1",
				Kind:    "Instance",
			},
			Plural: "instances",
		}

		templates, err := factory.GetAPITemplates(WithResource(res))
		if err != nil {
			t.Errorf("GetAPITemplates failed: %v", err)
		}
		if len(templates) == 0 {
			t.Error("GetAPITemplates should return templates")
		}
		for _, template := range templates {
			if template == nil {
				t.Error("Template should not be nil")
			}
		}
	})
}

func TestTemplateOptions(t *testing.T) {
	options := &TemplateOptions{}

	WithForce(true)(options)
	if !options.Force {
		t.Error("WithForce should set Force to true")
	}

	res := &resource.Resource{
		GVK: resource.GVK{
			Group:   "compute",
			Version: "v1alpha1",
			Kind:    "Instance",
		},
	}
	WithResource(res)(options)
	if options.Resource != res {
		t.Error("WithResource should set Resource")
	}

	customData := map[string]interface{}{"key": "value"}
	WithCustomData(customData)(options)
	if options.CustomData["key"] != "value" {
		t.Error("WithCustomData should set CustomData")
	}
}