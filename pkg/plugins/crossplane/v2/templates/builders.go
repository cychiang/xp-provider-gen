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
)

// InitTemplateBuilder builds init templates
type InitTemplateBuilder struct {
	templateType TemplateType
}

func NewInitTemplateBuilder(templateType TemplateType) TemplateBuilder {
	return &InitTemplateBuilder{templateType: templateType}
}

func (b *InitTemplateBuilder) GetTemplateType() TemplateType {
	return b.templateType
}

func (b *InitTemplateBuilder) Build(cfg config.Config, opts ...Option) (TemplateProduct, error) {
	options := &TemplateOptions{}
	for _, opt := range opts {
		opt(options)
	}

	var product TemplateProduct

	switch b.templateType {
	case GoModTemplateType:
		product = &GoModTemplateProduct{BaseTemplateProduct: NewBaseTemplateProduct(b.templateType)}
	case MakefileTemplateType:
		product = &MakefileTemplateProduct{BaseTemplateProduct: NewBaseTemplateProduct(b.templateType)}
	case READMETemplateType:
		product = &READMETemplateProduct{BaseTemplateProduct: NewBaseTemplateProduct(b.templateType)}
	case GitIgnoreTemplateType:
		product = &GitIgnoreTemplateProduct{BaseTemplateProduct: NewBaseTemplateProduct(b.templateType)}
	case MainGoTemplateType:
		product = &MainGoTemplateProduct{BaseTemplateProduct: NewBaseTemplateProduct(b.templateType)}
	case APIsTemplateType:
		product = &APIsTemplateProduct{BaseTemplateProduct: NewBaseTemplateProduct(b.templateType)}
	case GenerateGoTemplateType:
		product = &GenerateGoTemplateProduct{BaseTemplateProduct: NewBaseTemplateProduct(b.templateType)}
	case BoilerplateTemplateType:
		product = &BoilerplateTemplateProduct{BaseTemplateProduct: NewBaseTemplateProduct(b.templateType)}
	case ProviderConfigTypesType:
		product = &ProviderConfigTypesTemplateProduct{BaseTemplateProduct: NewBaseTemplateProduct(b.templateType)}
	case ProviderConfigRegisterType:
		product = &ProviderConfigRegisterTemplateProduct{BaseTemplateProduct: NewBaseTemplateProduct(b.templateType)}
	case CrossplanePackageType:
		product = &CrossplanePackageTemplateProduct{BaseTemplateProduct: NewBaseTemplateProduct(b.templateType)}
	case ConfigControllerType:
		product = &ConfigControllerTemplateProduct{BaseTemplateProduct: NewBaseTemplateProduct(b.templateType)}
	case ControllerRegisterType:
		product = &ControllerRegisterTemplateProduct{BaseTemplateProduct: NewBaseTemplateProduct(b.templateType)}
	case ClusterDockerfileType:
		product = &ClusterDockerfileTemplateProduct{BaseTemplateProduct: NewBaseTemplateProduct(b.templateType)}
	case ClusterMakefileType:
		product = &ClusterMakefileTemplateProduct{BaseTemplateProduct: NewBaseTemplateProduct(b.templateType)}
	case VersionGoType:
		product = &VersionGoTemplateProduct{BaseTemplateProduct: NewBaseTemplateProduct(b.templateType)}
	case LicenseType:
		product = &LicenseTemplateProduct{BaseTemplateProduct: NewBaseTemplateProduct(b.templateType)}
	default:
		return nil, fmt.Errorf("unsupported init template type: %s", b.templateType)
	}

	// Configure the product
	if err := product.Configure(cfg); err != nil {
		return nil, fmt.Errorf("failed to configure template: %w", err)
	}

	// Set options by getting the base template
	if baseProduct, ok := product.(interface{ GetBase() *BaseTemplateProduct }); ok {
		base := baseProduct.GetBase()
		if options.Force {
			base.SetForce(options.Force)
		}
		if options.CustomData != nil {
			base.SetCustomData(options.CustomData)
		}
	}

	// Call SetTemplateDefaults to set paths and template bodies
	if defaultSetter, ok := product.(interface{ SetTemplateDefaults() error }); ok {
		if err := defaultSetter.SetTemplateDefaults(); err != nil {
			return nil, fmt.Errorf("failed to set template defaults: %w", err)
		}
	}

	return product, nil
}

