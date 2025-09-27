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

// BuildStrategy defines different strategies for building templates.
type BuildStrategy interface {
	GetCategory() TemplateCategory
	ValidateOptions(options *TemplateOptions) error
	GenerateReplacements(cfg config.Config, options *TemplateOptions) (map[string]string, error)
}

// BaseTemplateBuilder provides unified template building functionality.
type BaseTemplateBuilder struct {
	templateType TemplateType
	strategy     BuildStrategy
}

// NewBaseTemplateBuilder creates a new template builder with the specified strategy.
func NewBaseTemplateBuilder(templateType TemplateType, strategy BuildStrategy) TemplateBuilder {
	return &BaseTemplateBuilder{
		templateType: templateType,
		strategy:     strategy,
	}
}

func (b *BaseTemplateBuilder) GetTemplateType() TemplateType {
	return b.templateType
}

func (b *BaseTemplateBuilder) Build(cfg config.Config, opts ...Option) (TemplateProduct, error) {
	options := parseOptions(opts)

	// Validate options using strategy
	if err := b.strategy.ValidateOptions(options); err != nil {
		return nil, err
	}

	// Find template info using strategy's category
	info, err := findTemplateInfoByCategory(b.strategy.GetCategory(), b.templateType)
	if err != nil {
		return nil, fmt.Errorf("failed to find template info: %w", err)
	}

	// Generate replacements using strategy
	replacements, err := b.strategy.GenerateReplacements(cfg, options)
	if err != nil {
		return nil, err
	}

	// Create and configure template product
	product := createTemplateProduct(b.templateType, info, replacements)

	if err := configureTemplateProduct(product, cfg, options); err != nil {
		return nil, err
	}

	return product, nil
}

// InitBuildStrategy implements strategy for init templates.
type InitBuildStrategy struct{}

func (s *InitBuildStrategy) GetCategory() TemplateCategory {
	return InitCategory
}

func (s *InitBuildStrategy) ValidateOptions(_ *TemplateOptions) error {
	// Init templates don't require special validation
	return nil
}

func (s *InitBuildStrategy) GenerateReplacements(cfg config.Config, _ *TemplateOptions) (map[string]string, error) {
	projectName := core.ExtractProjectName(cfg)
	return map[string]string{
		"IMAGENAME": projectName,
	}, nil
}

// APIBuildStrategy implements strategy for API templates.
type APIBuildStrategy struct{}

func (s *APIBuildStrategy) GetCategory() TemplateCategory {
	return APICategory
}

func (s *APIBuildStrategy) ValidateOptions(options *TemplateOptions) error {
	if options.Resource == nil {
		return fmt.Errorf("resource is required for API templates")
	}
	return nil
}

func (s *APIBuildStrategy) GenerateReplacements(
	cfg config.Config, options *TemplateOptions,
) (map[string]string, error) {
	projectName := core.ExtractProjectName(cfg)
	return map[string]string{
		"GROUP":     strings.ToLower(options.Resource.Group),
		"VERSION":   options.Resource.Version,
		"KIND":      strings.ToLower(options.Resource.Kind),
		"IMAGENAME": projectName,
	}, nil
}

// StaticBuildStrategy implements strategy for static templates.
type StaticBuildStrategy struct{}

func (s *StaticBuildStrategy) GetCategory() TemplateCategory {
	return StaticCategory
}

func (s *StaticBuildStrategy) ValidateOptions(_ *TemplateOptions) error {
	// Static templates don't require special validation
	return nil
}

func (s *StaticBuildStrategy) GenerateReplacements(
	cfg config.Config, options *TemplateOptions,
) (map[string]string, error) {
	projectName := core.ExtractProjectName(cfg)
	replacements := map[string]string{
		"IMAGENAME": projectName,
	}

	// Add resource-specific replacements if available
	if options.Resource != nil {
		replacements["GROUP"] = strings.ToLower(options.Resource.Group)
		replacements["VERSION"] = options.Resource.Version
		replacements["KIND"] = strings.ToLower(options.Resource.Kind)
	}

	return replacements, nil
}
