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
)

// GenericTemplateProduct is a universal template product that works with any template
// discovered through the autodiscovery system
type GenericTemplateProduct struct {
	*BaseTemplateProduct
	loader       *TemplateLoader
	outputPath   string
	templatePath string
}

// NewGenericTemplateProduct creates a new generic template product
func NewGenericTemplateProduct(templateType TemplateType, outputPath, templatePath string) *GenericTemplateProduct {
	return &GenericTemplateProduct{
		BaseTemplateProduct: NewBaseTemplateProduct(templateType),
		loader:              NewTemplateLoader(),
		outputPath:          outputPath,
		templatePath:        templatePath,
	}
}

// SetTemplateDefaults loads the template content from the scaffolds directory
func (t *GenericTemplateProduct) SetTemplateDefaults() error {
	if t.Path == "" {
		t.Path = t.outputPath
	}

	// Load template content using the original template path
	templateContent, err := t.loader.LoadTemplate(t.templatePath)
	if err != nil {
		return fmt.Errorf("failed to load template %s: %w", t.templatePath, err)
	}

	t.TemplateBody = templateContent
	return nil
}

// GetOutputPath returns the output path for this template
func (t *GenericTemplateProduct) GetOutputPath() string {
	return t.outputPath
}

// generateOutputPath converts a template info to its output path with variable replacements
func generateOutputPath(info TemplateInfo, replacements map[string]string) string {
	// The OutputDir is like "project/Makefile" or "apis/register.go"
	outputPath := info.OutputDir

	// Handle root files specially - remove the "project/" prefix
	if strings.HasPrefix(outputPath, "project/") {
		outputPath = strings.TrimPrefix(outputPath, "project/")
	}

	// Apply variable replacements for uppercase placeholders
	if replacements != nil {
		for placeholder, value := range replacements {
			outputPath = strings.ReplaceAll(outputPath, placeholder, value)
		}
	}

	return outputPath
}