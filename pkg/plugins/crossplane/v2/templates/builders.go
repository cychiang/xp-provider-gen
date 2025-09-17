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
	"strings"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
)

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

	typeStr := strings.ToLower(string(b.templateType))
	switch {
	case strings.Contains(typeStr, "gomod"):
		product = &GoModTemplateProduct{BaseTemplateProduct: NewBaseTemplateProduct(b.templateType)}
	case strings.Contains(typeStr, "makefile"):
		product = &MakefileTemplateProduct{BaseTemplateProduct: NewBaseTemplateProduct(b.templateType)}
	case strings.Contains(typeStr, "readme"):
		product = &READMETemplateProduct{BaseTemplateProduct: NewBaseTemplateProduct(b.templateType)}
	case strings.Contains(typeStr, "gitignore"):
		product = &GitIgnoreTemplateProduct{BaseTemplateProduct: NewBaseTemplateProduct(b.templateType)}
	case strings.Contains(typeStr, "maingo"):
		product = &MainGoTemplateProduct{BaseTemplateProduct: NewBaseTemplateProduct(b.templateType)}
	case strings.Contains(typeStr, "providerconfig") && strings.Contains(typeStr, "types"):
		product = &ProviderConfigTypesTemplateProduct{BaseTemplateProduct: NewBaseTemplateProduct(b.templateType)}
	case strings.Contains(typeStr, "v1alpha1") && strings.Contains(typeStr, "register"):
		product = &ProviderConfigRegisterTemplateProduct{BaseTemplateProduct: NewBaseTemplateProduct(b.templateType)}
	case strings.Contains(typeStr, "apis") && strings.Contains(typeStr, "register"):
		product = &APIsTemplateProduct{BaseTemplateProduct: NewBaseTemplateProduct(b.templateType)}
	case strings.Contains(typeStr, "generatego"):
		product = &GenerateGoTemplateProduct{BaseTemplateProduct: NewBaseTemplateProduct(b.templateType)}
	case strings.Contains(typeStr, "boilerplate"):
		product = &BoilerplateTemplateProduct{BaseTemplateProduct: NewBaseTemplateProduct(b.templateType)}
	case strings.Contains(typeStr, "crossplane") && strings.Contains(typeStr, "yaml"):
		product = &CrossplanePackageTemplateProduct{BaseTemplateProduct: NewBaseTemplateProduct(b.templateType)}
	case strings.Contains(typeStr, "controller") && strings.Contains(typeStr, "config"):
		product = &ConfigControllerTemplateProduct{BaseTemplateProduct: NewBaseTemplateProduct(b.templateType)}
	case strings.Contains(typeStr, "controller") && strings.Contains(typeStr, "register"):
		product = &ControllerRegisterTemplateProduct{BaseTemplateProduct: NewBaseTemplateProduct(b.templateType)}
	case strings.Contains(typeStr, "cluster") && strings.Contains(typeStr, "dockerfile"):
		product = &ClusterDockerfileTemplateProduct{BaseTemplateProduct: NewBaseTemplateProduct(b.templateType)}
	case strings.Contains(typeStr, "cluster") && strings.Contains(typeStr, "makefile"):
		product = &ClusterMakefileTemplateProduct{BaseTemplateProduct: NewBaseTemplateProduct(b.templateType)}
	case strings.Contains(typeStr, "versiongo"):
		product = &VersionGoTemplateProduct{BaseTemplateProduct: NewBaseTemplateProduct(b.templateType)}
	case strings.Contains(typeStr, "license"):
		product = &LicenseTemplateProduct{BaseTemplateProduct: NewBaseTemplateProduct(b.templateType)}
	case strings.Contains(typeStr, "docgo"):
		product = &DocGoTemplateProduct{BaseTemplateProduct: NewBaseTemplateProduct(b.templateType)}
	case strings.Contains(typeStr, "examplesproviderconfig"):
		product = &ExamplesProviderConfigTemplateProduct{BaseTemplateProduct: NewBaseTemplateProduct(b.templateType)}
	default:
		return nil, fmt.Errorf("unsupported init template type: %s", b.templateType)
	}

	if err := product.Configure(cfg); err != nil {
		return nil, fmt.Errorf("failed to configure template: %w", err)
	}

	if baseProduct, ok := product.(interface{ GetBase() *BaseTemplateProduct }); ok {
		base := baseProduct.GetBase()
		if options.Force {
			base.SetForce(options.Force)
		}
		if options.CustomData != nil {
			base.SetCustomData(options.CustomData)
		}
	}

	if defaultSetter, ok := product.(interface{ SetTemplateDefaults() error }); ok {
		if err := defaultSetter.SetTemplateDefaults(); err != nil {
			return nil, fmt.Errorf("failed to set template defaults: %w", err)
		}
	}

	return product, nil
}

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

	typeStr := strings.ToLower(string(b.templateType))
	switch {
	case strings.Contains(typeStr, "groupversiontypes"):
		product = &APITypesTemplateProduct{BaseTemplateProduct: NewBaseTemplateProduct(b.templateType)}
	case strings.Contains(typeStr, "groupversiongroupversion"):
		product = &APIGroupTemplateProduct{BaseTemplateProduct: NewBaseTemplateProduct(b.templateType)}
	case strings.Contains(typeStr, "controllerkindcontroller"):
		product = &ControllerTemplateProduct{BaseTemplateProduct: NewBaseTemplateProduct(b.templateType)}
	case strings.Contains(typeStr, "examplesgroupkind"):
		product = &ExamplesManagedResourceTemplateProduct{BaseTemplateProduct: NewBaseTemplateProduct(b.templateType)}
	default:
		return nil, fmt.Errorf("unsupported API template type: %s", b.templateType)
	}

	if err := product.Configure(cfg); err != nil {
		return nil, fmt.Errorf("failed to configure template: %w", err)
	}

	if err := product.SetResource(options.Resource); err != nil {
		return nil, fmt.Errorf("failed to set resource: %w", err)
	}

	if baseProduct, ok := product.(interface{ GetBase() *BaseTemplateProduct }); ok {
		base := baseProduct.GetBase()
		if options.Force {
			base.SetForce(options.Force)
		}
		if options.CustomData != nil {
			base.SetCustomData(options.CustomData)
		}
	}

	if defaultSetter, ok := product.(interface{ SetTemplateDefaults() error }); ok {
		if err := defaultSetter.SetTemplateDefaults(); err != nil {
			return nil, fmt.Errorf("failed to set template defaults: %w", err)
		}
	}

	return product, nil
}

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

	typeStr := strings.ToLower(string(b.templateType))
	switch {
	case strings.Contains(typeStr, "license"):
		product = &LicenseTemplateProduct{BaseTemplateProduct: NewBaseTemplateProduct(b.templateType)}
	default:
		return nil, fmt.Errorf("unsupported static template type: %s", b.templateType)
	}

	if err := product.Configure(cfg); err != nil {
		return nil, fmt.Errorf("failed to configure template: %w", err)
	}

	if baseProduct, ok := product.(interface{ GetBase() *BaseTemplateProduct }); ok {
		base := baseProduct.GetBase()
		if options.Force {
			base.SetForce(options.Force)
		}
		if options.CustomData != nil {
			base.SetCustomData(options.CustomData)
		}
	}

	if defaultSetter, ok := product.(interface{ SetTemplateDefaults() error }); ok {
		if err := defaultSetter.SetTemplateDefaults(); err != nil {
			return nil, fmt.Errorf("failed to set template defaults: %w", err)
		}
	}

	return product, nil
}