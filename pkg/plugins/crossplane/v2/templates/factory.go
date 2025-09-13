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
	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

// TemplateFactory creates templates with minimal configuration
// This is the main entry point for creating all template types
type TemplateFactory struct {
	config config.Config
}

// NewFactory creates a new template factory
func NewFactory(cfg config.Config) *TemplateFactory {
	return &TemplateFactory{config: cfg}
}

// Init Template Methods - delegates to individual template functions

// GoMod creates go.mod template
func (f *TemplateFactory) GoMod() machinery.Template {
	return GoMod(f.config)
}

// Makefile creates Makefile template  
func (f *TemplateFactory) Makefile() machinery.Template {
	return Makefile(f.config)
}

// GitIgnore creates .gitignore template
func (f *TemplateFactory) GitIgnore() machinery.Template {
	return GitIgnore(f.config)
}

// README creates README.md template
func (f *TemplateFactory) README() machinery.Template {
	return README(f.config)
}

// MainGo creates cmd/provider/main.go template
func (f *TemplateFactory) MainGo() machinery.Template {
	return MainGo(f.config)
}

// ProviderConfigTypes creates ProviderConfig types (placeholder)
func (f *TemplateFactory) ProviderConfigTypes() machinery.Template {
	return SimpleFile(f.config, "apis/v1alpha1/types.go", "package v1alpha1")
}

// ProviderConfigRegister creates ProviderConfig registration (placeholder)
func (f *TemplateFactory) ProviderConfigRegister() machinery.Template {
	return SimpleFile(f.config, "apis/v1alpha1/register.go", "package v1alpha1")
}

// CrossplanePackage creates package/crossplane.yaml (placeholder)
func (f *TemplateFactory) CrossplanePackage() machinery.Template {
	return SimpleFile(f.config, "package/crossplane.yaml", "apiVersion: meta.pkg.crossplane.io/v1alpha1\nkind: Provider")
}

// ConfigController creates config controller (placeholder)
func (f *TemplateFactory) ConfigController() machinery.Template {
	return SimpleFile(f.config, "internal/controller/config/config.go", "package config")
}

// ControllerRegister creates controller registration file (placeholder)
func (f *TemplateFactory) ControllerRegister() machinery.Template {
	return SimpleFile(f.config, "internal/controller/register.go", "package controller")
}

// VersionGo creates version management (placeholder)
func (f *TemplateFactory) VersionGo() machinery.Template {
	return SimpleFile(f.config, "internal/version/version.go", "package version")
}

// ClusterDockerfile creates cluster Dockerfile (placeholder)
func (f *TemplateFactory) ClusterDockerfile() machinery.Template {
	return SimpleFile(f.config, "cluster/images/provider/Dockerfile", "FROM alpine:latest")
}

// ClusterMakefile creates cluster Makefile (placeholder)
func (f *TemplateFactory) ClusterMakefile() machinery.Template {
	return SimpleFile(f.config, "cluster/images/provider/Makefile", "# Cluster Makefile")
}

// License creates LICENSE file (placeholder)
func (f *TemplateFactory) License() machinery.Template {
	return StaticFile(f.config, "LICENSE", "Apache License 2.0")
}

// API Template Methods - delegates to API template functions

// APITypes creates API types file for managed resource
func (f *TemplateFactory) APITypes(force bool) machinery.Template {
	return APITypes(f.config, force)
}

// APIGroup creates API group registration
func (f *TemplateFactory) APIGroup() machinery.Template {
	return APIGroup(f.config)
}

// Controller creates controller implementation
func (f *TemplateFactory) Controller(force bool) machinery.Template {
	return Controller(f.config, force)
}