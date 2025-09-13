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
	"testing"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
)

// mockConfig implements config.Config interface for testing
type mockConfig struct {
	domain string
	repo   string
}

func (m *mockConfig) GetDomain() string                                                  { return m.domain }
func (m *mockConfig) GetRepository() string                                              { return m.repo }
func (m *mockConfig) GetProjectName() string                                             { return "provider-test" }
func (m *mockConfig) GetVersion() config.Version                                         { return config.Version{} }
func (m *mockConfig) GetCliVersion() string                                              { return "v4.8.0" }
func (m *mockConfig) SetCliVersion(version string) error                                 { return nil }
func (m *mockConfig) SetDomain(domain string) error                                      { return nil }
func (m *mockConfig) SetRepository(repo string) error                                    { return nil }
func (m *mockConfig) SetProjectName(name string) error                                   { return nil }
func (m *mockConfig) GetPluginChain() []string                                           { return nil }
func (m *mockConfig) SetPluginChain(chain []string) error                                { return nil }
func (m *mockConfig) GetResources() ([]resource.Resource, error)                         { return nil, nil }
func (m *mockConfig) HasResource(gvk resource.GVK) bool                                  { return false }
func (m *mockConfig) GetResource(gvk resource.GVK) (resource.Resource, error)            { return resource.Resource{}, nil }
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

func newMockConfig() config.Config {
	return &mockConfig{
		domain: "example.com",
		repo:   "github.com/example/provider-test",
	}
}

func TestCrossplaneTemplateFactory_NewFactory(t *testing.T) {
	cfg := newMockConfig()
	factory := NewFactory(cfg)

	if factory == nil {
		t.Error("NewFactory should return non-nil factory")
	}

	concreteFactory, ok := factory.(*CrossplaneTemplateFactory)
	if !ok {
		t.Error("NewFactory should return *CrossplaneTemplateFactory")
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

	expectedTypes := map[TemplateType]bool{
		GoModTemplateType:            true,
		MakefileTemplateType:         true,
		READMETemplateType:           true,
		GitIgnoreTemplateType:        true,
		MainGoTemplateType:           true,
		ProviderConfigTypesType:      true,
		ProviderConfigRegisterType:   true,
		CrossplanePackageType:        true,
		ConfigControllerType:         true,
		ControllerRegisterType:       true,
		LicenseType:                  true,
		APITypesTemplateType:         true,
		APIGroupTemplateType:         true,
		ControllerTemplateType:       true,
	}

	foundTypes := make(map[TemplateType]bool)
	for _, templateType := range types {
		foundTypes[templateType] = true
	}

	for expectedType := range expectedTypes {
		if !foundTypes[expectedType] {
			t.Errorf("Expected template type %s not found in supported types", expectedType)
		}
	}
}

func TestCrossplaneTemplateFactory_CreateInitTemplate(t *testing.T) {
	cfg := newMockConfig()
	factory := NewFactory(cfg)

	tests := []struct {
		name         string
		templateType TemplateType
		expectError  bool
	}{
		{"GoMod", GoModTemplateType, false},
		{"Makefile", MakefileTemplateType, false},
		{"README", READMETemplateType, false},
		{"GitIgnore", GitIgnoreTemplateType, false},
		{"MainGo", MainGoTemplateType, false},
		{"ProviderConfigTypes", ProviderConfigTypesType, false},
		{"ProviderConfigRegister", ProviderConfigRegisterType, false},
		{"CrossplanePackage", CrossplanePackageType, false},
		{"ConfigController", ConfigControllerType, false},
		{"ControllerRegister", ControllerRegisterType, false},
		{"Invalid", TemplateType("invalid"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			product, err := factory.CreateInitTemplate(tt.templateType)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error for invalid template type")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if product == nil {
				t.Error("Product should not be nil")
				return
			}

			if product.GetTemplateType() != tt.templateType {
				t.Errorf("Expected template type %s, got %s", tt.templateType, product.GetTemplateType())
			}

			// Test that template implements machinery.Template
			if product.GetPath() == "" {
				t.Error("Template should have a path")
			}
		})
	}
}

