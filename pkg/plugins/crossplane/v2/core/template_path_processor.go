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

package core

import (
	"path/filepath"
	"strings"
)

// TemplatePathProcessor provides utilities for processing template paths.
type TemplatePathProcessor struct{}

// NewTemplatePathProcessor creates a new template path processor.
func NewTemplatePathProcessor() *TemplatePathProcessor {
	return &TemplatePathProcessor{}
}

// IsTemplateFile checks if a file path represents a template file.
func (p *TemplatePathProcessor) IsTemplateFile(path string) bool {
	return strings.HasSuffix(path, ".tmpl")
}

// CleanTemplatePath removes the "files/" prefix from a template path.
func (p *TemplatePathProcessor) CleanTemplatePath(path string) string {
	return strings.TrimPrefix(path, "files/")
}

// GetTemplateBaseName extracts the base name from a template path without the .tmpl extension.
func (p *TemplatePathProcessor) GetTemplateBaseName(path string) string {
	cleanPath := p.CleanTemplatePath(path)
	return strings.TrimSuffix(filepath.Base(cleanPath), ".tmpl")
}

// GetOutputPath converts a template path to its output path by removing .tmpl extension.
func (p *TemplatePathProcessor) GetOutputPath(templatePath string) string {
	cleanPath := p.CleanTemplatePath(templatePath)
	return strings.TrimSuffix(cleanPath, ".tmpl")
}

// ConvertToFilesystemPath converts a template name/path to filesystem path within the template FS.
func (p *TemplatePathProcessor) ConvertToFilesystemPath(templatePath string) string {
	// Ensure we always use forward slashes for embedded filesystem paths
	cleanPath := strings.ReplaceAll(templatePath, "\\", "/")
	return filepath.Join("files", cleanPath)
}

// ApplyVariableReplacements applies variable replacements to a path.
func (p *TemplatePathProcessor) ApplyVariableReplacements(path string, replacements map[string]string) string {
	result := path
	for placeholder, value := range replacements {
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}

// GenerateOutputPath converts a template path to its final output path with variable replacements.
func (p *TemplatePathProcessor) GenerateOutputPath(templatePath string, replacements map[string]string) string {
	outputPath := p.GetOutputPath(templatePath)

	// Handle root files specially - remove the "project/" prefix
	outputPath = strings.TrimPrefix(outputPath, "project/")

	// Apply variable replacements
	return p.ApplyVariableReplacements(outputPath, replacements)
}

// SplitPathComponents splits a path into its directory components.
func (p *TemplatePathProcessor) SplitPathComponents(path string) []string {
	cleanPath := strings.Trim(path, "/")
	if cleanPath == "" {
		return []string{}
	}
	return strings.Split(cleanPath, "/")
}

// NormalizePath normalizes a path by cleaning up redundant separators and components.
func (p *TemplatePathProcessor) NormalizePath(path string) string {
	// Use filepath.Clean but convert back to forward slashes for consistency
	cleaned := filepath.Clean(path)
	return strings.ReplaceAll(cleaned, "\\", "/")
}

// PathHasPattern checks if a path contains any of the given patterns.
func (p *TemplatePathProcessor) PathHasPattern(path string, patterns []string) bool {
	for _, pattern := range patterns {
		if strings.Contains(path, pattern) {
			return true
		}
	}
	return false
}

// ExtractPathVariables extracts variable placeholders from a path (e.g., KIND, GROUP, VERSION).
func (p *TemplatePathProcessor) ExtractPathVariables(path string) []string {
	var variables []string
	components := p.SplitPathComponents(path)

	for _, component := range components {
		if strings.ToUpper(component) == component && len(component) > 1 {
			// This looks like a variable placeholder (all uppercase)
			variables = append(variables, component)
		}
	}

	return variables
}
