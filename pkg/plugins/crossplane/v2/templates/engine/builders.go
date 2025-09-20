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

	// Find template info for this template type
	info, err := b.findTemplateInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to find template info: %w", err)
	}

	// Create replacement map for init template variables (basic replacements only)
	projectName := cfg.GetProjectName()

	// Fallback: extract project name from repository URL if GetProjectName() is empty
	if projectName == "" {
		repo := cfg.GetRepository()
		if repo != "" {
			// Extract last part of repo URL: "github.com/example/provider-test" -> "provider-test"
			parts := strings.Split(repo, "/")
			if len(parts) > 0 {
				projectName = parts[len(parts)-1]
			}
		}
	}

	replacements := map[string]string{
		"IMAGENAME": projectName, // Use project name for image
	}

	// Create generic template product
	outputPath := generateOutputPath(info, replacements)
	templatePath := strings.TrimPrefix(info.Path, "files/")
	product := NewGenericTemplateProduct(b.templateType, outputPath, templatePath)

	if err := product.Configure(cfg); err != nil {
		return nil, fmt.Errorf("failed to configure template: %w", err)
	}

	base := product.GetBase()
	if options.Force {
		base.SetForce(options.Force)
	}
	if options.CustomData != nil {
		base.SetCustomData(options.CustomData)
	}

	if err := product.SetTemplateDefaults(); err != nil {
		return nil, fmt.Errorf("failed to set template defaults: %w", err)
	}

	return product, nil
}

func (b *InitTemplateBuilder) findTemplateInfo() (TemplateInfo, error) {
	// Walk through the templates to find the one matching our type
	var foundInfo TemplateInfo
	found := false

	err := walkTemplateFS("files", func(path string, isDir bool) error {
		if isDir || !strings.HasSuffix(path, ".tmpl") {
			return nil
		}

		info := AnalyzeTemplatePath(path)
		if info.Category == InitCategory && TemplateType(info.GenerateTemplateType()) == b.templateType {
			foundInfo = info
			found = true
		}
		return nil
	})

	if err != nil {
		return TemplateInfo{}, err
	}

	if !found {
		return TemplateInfo{}, fmt.Errorf("template not found for type: %s", b.templateType)
	}

	return foundInfo, nil
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

	// Find template info for this template type
	info, err := b.findTemplateInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to find template info: %w", err)
	}

	// Create replacement map for API template variables
	projectName := cfg.GetProjectName()

	// Fallback: extract project name from repository URL if GetProjectName() is empty
	if projectName == "" {
		repo := cfg.GetRepository()
		if repo != "" {
			// Extract last part of repo URL: "github.com/example/provider-test" -> "provider-test"
			parts := strings.Split(repo, "/")
			if len(parts) > 0 {
				projectName = parts[len(parts)-1]
			}
		}
	}

	replacements := map[string]string{
		"GROUP":     strings.ToLower(options.Resource.Group),
		"VERSION":   options.Resource.Version,
		"KIND":      strings.ToLower(options.Resource.Kind),
		"IMAGENAME": projectName, // Use project name for image
	}

	// Create generic template product
	outputPath := generateOutputPath(info, replacements)
	templatePath := strings.TrimPrefix(info.Path, "files/")
	product := NewGenericTemplateProduct(b.templateType, outputPath, templatePath)

	if err := product.Configure(cfg); err != nil {
		return nil, fmt.Errorf("failed to configure template: %w", err)
	}

	if err := product.SetResource(options.Resource); err != nil {
		return nil, fmt.Errorf("failed to set resource: %w", err)
	}

	base := product.GetBase()
	if options.Force {
		base.SetForce(options.Force)
	}
	if options.CustomData != nil {
		base.SetCustomData(options.CustomData)
	}

	if err := product.SetTemplateDefaults(); err != nil {
		return nil, fmt.Errorf("failed to set template defaults: %w", err)
	}

	return product, nil
}

func (b *APITemplateBuilder) findTemplateInfo() (TemplateInfo, error) {
	// Walk through the templates to find the one matching our type
	var foundInfo TemplateInfo
	found := false

	err := walkTemplateFS("files", func(path string, isDir bool) error {
		if isDir || !strings.HasSuffix(path, ".tmpl") {
			return nil
		}

		info := AnalyzeTemplatePath(path)
		if info.Category == APICategory && TemplateType(info.GenerateTemplateType()) == b.templateType {
			foundInfo = info
			found = true
		}
		return nil
	})

	if err != nil {
		return TemplateInfo{}, err
	}

	if !found {
		return TemplateInfo{}, fmt.Errorf("template not found for type: %s", b.templateType)
	}

	return foundInfo, nil
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

	// Find template info for this template type
	info, err := b.findTemplateInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to find template info: %w", err)
	}

	// Create replacement map for static template variables (basic replacements only)
	projectName := cfg.GetProjectName()

	// Fallback: extract project name from repository URL if GetProjectName() is empty
	if projectName == "" {
		repo := cfg.GetRepository()
		if repo != "" {
			// Extract last part of repo URL: "github.com/example/provider-test" -> "provider-test"
			parts := strings.Split(repo, "/")
			if len(parts) > 0 {
				projectName = parts[len(parts)-1]
			}
		}
	}

	replacements := map[string]string{
		"IMAGENAME": projectName, // Use project name for image
	}

	// Add API-specific replacements if resource is available
	if options.Resource != nil {
		replacements["GROUP"] = strings.ToLower(options.Resource.Group)
		replacements["VERSION"] = options.Resource.Version
		replacements["KIND"] = strings.ToLower(options.Resource.Kind)
	}

	// Create generic template product
	outputPath := generateOutputPath(info, replacements)
	templatePath := strings.TrimPrefix(info.Path, "files/")
	product := NewGenericTemplateProduct(b.templateType, outputPath, templatePath)

	if err := product.Configure(cfg); err != nil {
		return nil, fmt.Errorf("failed to configure template: %w", err)
	}

	base := product.GetBase()
	if options.Force {
		base.SetForce(options.Force)
	}
	if options.CustomData != nil {
		base.SetCustomData(options.CustomData)
	}

	if err := product.SetTemplateDefaults(); err != nil {
		return nil, fmt.Errorf("failed to set template defaults: %w", err)
	}

	return product, nil
}

func (b *StaticTemplateBuilder) findTemplateInfo() (TemplateInfo, error) {
	// Walk through the templates to find the one matching our type
	var foundInfo TemplateInfo
	found := false

	err := walkTemplateFS("files", func(path string, isDir bool) error {
		if isDir || !strings.HasSuffix(path, ".tmpl") {
			return nil
		}

		info := AnalyzeTemplatePath(path)
		if info.Category == StaticCategory && TemplateType(info.GenerateTemplateType()) == b.templateType {
			foundInfo = info
			found = true
		}
		return nil
	})

	if err != nil {
		return TemplateInfo{}, err
	}

	if !found {
		return TemplateInfo{}, fmt.Errorf("template not found for type: %s", b.templateType)
	}

	return foundInfo, nil
}