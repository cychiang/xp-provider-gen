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
	"fmt"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
)

// CrossplaneTemplateFactory implements the TemplateFactory interface
type CrossplaneTemplateFactory struct {
	config         config.Config
	initRegistry   map[TemplateType]TemplateBuilder
	apiRegistry    map[TemplateType]TemplateBuilder
	staticRegistry map[TemplateType]TemplateBuilder
}

// NewFactory creates a new CrossplaneTemplateFactory
func NewFactory(cfg config.Config) TemplateFactory {
	factory := &CrossplaneTemplateFactory{
		config:         cfg,
		initRegistry:   make(map[TemplateType]TemplateBuilder),
		apiRegistry:    make(map[TemplateType]TemplateBuilder),
		staticRegistry: make(map[TemplateType]TemplateBuilder),
	}

	// Register init template builders
	initTypes := []TemplateType{
		GoModTemplateType, MakefileTemplateType, READMETemplateType,
		GitIgnoreTemplateType, MainGoTemplateType, APIsTemplateType, GenerateGoTemplateType,
		BoilerplateTemplateType, ProviderConfigTypesType, ProviderConfigRegisterType,
		CrossplanePackageType, ConfigControllerType, ControllerRegisterType,
		ClusterDockerfileType, ClusterMakefileType, VersionGoType, DocGoType,
		ExamplesProviderConfigTemplateType,
	}
	for _, templateType := range initTypes {
		factory.initRegistry[templateType] = NewInitTemplateBuilder(templateType)
	}

	// Register API template builders
	apiTypes := []TemplateType{
		APITypesTemplateType, APIGroupTemplateType, ControllerTemplateType,
		ExamplesManagedResourceTemplateType,
	}
	for _, templateType := range apiTypes {
		factory.apiRegistry[templateType] = NewAPITemplateBuilder(templateType)
	}

	// Register static template builders
	staticTypes := []TemplateType{
		LicenseType,
	}
	for _, templateType := range staticTypes {
		factory.staticRegistry[templateType] = NewStaticTemplateBuilder(templateType)
	}

	return factory
}

// CreateInitTemplate creates initialization templates
func (f *CrossplaneTemplateFactory) CreateInitTemplate(templateType TemplateType, opts ...Option) (TemplateProduct, error) {
	builder, exists := f.initRegistry[templateType]
	if !exists {
		return nil, fmt.Errorf("unsupported init template type: %s", templateType)
	}

	product, err := builder.Build(f.config, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to build init template %s: %w", templateType, err)
	}

	return product, nil
}

// CreateAPITemplate creates API templates
func (f *CrossplaneTemplateFactory) CreateAPITemplate(templateType TemplateType, opts ...Option) (TemplateProduct, error) {
	builder, exists := f.apiRegistry[templateType]
	if !exists {
		return nil, fmt.Errorf("unsupported API template type: %s", templateType)
	}

	product, err := builder.Build(f.config, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to build API template %s: %w", templateType, err)
	}

	return product, nil
}

// CreateStaticTemplate creates static templates
func (f *CrossplaneTemplateFactory) CreateStaticTemplate(templateType TemplateType, opts ...Option) (TemplateProduct, error) {
	builder, exists := f.staticRegistry[templateType]
	if !exists {
		return nil, fmt.Errorf("unsupported static template type: %s", templateType)
	}

	product, err := builder.Build(f.config, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to build static template %s: %w", templateType, err)
	}

	return product, nil
}

// GetSupportedTypes returns all supported template types
func (f *CrossplaneTemplateFactory) GetSupportedTypes() []TemplateType {
	var types []TemplateType

	for templateType := range f.initRegistry {
		types = append(types, templateType)
	}

	for templateType := range f.apiRegistry {
		types = append(types, templateType)
	}

	for templateType := range f.staticRegistry {
		types = append(types, templateType)
	}

	return types
}

// Convenience methods for backward compatibility and ease of use

// GoMod creates a go.mod template
func (f *CrossplaneTemplateFactory) GoMod() (TemplateProduct, error) {
	return f.CreateInitTemplate(GoModTemplateType)
}

// Makefile creates a Makefile template
func (f *CrossplaneTemplateFactory) Makefile() (TemplateProduct, error) {
	return f.CreateInitTemplate(MakefileTemplateType)
}

// README creates a README.md template
func (f *CrossplaneTemplateFactory) README() (TemplateProduct, error) {
	return f.CreateInitTemplate(READMETemplateType)
}