// APITemplateBuilder builds API templates
type APITemplateBuilder struct {
	templateType TemplateType
}

func NewAPITemplateBuilder(templateType TemplateType) TemplateBuilder {
	return &APITemplateBuilder{templateType: templateType}
}

func (b *APITemplateBuilder) GetTemplateType() TemplateType {
	return b.templateType
}

func (b *APITemplateBuilder) Build(cfg config.Config, opts ...Option) (TemplateProduct, error) {
	options := &TemplateOptions{}
	for _, opt := range opts {
		opt(options)
	}

	if options.Resource == nil {
		return nil, fmt.Errorf("resource is required for API templates")
	}

	var product TemplateProduct

	switch b.templateType {
	case APITypesTemplateType:
		product = &APITypesTemplateProduct{BaseTemplateProduct: NewBaseTemplateProduct(b.templateType)}
	case APIGroupTemplateType:
		product = &APIGroupTemplateProduct{BaseTemplateProduct: NewBaseTemplateProduct(b.templateType)}
	case ControllerTemplateType:
		product = &ControllerTemplateProduct{BaseTemplateProduct: NewBaseTemplateProduct(b.templateType)}
	default:
		return nil, fmt.Errorf("unsupported API template type: %s", b.templateType)
	}

	// Configure the product
	if err := product.Configure(cfg); err != nil {
		return nil, fmt.Errorf("failed to configure template: %w", err)
	}

	// Set resource
	if err := product.SetResource(options.Resource); err != nil {
		return nil, fmt.Errorf("failed to set resource: %w", err)
	}

	// Set options by getting the base template
	if baseProduct, ok := product.(interface{ GetBase() *BaseTemplateProduct }); ok {
		base := baseProduct.GetBase()
		if options.Force {
			base.SetForce(options.Force)
		}
		if options.CustomData != nil {
			base.SetCustomData(options.CustomData)
		}
	}

	// Call SetTemplateDefaults to set paths and template bodies
	if defaultSetter, ok := product.(interface{ SetTemplateDefaults() error }); ok {
		if err := defaultSetter.SetTemplateDefaults(); err != nil {
			return nil, fmt.Errorf("failed to set template defaults: %w", err)
		}
	}

	return product, nil
}

// StaticTemplateBuilder builds static templates
type StaticTemplateBuilder struct {
	templateType TemplateType
}

func NewStaticTemplateBuilder(templateType TemplateType) TemplateBuilder {
	return &StaticTemplateBuilder{templateType: templateType}
}

func (b *StaticTemplateBuilder) GetTemplateType() TemplateType {
	return b.templateType
}

func (b *StaticTemplateBuilder) Build(cfg config.Config, opts ...Option) (TemplateProduct, error) {
	options := &TemplateOptions{}
	for _, opt := range opts {
		opt(options)
	}

	var product TemplateProduct

	switch b.templateType {
	case LicenseType:
		product = &LicenseTemplateProduct{BaseTemplateProduct: NewBaseTemplateProduct(b.templateType)}
	default:
		return nil, fmt.Errorf("unsupported static template type: %s", b.templateType)
	}

	// Configure the product
	if err := product.Configure(cfg); err != nil {
		return nil, fmt.Errorf("failed to configure template: %w", err)
	}

	// Set options by getting the base template
	if baseProduct, ok := product.(interface{ GetBase() *BaseTemplateProduct }); ok {
		base := baseProduct.GetBase()
		if options.Force {
			base.SetForce(options.Force)
		}
		if options.CustomData != nil {
			base.SetCustomData(options.CustomData)
		}
	}

	// Call SetTemplateDefaults to set paths and template bodies
	if defaultSetter, ok := product.(interface{ SetTemplateDefaults() error }); ok {
		if err := defaultSetter.SetTemplateDefaults(); err != nil {
			return nil, fmt.Errorf("failed to set template defaults: %w", err)
		}
	}

	return product, nil
}