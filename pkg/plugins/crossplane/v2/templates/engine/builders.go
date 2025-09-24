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
	"fmt"
	"strings"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"

	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/core"
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
	options := parseOptions(opts)

	info, err := findTemplateInfoByCategory(InitCategory, b.templateType)
	if err != nil {
		return nil, fmt.Errorf("failed to find template info: %w", err)
	}

	projectName := core.ExtractProjectName(cfg)
	replacements := map[string]string{
		"IMAGENAME": projectName,
	}

	product := createTemplateProduct(b.templateType, info, replacements)

	if err := configureTemplateProduct(product, cfg, options); err != nil {
		return nil, err
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
	options := parseOptions(opts)

	if options.Resource == nil {
		return nil, fmt.Errorf("resource is required for API templates")
	}

	info, err := findTemplateInfoByCategory(APICategory, b.templateType)
	if err != nil {
		return nil, fmt.Errorf("failed to find template info: %w", err)
	}

	projectName := core.ExtractProjectName(cfg)
	replacements := map[string]string{
		"GROUP":     strings.ToLower(options.Resource.Group),
		"VERSION":   options.Resource.Version,
		"KIND":      strings.ToLower(options.Resource.Kind),
		"IMAGENAME": projectName,
	}

	product := createTemplateProduct(b.templateType, info, replacements)

	if err := configureTemplateProduct(product, cfg, options); err != nil {
		return nil, err
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
	options := parseOptions(opts)

	info, err := findTemplateInfoByCategory(StaticCategory, b.templateType)
	if err != nil {
		return nil, fmt.Errorf("failed to find template info: %w", err)
	}

	projectName := core.ExtractProjectName(cfg)
	replacements := map[string]string{
		"IMAGENAME": projectName,
	}

	if options.Resource != nil {
		replacements["GROUP"] = strings.ToLower(options.Resource.Group)
		replacements["VERSION"] = options.Resource.Version
		replacements["KIND"] = strings.ToLower(options.Resource.Kind)
	}

	product := createTemplateProduct(b.templateType, info, replacements)

	if err := configureTemplateProduct(product, cfg, options); err != nil {
		return nil, err
	}

	return product, nil
}