func TestCrossplaneTemplateFactory_CreateAPITemplate(t *testing.T) {
	cfg := newMockConfig()
	factory := NewFactory(cfg)

	testResource := &resource.Resource{
		GVK: resource.GVK{
			Group:   "compute",
			Version: "v1alpha1",
			Kind:    "Instance",
		},
	}

	tests := []struct {
		name         string
		templateType TemplateType
		resource     *resource.Resource
		expectError  bool
	}{
		{"APITypes", APITypesTemplateType, testResource, false},
		{"APIGroup", APIGroupTemplateType, testResource, false},
		{"Controller", ControllerTemplateType, testResource, false},
		{"APITypes without resource", APITypesTemplateType, nil, true},
		{"Invalid", TemplateType("invalid"), testResource, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := []Option{}
			if tt.resource != nil {
				opts = append(opts, WithResource(tt.resource))
			}

			product, err := factory.CreateAPITemplate(tt.templateType, opts...)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if product == nil {
				t.Error("Product should not be nil")
				return
			}

			if product.GetTemplateType() != tt.templateType {
				t.Errorf("Expected template type %s, got %s", tt.templateType, product.GetTemplateType())
			}
		})
	}
}

func TestCrossplaneTemplateFactory_CreateStaticTemplate(t *testing.T) {
	cfg := newMockConfig()
	factory := NewFactory(cfg)

	tests := []struct {
		name         string
		templateType TemplateType
		expectError  bool
	}{
		{"License", LicenseType, false},
		{"Invalid", TemplateType("invalid"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			product, err := factory.CreateStaticTemplate(tt.templateType)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error for invalid template type")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if product == nil {
				t.Error("Product should not be nil")
				return
			}

			if product.GetTemplateType() != tt.templateType {
				t.Errorf("Expected template type %s, got %s", tt.templateType, product.GetTemplateType())
			}
		})
	}
}

func TestCrossplaneTemplateFactory_ConvenienceMethods(t *testing.T) {
	cfg := newMockConfig()
	factory := NewFactory(cfg).(*CrossplaneTemplateFactory)

	t.Run("Init templates", func(t *testing.T) {
		initMethods := []func() (TemplateProduct, error){
			factory.GoMod,
			factory.Makefile,
			factory.README,
			factory.GitIgnore,
			factory.MainGo,
			factory.ProviderConfigTypes,
			factory.ProviderConfigRegister,
			factory.CrossplanePackage,
			factory.ConfigController,
			factory.ControllerRegister,
		}

		for i, method := range initMethods {
			product, err := method()
			if err != nil {
				t.Errorf("Method %d failed: %v", i, err)
			}
			if product == nil {
				t.Errorf("Method %d returned nil product", i)
			}
		}
	})

	t.Run("Static templates", func(t *testing.T) {
		product, err := factory.License()
		if err != nil {
			t.Errorf("License method failed: %v", err)
		}
		if product == nil {
			t.Error("License method returned nil product")
		}
	})

	t.Run("API templates", func(t *testing.T) {
		testResource := &resource.Resource{
			GVK: resource.GVK{
				Group:   "compute",
				Version: "v1alpha1",
				Kind:    "Instance",
			},
		}

		apiMethods := []func() (TemplateProduct, error){
			func() (TemplateProduct, error) { return factory.APITypes(false, testResource) },
			func() (TemplateProduct, error) { return factory.APIGroup(testResource) },
			func() (TemplateProduct, error) { return factory.Controller(false, testResource) },
		}

		for i, method := range apiMethods {
			product, err := method()
			if err != nil {
				t.Errorf("API method %d failed: %v", i, err)
			}
			if product == nil {
				t.Errorf("API method %d returned nil product", i)
			}
		}
	})
}

func TestTemplateOptions(t *testing.T) {
	testResource := &resource.Resource{
		GVK: resource.GVK{
			Group:   "compute",
			Version: "v1alpha1",
			Kind:    "Instance",
		},
	}

	testData := map[string]interface{}{
		"custom": "value",
	}

	opts := &TemplateOptions{}

	WithForce(true)(opts)
	if !opts.Force {
		t.Error("WithForce should set Force to true")
	}

	WithResource(testResource)(opts)
	if opts.Resource != testResource {
		t.Error("WithResource should set Resource")
	}

	WithCustomData(testData)(opts)
	if opts.CustomData["custom"] != "value" {
		t.Error("WithCustomData should set CustomData")
	}
}