// GitIgnore creates a .gitignore template
func (f *CrossplaneTemplateFactory) GitIgnore() (TemplateProduct, error) {
	return f.CreateInitTemplate(GitIgnoreTemplateType)
}

// MainGo creates a main.go template
func (f *CrossplaneTemplateFactory) MainGo() (TemplateProduct, error) {
	return f.CreateInitTemplate(MainGoTemplateType)
}

// ProviderConfigTypes creates ProviderConfig types template
func (f *CrossplaneTemplateFactory) ProviderConfigTypes() (TemplateProduct, error) {
	return f.CreateInitTemplate(ProviderConfigTypesType)
}

// ProviderConfigRegister creates ProviderConfig registration template
func (f *CrossplaneTemplateFactory) ProviderConfigRegister() (TemplateProduct, error) {
	return f.CreateInitTemplate(ProviderConfigRegisterType)
}

// CrossplanePackage creates package/crossplane.yaml template
func (f *CrossplaneTemplateFactory) CrossplanePackage() (TemplateProduct, error) {
	return f.CreateInitTemplate(CrossplanePackageType)
}

// ConfigController creates config controller template
func (f *CrossplaneTemplateFactory) ConfigController() (TemplateProduct, error) {
	return f.CreateInitTemplate(ConfigControllerType)
}

// ControllerRegister creates controller registration template
func (f *CrossplaneTemplateFactory) ControllerRegister() (TemplateProduct, error) {
	return f.CreateInitTemplate(ControllerRegisterType)
}

// ClusterDockerfile creates cluster Dockerfile template
func (f *CrossplaneTemplateFactory) ClusterDockerfile() (TemplateProduct, error) {
	return f.CreateInitTemplate(ClusterDockerfileType)
}

// ClusterMakefile creates cluster Makefile template
func (f *CrossplaneTemplateFactory) ClusterMakefile() (TemplateProduct, error) {
	return f.CreateInitTemplate(ClusterMakefileType)
}

// VersionGo creates version.go template
func (f *CrossplaneTemplateFactory) VersionGo() (TemplateProduct, error) {
	return f.CreateInitTemplate(VersionGoType)
}

// License creates LICENSE template
func (f *CrossplaneTemplateFactory) License() (TemplateProduct, error) {
	return f.CreateStaticTemplate(LicenseType)
}

// DocGo creates doc.go template for v1alpha1 package
func (f *CrossplaneTemplateFactory) DocGo() (TemplateProduct, error) {
	return f.CreateInitTemplate(DocGoType)
}

// APITypes creates API types template
func (f *CrossplaneTemplateFactory) APITypes(force bool, res interface{}) (TemplateProduct, error) {
	opts := []Option{WithForce(force)}
	if res != nil {
		if r, ok := res.(*resource.Resource); ok {
			opts = append(opts, WithResource(r))
		}
	}
	return f.CreateAPITemplate(APITypesTemplateType, opts...)
}

// APIGroup creates API group template
func (f *CrossplaneTemplateFactory) APIGroup(res interface{}) (TemplateProduct, error) {
	opts := []Option{}
	if res != nil {
		if r, ok := res.(*resource.Resource); ok {
			opts = append(opts, WithResource(r))
		}
	}
	return f.CreateAPITemplate(APIGroupTemplateType, opts...)
}

// Controller creates controller template
func (f *CrossplaneTemplateFactory) Controller(force bool, res interface{}) (TemplateProduct, error) {
	opts := []Option{WithForce(force)}
	if res != nil {
		if r, ok := res.(*resource.Resource); ok {
			opts = append(opts, WithResource(r))
		}
	}
	return f.CreateAPITemplate(ControllerTemplateType, opts...)
}

// ExamplesProviderConfig returns the provider config examples template
func (f *CrossplaneTemplateFactory) ExamplesProviderConfig() (TemplateProduct, error) {
	return f.CreateInitTemplate(ExamplesProviderConfigTemplateType)
}

// ExamplesManagedResource returns the managed resource examples template
func (f *CrossplaneTemplateFactory) ExamplesManagedResource(res interface{}) (TemplateProduct, error) {
	opts := []Option{}
	if res != nil {
		if r, ok := res.(*resource.Resource); ok {
			opts = append(opts, WithResource(r))
		}
	}
	return f.CreateAPITemplate(ExamplesManagedResourceTemplateType, opts...)
}